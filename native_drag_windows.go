//go:build windows

package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
	"ui_console/data/subexport"
)

const (
	sOK                        uintptr = 0x00000000
	eNoInterface               uintptr = 0x80004002
	eNotImpl                   uintptr = 0x80004001
	dvEFormatEtc               uintptr = 0x80040064
	sFalse                     uintptr = 0x00000001
	dragdropSCancel            uintptr = 0x00040101
	dragdropSDrop              uintptr = 0x00040100
	dragdropSUseDefaultCursors uintptr = 0x00040102

	cfHDrop         uint16 = 15
	dvaspectContent uint32 = 1
	tymedHGlobal    uint32 = 1

	gmemMoveable uint32 = 0x0002
	gmemZeroinit uint32 = 0x0040

	dropEffectCopy uint32 = 1
	dataDirGet     uint32 = 1
	mkLButton      uint32 = 0x0001
	gaRoot         uint32 = 2
	vkLButton      int32  = 0x01
	pmNoRemove     uint32 = 0x0000
)

var (
	ole32                        = windows.NewLazySystemDLL("ole32.dll")
	user32                       = windows.NewLazySystemDLL("user32.dll")
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procOleInitialize            = ole32.NewProc("OleInitialize")
	procOleUninitialize          = ole32.NewProc("OleUninitialize")
	procDoDragDrop               = ole32.NewProc("DoDragDrop")
	procGlobalAlloc              = kernel32.NewProc("GlobalAlloc")
	procGlobalLock               = kernel32.NewProc("GlobalLock")
	procGlobalUnlock             = kernel32.NewProc("GlobalUnlock")
	procGlobalFree               = kernel32.NewProc("GlobalFree")
	procGetCurrentThreadID       = kernel32.NewProc("GetCurrentThreadId")
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")
	procGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	procRegisterClipboardFormatW = user32.NewProc("RegisterClipboardFormatW")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	procPeekMessageW             = user32.NewProc("PeekMessageW")
	procWindowFromPoint          = user32.NewProc("WindowFromPoint")
	procGetAncestor              = user32.NewProc("GetAncestor")
	procGetClassNameW            = user32.NewProc("GetClassNameW")
	iidIUnknown                  = windows.GUID{Data1: 0x00000000, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIDataObject               = windows.GUID{Data1: 0x0000010e, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIDropSource               = windows.GUID{Data1: 0x00000121, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIEnumFormatEtc            = windows.GUID{Data1: 0x00000103, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	dataObjectVtbl               = &windowsDataObjectVtbl{
		QueryInterface:        syscall.NewCallback(windowsDataObjectQueryInterface),
		AddRef:                syscall.NewCallback(windowsDataObjectAddRef),
		Release:               syscall.NewCallback(windowsDataObjectRelease),
		GetData:               syscall.NewCallback(windowsDataObjectGetData),
		GetDataHere:           syscall.NewCallback(windowsDataObjectGetDataHere),
		QueryGetData:          syscall.NewCallback(windowsDataObjectQueryGetData),
		GetCanonicalFormatEtc: syscall.NewCallback(windowsDataObjectGetCanonicalFormatEtc),
		SetData:               syscall.NewCallback(windowsDataObjectSetData),
		EnumFormatEtc:         syscall.NewCallback(windowsDataObjectEnumFormatEtc),
		DAdvise:               syscall.NewCallback(windowsDataObjectDAdvise),
		DUnadvise:             syscall.NewCallback(windowsDataObjectDUnadvise),
		EnumDAdvise:           syscall.NewCallback(windowsDataObjectEnumDAdvise),
	}
	dropSourceVtbl = &windowsDropSourceVtbl{
		QueryInterface:    syscall.NewCallback(windowsDropSourceQueryInterface),
		AddRef:            syscall.NewCallback(windowsDropSourceAddRef),
		Release:           syscall.NewCallback(windowsDropSourceRelease),
		QueryContinueDrag: syscall.NewCallback(windowsDropSourceQueryContinueDrag),
		GiveFeedback:      syscall.NewCallback(windowsDropSourceGiveFeedback),
	}
	formatEtcEnumVtbl = &windowsFormatEtcEnumVtbl{
		QueryInterface: syscall.NewCallback(windowsFormatEtcEnumQueryInterface),
		AddRef:         syscall.NewCallback(windowsFormatEtcEnumAddRef),
		Release:        syscall.NewCallback(windowsFormatEtcEnumRelease),
		Next:           syscall.NewCallback(windowsFormatEtcEnumNext),
		Skip:           syscall.NewCallback(windowsFormatEtcEnumSkip),
		Reset:          syscall.NewCallback(windowsFormatEtcEnumReset),
		Clone:          syscall.NewCallback(windowsFormatEtcEnumClone),
	}
	windowsNativeDragActive      int32
	windowsNativeDragUnavailable int32
	preferredDropEffectOnce      sync.Once
	preferredDropEffectFormat    uint16
	formatEtcEnumRefs            sync.Map
)

type windowsPoint struct {
	X int32
	Y int32
}

type windowsMessage struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      windowsPoint
}

type windowsFormatEtc struct {
	CfFormat uint16
	Ptd      uintptr
	DwAspect uint32
	Lindex   int32
	Tymed    uint32
}

type windowsStgMedium struct {
	Tymed          uint32
	_              uint32
	HGlobal        uintptr
	PUnkForRelease uintptr
}

type windowsDropFiles struct {
	PFiles uint32
	Pt     windowsPoint
	FNC    int32
	FWide  int32
}

type windowsDataObjectVtbl struct {
	QueryInterface        uintptr
	AddRef                uintptr
	Release               uintptr
	GetData               uintptr
	GetDataHere           uintptr
	QueryGetData          uintptr
	GetCanonicalFormatEtc uintptr
	SetData               uintptr
	EnumFormatEtc         uintptr
	DAdvise               uintptr
	DUnadvise             uintptr
	EnumDAdvise           uintptr
}

type windowsDataObject struct {
	Vtbl *windowsDataObjectVtbl
	Ref  int32
	Path string
}

type windowsFormatEtcEnumVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Next           uintptr
	Skip           uintptr
	Reset          uintptr
	Clone          uintptr
}

type windowsFormatEtcEnum struct {
	Vtbl    *windowsFormatEtcEnumVtbl
	Ref     int32
	Index   int32
	Formats [2]windowsFormatEtc
	Count   int32
}

type windowsDropSourceVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	QueryContinueDrag uintptr
	GiveFeedback      uintptr
}

type windowsDropSource struct {
	Vtbl            *windowsDropSourceVtbl
	Ref             int32
	DropPoint       windowsPoint
	HasPoint        int32
	QueryTraceCount int32
}

type windowsDropTarget struct {
	Kind      string
	Directory string
}

type windowsDragCallResult struct {
	hr      uintptr
	effect  uint32
	source  *windowsDropSource
	started time.Time
}

func startNativeFileDrag(path string) nativeDragResult {
	writeNativeDragPhase("windows-start", fmt.Sprintf("path=%q", path))
	info, err := os.Stat(path)
	if err != nil {
		writeNativeDragPhase("windows-source-missing", err.Error())
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: fmt.Sprintf("Windows native drag source missing: %v", err)}
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		writeNativeDragPhase("windows-source-invalid", fmt.Sprintf("mode=%s", info.Mode()))
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: "Windows native drag source is not a file or folder"}
	}
	if !atomic.CompareAndSwapInt32(&windowsNativeDragActive, 0, 1) {
		writeNativeDragPhase("windows-dodragdrop-busy", "")
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: "Windows native drag is already stuck or running"}
	}

	done := make(chan windowsDragCallResult, 1)
	source := &windowsDropSource{Vtbl: dropSourceVtbl, Ref: 1}
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		writeNativeDragPhase("windows-thread-locked", "")

		if hr, _, _ := procOleInitialize.Call(0); failedHRESULT(hr) {
			writeNativeDragPhase("windows-ole-init-failed", fmt.Sprintf("hr=0x%08x", uint32(hr)))
			done <- windowsDragCallResult{hr: hr, source: source, started: time.Now()}
			atomic.StoreInt32(&windowsNativeDragActive, 0)
			return
		}
		defer procOleUninitialize.Call()
		writeNativeDragPhase("windows-ole-init-ok", "")

		if !windowsLeftButtonDown() {
			writeNativeDragPhase("windows-left-button-up-before-drag", "")
			done <- windowsDragCallResult{hr: dragdropSCancel, source: source, started: time.Now()}
			atomic.StoreInt32(&windowsNativeDragActive, 0)
			return
		}

		detachInput := prepareWindowsDragThreadInput()
		defer detachInput()

		data := &windowsDataObject{Vtbl: dataObjectVtbl, Ref: 1, Path: path}
		var effect uint32
		started := time.Now()
		// DoDragDrop takes over the active mouse drag and gives Explorer CF_HDROP.
		writeNativeDragPhase("windows-dodragdrop-start", fmt.Sprintf("path=%q", path))
		hr, _, _ := procDoDragDrop.Call(
			uintptr(unsafe.Pointer(data)),
			uintptr(unsafe.Pointer(source)),
			uintptr(dropEffectCopy),
			uintptr(unsafe.Pointer(&effect)),
		)
		runtime.KeepAlive(data)
		runtime.KeepAlive(source)
		writeNativeDragPhase("windows-dodragdrop-return", fmt.Sprintf("hr=0x%08x effect=%d", uint32(hr), effect))
		done <- windowsDragCallResult{hr: hr, effect: effect, source: source, started: started}
		atomic.StoreInt32(&windowsNativeDragActive, 0)
	}()

	var call windowsDragCallResult
	select {
	case call = <-done:
	case <-time.After(1200 * time.Millisecond):
		if atomic.LoadInt32(&source.QueryTraceCount) == 0 {
			writeNativeDragPhase("windows-dodragdrop-no-callback", "")
			atomic.StoreInt32(&windowsNativeDragActive, 0)
			return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: "Windows OLE DoDragDrop did not enter drag callbacks; no drop was completed"}
		}
		call = <-done
	}

	hr := call.hr
	effect := call.effect

	if hr == dragdropSCancel || effect == 0 {
		return nativeDragResult{Status: nativeDragStatusCancelled, FallbackRequired: false, Message: "Windows native drag cancelled"}
	}
	if failedHRESULT(hr) {
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: fmt.Sprintf("DoDragDrop failed: 0x%08x", uint32(hr))}
	}

	target := resolveWindowsDropTarget(call.source)
	if target.Directory == "" {
		writeNativeDragPhase("windows-target-empty", "")
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: "Windows native drag finished, but the drop folder could not be resolved"}
	}
	writeNativeDragPhase("windows-target", fmt.Sprintf("kind=%s dir=%q", target.Kind, target.Directory))
	landedPath := filepath.Join(target.Directory, filepath.Base(path))
	landedPath = waitForWindowsLandedPath(landedPath, filepath.Base(path), call.started)
	if landedPath == "" {
		writeNativeDragPhase("windows-landed-missing", fmt.Sprintf("dir=%q base=%q", target.Directory, filepath.Base(path)))
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: fmt.Sprintf("Windows native drag finished, but copied item was not found in %s", target.Directory)}
	}
	writeNativeDragPhase("windows-landed", fmt.Sprintf("path=%q", landedPath))

	return nativeDragResult{
		Status:           nativeDragStatusSuccess,
		FallbackRequired: false,
		Message:          fmt.Sprintf("Windows native CF_HDROP drag completed to %s", target.Kind),
		LandedPath:       landedPath,
		DropTargetKind:   target.Kind,
		DropTargetDir:    target.Directory,
	}
}

