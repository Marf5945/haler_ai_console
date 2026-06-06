//go:build windows

package visual_learning

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	ortAPIVersionMax = 24

	ortCreateEnv                        = 3
	ortGetErrorMessage                  = 2
	ortCreateSession                    = 7
	ortRun                              = 9
	ortCreateSessionOptions             = 10
	ortSetSessionGraphOptimizationLevel = 23
	ortCreateTensorWithDataAsOrtValue   = 49
	ortGetTensorMutableData             = 51
	ortCreateCpuMemoryInfo              = 69
)

const (
	ortLoggingLevelWarning           = 2
	ortGraphOptimizationAll          = 99
	ortArenaAllocator                = 1
	ortMemTypeDefault                = 0
	onnxTensorElementDataTypeFloat32 = 1
)

type directmlEngine struct {
	mu sync.Mutex

	dlls           []*syscall.DLL
	ortDLL         *syscall.DLL
	dmlAppendProc  uintptr
	api            []uintptr
	env            uintptr
	sessionOptions uintptr
	session        uintptr
	memoryInfo     uintptr

	inputName  string
	outputName string
	inputSize  int
	loaded     bool
	closed     bool
}

// NewInferenceEngine returns the Windows ONNX Runtime DirectML bridge.
func NewInferenceEngine() InferenceEngine {
	return &directmlEngine{
		inputName:  "images",
		outputName: "output",
		inputSize:  DefaultYOLOXButtonSConfig.InputSize,
	}
}

func (e *directmlEngine) LoadModel(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return ErrEngineAlreadyClosed
	}

	verifier, vErr := NewModelVerifier(embeddedModelHashes)
	if vErr != nil {
		return fmt.Errorf("%w: model hash manifest parse error: %v", ErrInferenceUnavailable, vErr)
	}
	if err := verifier.Verify(path); err != nil {
		return err
	}

	status := CheckDirectMLRuntime()
	if !status.Available {
		return fmt.Errorf("%w: %s", ErrInferenceUnavailable, status.Reason)
	}

	if err := e.loadRuntime(status.RuntimeDir); err != nil {
		return fmt.Errorf("%w: %v", ErrInferenceUnavailable, err)
	}
	if err := e.createEnv(); err != nil {
		return fmt.Errorf("%w: %v", ErrInferenceUnavailable, err)
	}
	if err := e.createSessionOptions(); err != nil {
		return fmt.Errorf("%w: %v", ErrInferenceUnavailable, err)
	}
	if err := e.appendDirectMLProvider(); err != nil {
		return fmt.Errorf("%w: %v", ErrInferenceUnavailable, err)
	}
	if err := e.createSession(path); err != nil {
		return fmt.Errorf("%w: %v", ErrInferenceUnavailable, err)
	}
	if err := e.createMemoryInfo(); err != nil {
		return fmt.Errorf("%w: %v", ErrInferenceUnavailable, err)
	}

	e.loaded = true
	return nil
}