func windowsLeftButtonDown() bool {
	state, _, _ := procGetAsyncKeyState.Call(uintptr(vkLButton))
	return state&0x8000 != 0
}

func prepareWindowsDragThreadInput() func() {
	// Ensure this STA thread owns a message queue before OLE starts its modal
	// drag loop. Without this, DoDragDrop can sit without QueryContinueDrag.
	var msg windowsMessage
	procPeekMessageW.Call(
		uintptr(unsafe.Pointer(&msg)),
		0,
		0,
		0,
		uintptr(pmNoRemove),
	)

	currentThread, _, _ := procGetCurrentThreadID.Call()
	foreground, _, _ := procGetForegroundWindow.Call()
	if foreground == 0 || currentThread == 0 {
		writeNativeDragPhase("windows-input-thread-none", fmt.Sprintf("current=%d foreground=0x%x", currentThread, foreground))
		return func() {}
	}
	var pid uint32
	foregroundThread, _, _ := procGetWindowThreadProcessID.Call(
		foreground,
		uintptr(unsafe.Pointer(&pid)),
	)
	if foregroundThread == 0 || foregroundThread == currentThread {
		writeNativeDragPhase("windows-input-thread-same", fmt.Sprintf("thread=%d hwnd=0x%x", currentThread, foreground))
		return func() {}
	}
	attached, _, err := procAttachThreadInput.Call(currentThread, foregroundThread, 1)
	if attached == 0 {
		writeNativeDragPhase("windows-input-attach-failed", fmt.Sprintf("current=%d foregroundThread=%d err=%v", currentThread, foregroundThread, err))
		return func() {}
	}
	writeNativeDragPhase("windows-input-attached", fmt.Sprintf("current=%d foregroundThread=%d hwnd=0x%x", currentThread, foregroundThread, foreground))
	return func() {
		procAttachThreadInput.Call(currentThread, foregroundThread, 0)
		writeNativeDragPhase("windows-input-detached", fmt.Sprintf("current=%d foregroundThread=%d", currentThread, foregroundThread))
	}
}

func windowsDataObjectQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	if ppv == 0 {
		return eNoInterface
	}
	guid := (*windows.GUID)(unsafe.Pointer(riid))
	if equalGUID(guid, &iidIUnknown) || equalGUID(guid, &iidIDataObject) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		windowsDataObjectAddRef(this)
		return sOK
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	return eNoInterface
}

func windowsDataObjectAddRef(this uintptr) uintptr {
	obj := (*windowsDataObject)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&obj.Ref, 1))
}

func windowsDataObjectRelease(this uintptr) uintptr {
	obj := (*windowsDataObject)(unsafe.Pointer(this))
	next := atomic.AddInt32(&obj.Ref, -1)
	return uintptr(next)
}

func windowsDataObjectGetData(this uintptr, formatEtc uintptr, medium uintptr) uintptr {
	obj := (*windowsDataObject)(unsafe.Pointer(this))
	fe := (*windowsFormatEtc)(unsafe.Pointer(formatEtc))
	if isPreferredDropEffectFormat(fe) {
		hglobal, err := createDropEffectGlobal(dropEffectCopy)
		if err != nil {
			writeNativeDragPhase("windows-getdata-drop-effect-error", err.Error())
			return dvEFormatEtc
		}
		stg := (*windowsStgMedium)(unsafe.Pointer(medium))
		stg.Tymed = tymedHGlobal
		stg.HGlobal = hglobal
		stg.PUnkForRelease = 0
		writeNativeDragPhase("windows-getdata-drop-effect", "copy")
		return sOK
	}
	if !isHDropFormat(fe) {
		return dvEFormatEtc
	}
	hglobal, err := createHDropGlobal(obj.Path)
	if err != nil {
		writeNativeDragPhase("windows-getdata-hdrop-error", err.Error())
		return dvEFormatEtc
	}
	stg := (*windowsStgMedium)(unsafe.Pointer(medium))
	stg.Tymed = tymedHGlobal
	stg.HGlobal = hglobal
	stg.PUnkForRelease = 0
	writeNativeDragPhase("windows-getdata-hdrop", fmt.Sprintf("path=%q", obj.Path))
	return sOK
}