func (e *directmlEngine) Infer(rgba []byte, width, height int) (RawTensor, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return RawTensor{}, ErrEngineAlreadyClosed
	}
	if !e.loaded {
		return RawTensor{}, ErrModelNotLoaded
	}
	if len(rgba) < width*height*4 || width <= 0 || height <= 0 {
		return RawTensor{}, fmt.Errorf("directml infer: invalid RGBA image %dx%d (%d bytes)", width, height, len(rgba))
	}

	input := preprocessYOLOXRGBA(rgba, width, height, e.inputSize)
	dims := []int64{1, 3, int64(e.inputSize), int64(e.inputSize)}
	var inputValue uintptr
	if err := e.callStatus(ortCreateTensorWithDataAsOrtValue,
		e.memoryInfo,
		uintptr(unsafe.Pointer(&input[0])),
		uintptr(len(input)*4),
		uintptr(unsafe.Pointer(&dims[0])),
		uintptr(len(dims)),
		onnxTensorElementDataTypeFloat32,
		uintptr(unsafe.Pointer(&inputValue)),
	); err != nil {
		return RawTensor{}, err
	}

	inputNameBytes := nulTerminated(e.inputName)
	outputNameBytes := nulTerminated(e.outputName)
	inputNames := []uintptr{uintptr(unsafe.Pointer(&inputNameBytes[0]))}
	outputNames := []uintptr{uintptr(unsafe.Pointer(&outputNameBytes[0]))}
	inputValues := []uintptr{inputValue}
	outputValues := []uintptr{0}

	if err := e.callStatus(ortRun,
		e.session,
		0,
		uintptr(unsafe.Pointer(&inputNames[0])),
		uintptr(unsafe.Pointer(&inputValues[0])),
		1,
		uintptr(unsafe.Pointer(&outputNames[0])),
		1,
		uintptr(unsafe.Pointer(&outputValues[0])),
	); err != nil {
		return RawTensor{}, err
	}
	if outputValues[0] == 0 {
		return RawTensor{}, fmt.Errorf("directml infer: OrtRun returned nil output")
	}

	var outputData uintptr
	if err := e.callStatus(ortGetTensorMutableData, outputValues[0], uintptr(unsafe.Pointer(&outputData))); err != nil {
		return RawTensor{}, err
	}
	if outputData == 0 {
		return RawTensor{}, fmt.Errorf("directml infer: output tensor has nil data")
	}

	cells := yoloxCellCount(DefaultYOLOXButtonSConfig)
	entryLen := 5 + DefaultYOLOXButtonSConfig.NumClasses
	total := cells * entryLen
	native := unsafe.Slice((*float32)(unsafe.Pointer(outputData)), total)
	copied := append([]float32(nil), native...)
	return RawTensor{Data: copied, Shape: []int{1, cells, entryLen}}, nil
}

func (e *directmlEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	// ORT release function offsets vary by API version; keep loaded DLLs for process lifetime.
	return nil
}

func InferenceBackendName() string {
	return "directml"
}

func (e *directmlEngine) loadRuntime(runtimeDir string) error {
	for _, name := range []string{
		"DirectML.dll",
		"onnxruntime_providers_shared.dll",
		"onnxruntime.dll",
	} {
		dll, err := syscall.LoadDLL(filepath.Join(runtimeDir, name))
		if err != nil {
			return fmt.Errorf("LoadLibrary(%s): %w", name, err)
		}
		e.dlls = append(e.dlls, dll)
		if strings.EqualFold(name, "onnxruntime.dll") {
			e.ortDLL = dll
		}
	}

	if e.ortDLL == nil {
		return fmt.Errorf("onnxruntime.dll was not loaded")
	}
	getAPIBase, err := e.ortDLL.FindProc("OrtGetApiBase")
	if err != nil {
		return fmt.Errorf("GetProcAddress(OrtGetApiBase): %w", err)
	}
	apiBase, _, callErr := syscall.SyscallN(getAPIBase.Addr())
	if apiBase == 0 {
		return fmt.Errorf("OrtGetApiBase returned nil: %v", callErr)
	}

	apiBaseTable := unsafe.Slice((*uintptr)(unsafe.Pointer(apiBase)), 1)
	getAPI := apiBaseTable[0]
	for version := uintptr(ortAPIVersionMax); version >= 1; version-- {
		apiPtr, _, _ := syscall.SyscallN(getAPI, version)
		if apiPtr != 0 {
			e.api = unsafe.Slice((*uintptr)(unsafe.Pointer(apiPtr)), 128)
			break
		}
		if version == 1 {
			break
		}
	}
	if len(e.api) == 0 {
		return fmt.Errorf("OrtApiBase.GetApi did not accept API versions 1..%d", ortAPIVersionMax)
	}

	appendProc, err := e.findRuntimeProc("OrtSessionOptionsAppendExecutionProvider_DML")
	if err != nil {
		return fmt.Errorf("GetProcAddress(OrtSessionOptionsAppendExecutionProvider_DML): %w", err)
	}
	e.dmlAppendProc = appendProc.Addr()
	return nil
}