func windowsDataObjectGetDataHere(this uintptr, formatEtc uintptr, medium uintptr) uintptr {
	return eNotImpl
}

func windowsDataObjectQueryGetData(this uintptr, formatEtc uintptr) uintptr {
	fe := (*windowsFormatEtc)(unsafe.Pointer(formatEtc))
	if isHDropFormat(fe) || isPreferredDropEffectFormat(fe) {
		return sOK
	}
	return dvEFormatEtc
}

func windowsDataObjectGetCanonicalFormatEtc(this uintptr, in uintptr, out uintptr) uintptr {
	return eNotImpl
}

func windowsDataObjectSetData(this uintptr, formatEtc uintptr, medium uintptr, release uintptr) uintptr {
	return eNotImpl
}

func windowsDataObjectEnumFormatEtc(this uintptr, direction uintptr, enum uintptr) uintptr {
	if enum == 0 || uint32(direction) != dataDirGet {
		return eNotImpl
	}
	out := newWindowsFormatEtcEnum()
	*(*uintptr)(unsafe.Pointer(enum)) = uintptr(unsafe.Pointer(out))
	writeNativeDragPhase("windows-enum-formatetc", fmt.Sprintf("count=%d", out.Count))
	return sOK
}

func windowsDataObjectDAdvise(this uintptr, formatEtc uintptr, flags uintptr, sink uintptr, connection uintptr) uintptr {
	return eNotImpl
}