func (e *directmlEngine) findRuntimeProc(name string) (*syscall.Proc, error) {
	var lastErr error
	for _, dll := range e.dlls {
		proc, err := dll.FindProc(name)
		if err == nil {
			return proc, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, syscall.ENOENT
}

func (e *directmlEngine) createEnv() error {
	logID := nulTerminated("ui-console-yolox")
	return e.callStatus(ortCreateEnv,
		ortLoggingLevelWarning,
		uintptr(unsafe.Pointer(&logID[0])),
		uintptr(unsafe.Pointer(&e.env)),
	)
}

func (e *directmlEngine) createSessionOptions() error {
	if err := e.callStatus(ortCreateSessionOptions, uintptr(unsafe.Pointer(&e.sessionOptions))); err != nil {
		return err
	}
	// Optimization level is best-effort; DirectML still works if the runtime rejects it.
	_ = e.callStatus(ortSetSessionGraphOptimizationLevel, e.sessionOptions, ortGraphOptimizationAll)
	return nil
}

func (e *directmlEngine) appendDirectMLProvider() error {
	status, _, _ := syscall.SyscallN(e.dmlAppendProc, e.sessionOptions, 0)
	return e.statusError(status)
}

func (e *directmlEngine) createSession(path string) error {
	modelPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return e.callStatus(ortCreateSession,
		e.env,
		uintptr(unsafe.Pointer(modelPath)),
		e.sessionOptions,
		uintptr(unsafe.Pointer(&e.session)),
	)
}

func (e *directmlEngine) createMemoryInfo() error {
	return e.callStatus(ortCreateCpuMemoryInfo,
		ortArenaAllocator,
		ortMemTypeDefault,
		uintptr(unsafe.Pointer(&e.memoryInfo)),
	)
}

func (e *directmlEngine) callStatus(index int, args ...uintptr) error {
	if index < 0 || index >= len(e.api) || e.api[index] == 0 {
		return fmt.Errorf("onnxruntime API function index %d is unavailable", index)
	}
	status, _, _ := syscall.SyscallN(e.api[index], args...)
	return e.statusError(status)
}

func (e *directmlEngine) statusError(status uintptr) error {
	if status == 0 {
		return nil
	}
	if len(e.api) <= ortGetErrorMessage || e.api[ortGetErrorMessage] == 0 {
		return fmt.Errorf("onnxruntime returned status 0x%x", status)
	}
	messagePtr, _, _ := syscall.SyscallN(e.api[ortGetErrorMessage], status)
	message := cString(messagePtr)
	if message == "" {
		message = fmt.Sprintf("status 0x%x", status)
	}
	return fmt.Errorf("onnxruntime: %s", message)
}

func preprocessYOLOXRGBA(rgba []byte, width, height, size int) []float32 {
	out := make([]float32, 3*size*size)
	plane := size * size
	for y := 0; y < size; y++ {
		srcY := y * height / size
		for x := 0; x < size; x++ {
			srcX := x * width / size
			src := (srcY*width + srcX) * 4
			dst := y*size + x
			// YOLO input is NCHW RGB normalized to 0..1.
			out[dst] = float32(rgba[src]) / 255.0
			out[plane+dst] = float32(rgba[src+1]) / 255.0
			out[2*plane+dst] = float32(rgba[src+2]) / 255.0
		}
	}
	return out
}

func yoloxCellCount(config YOLOConfig) int {
	total := 0
	for _, stride := range config.Strides {
		grid := config.InputSize / stride
		total += grid * grid
	}
	return total
}

func nulTerminated(s string) []byte {
	s = strings.ReplaceAll(s, "\x00", "")
	return append([]byte(s), 0)
}

func cString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	bytes := make([]byte, 0, 128)
	for p := ptr; ; p++ {
		b := *(*byte)(unsafe.Pointer(p))
		if b == 0 {
			break
		}
		bytes = append(bytes, b)
	}
	return string(bytes)
}