func windowsDataObjectDUnadvise(this uintptr, connection uintptr) uintptr {
	return eNotImpl
}

func windowsDataObjectEnumDAdvise(this uintptr, enum uintptr) uintptr {
	return eNotImpl
}

func windowsFormatEtcEnumQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	if ppv == 0 {
		return eNoInterface
	}
	guid := (*windows.GUID)(unsafe.Pointer(riid))
	if equalGUID(guid, &iidIUnknown) || equalGUID(guid, &iidIEnumFormatEtc) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		windowsFormatEtcEnumAddRef(this)
		return sOK
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	return eNoInterface
}

func windowsFormatEtcEnumAddRef(this uintptr) uintptr {
	enum := (*windowsFormatEtcEnum)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&enum.Ref, 1))
}

func windowsFormatEtcEnumRelease(this uintptr) uintptr {
	enum := (*windowsFormatEtcEnum)(unsafe.Pointer(this))
	next := atomic.AddInt32(&enum.Ref, -1)
	if next <= 0 {
		formatEtcEnumRefs.Delete(this)
	}
	return uintptr(next)
}

func windowsFormatEtcEnumNext(this uintptr, celt uintptr, rgelt uintptr, pceltFetched uintptr) uintptr {
	enum := (*windowsFormatEtcEnum)(unsafe.Pointer(this))
	if rgelt == 0 {
		return eNoInterface
	}
	fetched := uintptr(0)
	for fetched < celt && atomic.LoadInt32(&enum.Index) < enum.Count {
		index := atomic.LoadInt32(&enum.Index)
		target := (*windowsFormatEtc)(unsafe.Pointer(rgelt + fetched*unsafe.Sizeof(windowsFormatEtc{})))
		*target = enum.Formats[index]
		atomic.AddInt32(&enum.Index, 1)
		fetched++
	}
	if pceltFetched != 0 {
		*(*uintptr)(unsafe.Pointer(pceltFetched)) = fetched
	}
	if fetched == celt {
		return sOK
	}
	return sFalse
}

func windowsFormatEtcEnumSkip(this uintptr, celt uintptr) uintptr {
	enum := (*windowsFormatEtcEnum)(unsafe.Pointer(this))
	next := atomic.AddInt32(&enum.Index, int32(celt))
	if next > enum.Count {
		atomic.StoreInt32(&enum.Index, enum.Count)
		return sFalse
	}
	return sOK
}

func windowsFormatEtcEnumReset(this uintptr) uintptr {
	enum := (*windowsFormatEtcEnum)(unsafe.Pointer(this))
	atomic.StoreInt32(&enum.Index, 0)
	return sOK
}

func windowsFormatEtcEnumClone(this uintptr, out uintptr) uintptr {
	return eNotImpl
}

func windowsDropSourceQueryInterface(this uintptr, riid uintptr, ppv uintptr) uintptr {
	if ppv == 0 {
		return eNoInterface
	}
	guid := (*windows.GUID)(unsafe.Pointer(riid))
	if equalGUID(guid, &iidIUnknown) || equalGUID(guid, &iidIDropSource) {
		*(*uintptr)(unsafe.Pointer(ppv)) = this
		windowsDropSourceAddRef(this)
		return sOK
	}
	*(*uintptr)(unsafe.Pointer(ppv)) = 0
	return eNoInterface
}

func windowsDropSourceAddRef(this uintptr) uintptr {
	source := (*windowsDropSource)(unsafe.Pointer(this))
	return uintptr(atomic.AddInt32(&source.Ref, 1))
}

func windowsDropSourceRelease(this uintptr) uintptr {
	source := (*windowsDropSource)(unsafe.Pointer(this))
	next := atomic.AddInt32(&source.Ref, -1)
	return uintptr(next)
}

func windowsDropSourceQueryContinueDrag(this uintptr, escapePressed uintptr, keyState uintptr) uintptr {
	source := (*windowsDropSource)(unsafe.Pointer(this))
	traceCount := atomic.AddInt32(&source.QueryTraceCount, 1)
	if traceCount <= 5 || traceCount%30 == 0 {
		writeNativeDragPhase("windows-query-continue", fmt.Sprintf("count=%d escape=%d keyState=0x%04x", traceCount, escapePressed, uint32(keyState)))
	}
	if escapePressed != 0 {
		return dragdropSCancel
	}
	if uint32(keyState)&mkLButton == 0 {
		if point, ok := getCursorPoint(); ok {
			source.DropPoint = point
			atomic.StoreInt32(&source.HasPoint, 1)
		}
		return dragdropSDrop
	}
	return sOK
}

func windowsDropSourceGiveFeedback(this uintptr, effect uintptr) uintptr {
	return dragdropSUseDefaultCursors
}

func isHDropFormat(fe *windowsFormatEtc) bool {
	return fe != nil &&
		fe.CfFormat == cfHDrop &&
		fe.DwAspect == dvaspectContent &&
		(fe.Tymed&tymedHGlobal) != 0
}

func isPreferredDropEffectFormat(fe *windowsFormatEtc) bool {
	return fe != nil &&
		fe.CfFormat == getPreferredDropEffectFormat() &&
		fe.DwAspect == dvaspectContent &&
		(fe.Tymed&tymedHGlobal) != 0
}

func getPreferredDropEffectFormat() uint16 {
	preferredDropEffectOnce.Do(func() {
		name, _ := windows.UTF16PtrFromString("Preferred DropEffect")
		format, _, _ := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(name)))
		preferredDropEffectFormat = uint16(format)
		writeNativeDragPhase("windows-register-preferred-drop-effect", fmt.Sprintf("format=%d", preferredDropEffectFormat))
	})
	return preferredDropEffectFormat
}

func newWindowsFormatEtcEnum() *windowsFormatEtcEnum {
	enum := &windowsFormatEtcEnum{
		Vtbl:  formatEtcEnumVtbl,
		Ref:   1,
		Count: 2,
	}
	enum.Formats[0] = windowsFormatEtc{
		CfFormat: cfHDrop,
		DwAspect: dvaspectContent,
		Lindex:   -1,
		Tymed:    tymedHGlobal,
	}
	enum.Formats[1] = windowsFormatEtc{
		CfFormat: getPreferredDropEffectFormat(),
		DwAspect: dvaspectContent,
		Lindex:   -1,
		Tymed:    tymedHGlobal,
	}
	formatEtcEnumRefs.Store(uintptr(unsafe.Pointer(enum)), enum)
	return enum
}

func createDropEffectGlobal(effect uint32) (uintptr, error) {
	hglobal, _, _ := procGlobalAlloc.Call(uintptr(gmemMoveable|gmemZeroinit), unsafe.Sizeof(effect))
	if hglobal == 0 {
		return 0, fmt.Errorf("GlobalAlloc failed")
	}
	ptr, _, _ := procGlobalLock.Call(hglobal)
	if ptr == 0 {
		procGlobalFree.Call(hglobal)
		return 0, fmt.Errorf("GlobalLock failed")
	}
	*(*uint32)(unsafe.Pointer(ptr)) = effect
	procGlobalUnlock.Call(hglobal)
	return hglobal, nil
}

func createHDropGlobal(path string) (uintptr, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	// CF_HDROP is the file/folder payload format accepted by Explorer.
	utf16, err := windows.UTF16FromString(abs)
	if err != nil {
		return 0, err
	}
	headerSize := int(unsafe.Sizeof(windowsDropFiles{}))
	bytesLen := headerSize + (len(utf16)+1)*2
	hglobal, _, _ := procGlobalAlloc.Call(uintptr(gmemMoveable|gmemZeroinit), uintptr(bytesLen))
	if hglobal == 0 {
		return 0, fmt.Errorf("GlobalAlloc failed")
	}
	ptr, _, _ := procGlobalLock.Call(hglobal)
	if ptr == 0 {
		procGlobalFree.Call(hglobal)
		return 0, fmt.Errorf("GlobalLock failed")
	}

	drop := (*windowsDropFiles)(unsafe.Pointer(ptr))
	drop.PFiles = uint32(headerSize)
	drop.FWide = 1
	dst := unsafe.Pointer(ptr + uintptr(headerSize))
	copy(unsafe.Slice((*uint16)(dst), len(utf16)), utf16)
	procGlobalUnlock.Call(hglobal)
	return hglobal, nil
}

func getCursorPoint() (windowsPoint, bool) {
	var point windowsPoint
	ok, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	return point, ok != 0
}

func resolveWindowsDropTarget(source *windowsDropSource) windowsDropTarget {
	if source == nil || atomic.LoadInt32(&source.HasPoint) == 0 {
		return windowsDropTarget{}
	}
	point := source.DropPoint
	hwnd, _, _ := procWindowFromPoint.Call(uintptr(point.X), uintptr(point.Y))
	if hwnd == 0 {
		return windowsDropTarget{}
	}
	root, _, _ := procGetAncestor.Call(hwnd, uintptr(gaRoot))
	if root == 0 {
		root = hwnd
	}
	// Desktop icons live under several shell window classes.
	if isDesktopWindow(root) || isDesktopWindow(hwnd) {
		return windowsDropTarget{Kind: "desktop", Directory: windowsDesktopDirectory()}
	}
	// Explorer exposes its active folder through Shell.Application.
	if dir := explorerDirectoryForHWND(root); dir != "" {
		return windowsDropTarget{Kind: "explorer", Directory: dir}
	}
	return windowsDropTarget{}
}

func isDesktopWindow(hwnd uintptr) bool {
	className := windowsClassName(hwnd)
	return className == "Progman" || className == "WorkerW" || className == "SHELLDLL_DefView" || className == "SysListView32"
}

func windowsClassName(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}

func windowsDesktopDirectory() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "Desktop")
}

func copyWindowsDragFallback(path, reason string) nativeDragResult {
	desktop := windowsDesktopDirectory()
	if desktop == "" {
		writeNativeDragPhase("windows-fallback-no-desktop", "")
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "; desktop path could not be resolved"}
	}
	target := filepath.Join(desktop, filepath.Base(path))
	if _, err := os.Stat(target); err == nil {
		writeNativeDragPhase("windows-fallback-target-exists", fmt.Sprintf("target=%q", target))
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "; fallback target already exists on Desktop"}
	}
	info, err := os.Stat(path)
	if err != nil {
		writeNativeDragPhase("windows-fallback-source-missing", err.Error())
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "; source missing"}
	}
	if info.IsDir() {
		err = copySubExportDirectory(path, target)
	} else {
		err = copySubExportFile(path, target, info.Mode())
	}
	if err != nil {
		writeNativeDragPhase("windows-fallback-copy-error", err.Error())
		return nativeDragResult{Status: nativeDragStatusFailed, FallbackRequired: true, Message: reason + "; fallback copy failed: " + err.Error()}
	}
	writeNativeDragPhase("windows-fallback-copied", fmt.Sprintf("target=%q", target))
	return nativeDragResult{
		Status:           nativeDragStatusSuccess,
		FallbackRequired: true,
		Message:          reason,
		LandedPath:       target,
		DropTargetKind:   "desktop-fallback",
		DropTargetDir:    desktop,
	}
}

func explorerDirectoryForHWND(root uintptr) string {
	shellObj, err := createDispatch("Shell.Application")
	if err != nil {
		return ""
	}
	defer shellObj.Release()
	windowsVariant, err := oleCall(shellObj, "Windows")
	if err != nil {
		return ""
	}
	windowsDisp := windowsVariant.ToIDispatch()
	if windowsDisp == nil {
		return ""
	}
	defer windowsDisp.Release()
	countVar, err := oleGet(windowsDisp, "Count")
	if err != nil {
		return ""
	}
	count := int(countVar.Val)
	for i := 0; i < count; i++ {
		itemVar, err := oleCall(windowsDisp, "Item", i)
		if err != nil {
			continue
		}
		itemDisp := itemVar.ToIDispatch()
		if itemDisp == nil {
			continue
		}
		hwndVar, err := oleGet(itemDisp, "HWND")
		if err != nil {
			itemDisp.Release()
			continue
		}
		if uintptr(hwndVar.Val) == root {
			urlVar, err := oleGet(itemDisp, "LocationURL")
			itemDisp.Release()
			if err != nil {
				continue
			}
			if dir := windowsPathFromFileURL(urlVar.ToString()); dir != "" {
				return dir
			}
		} else {
			itemDisp.Release()
		}
	}
	return ""
}

func createDispatch(progID string) (*ole.IDispatch, error) {
	unknown, err := oleutil.CreateObject(progID)
	if err != nil {
		return nil, err
	}
	defer unknown.Release()
	dispatch, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	return dispatch, nil
}

func oleCall(dispatch *ole.IDispatch, name string, params ...interface{}) (*ole.VARIANT, error) {
	if dispatch == nil {
		return nil, fmt.Errorf("nil dispatch")
	}
	return oleutil.CallMethod(dispatch, name, params...)
}

func oleGet(dispatch *ole.IDispatch, name string, params ...interface{}) (*ole.VARIANT, error) {
	if dispatch == nil {
		return nil, fmt.Errorf("nil dispatch")
	}
	return oleutil.GetProperty(dispatch, name, params...)
}

func windowsPathFromFileURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(strings.ToLower(value), "file:") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.Host != "" {
		path, _ := url.PathUnescape(parsed.EscapedPath())
		return `\\` + parsed.Host + filepath.FromSlash(path)
	}
	path, _ := url.PathUnescape(parsed.EscapedPath())
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

func waitForWindowsLandedPath(expectedPath, baseName string, started time.Time) string {
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(expectedPath); err == nil {
			return expectedPath
		}
		if candidate := findRecentWindowsLandedManifest(filepath.Dir(expectedPath), baseName, started); candidate != "" {
			return candidate
		}
		time.Sleep(100 * time.Millisecond)
	}
	dir := filepath.Dir(expectedPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !strings.HasPrefix(strings.ToLower(entry.Name()), strings.ToLower(baseName)) {
			continue
		}
		candidate := filepath.Join(dir, entry.Name())
		info, err := os.Stat(candidate)
		if err == nil && info.ModTime().After(started.Add(-2*time.Second)) {
			return candidate
		}
	}
	return ""
}

func findRecentWindowsLandedManifest(dir, expectedSystemCode string, started time.Time) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(dir, entry.Name())
		info, err := os.Stat(candidate)
		if err != nil || info.ModTime().Before(started.Add(-2*time.Second)) {
			continue
		}
		manifest, err := subexport.LoadManifest(candidate)
		if err != nil {
			continue
		}
		if manifest.ExportType == "sub_handler" && manifest.SourceSystemCode == expectedSystemCode {
			writeNativeDragPhase("windows-landed-manifest-match", fmt.Sprintf("path=%q", candidate))
			return candidate
		}
	}
	return ""
}

func failedHRESULT(hr uintptr) bool {
	return uint32(hr)&0x80000000 != 0
}

func equalGUID(a, b *windows.GUID) bool {
	return a != nil && b != nil && *a == *b
}
