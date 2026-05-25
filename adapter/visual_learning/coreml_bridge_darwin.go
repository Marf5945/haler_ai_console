// visual_learning/coreml_bridge_darwin.go — macOS CoreML 推論引擎。
//
// Build tag: 僅在 macOS 編譯。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.1 + §14.6.2 + §14.6.3
//
// 架構：
//   Go (本檔案) ↔ cgo FFI ↔ Objective-C (coreml_bridge_darwin.m) ↔ CoreML
//
// 硬規範遵循：
//   §14.6.1 — Objective-C 在獨立 .m 檔案（coreml_bridge_darwin.m），
//              不 inline 進本 .go 檔案。
//   §14.6.2 — 所有 CoreML C API 呼叫在 runtime.LockOSThread() 的
//              專屬 worker goroutine 上序列化執行。
//   §14.6.3 — unsafe 操作（C pointer → Go slice 複製）只在本檔案內部。
//
// 失敗處理（使用者原則 #3）：
//   CoreML 相關失敗（framework 不可用、模型格式錯誤、推論失敗）
//   一律 wrap ErrInferenceUnavailable，讓系統降級到 OpenCV-only pipeline。
//
// 模型載入（使用者原則 #2）：
//   模型從檔案系統外掛載入（.mlmodelc 目錄），不 embed 進 binary。
//
// Worker 架構：
//   NewInferenceEngine() 啟動一個 LockOSThread 的 worker goroutine。
//   LoadModel / Infer / Close 透過 channel 發送請求到 worker，
//   worker 在同一 OS thread 上序列處理所有 CoreML 操作。
//   這確保 Apple framework 的 thread-safety 要求被滿足。

//go:build darwin

package visual_learning

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework CoreML -framework Foundation -framework Vision -framework CoreGraphics
#include "coreml_bridge_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

// ──────────────────────────────────────────────
// Worker 請求/回應型別
// ──────────────────────────────────────────────

// coremlReqKind 區分不同的 worker 請求類型。
type coremlReqKind int

const (
	coremlReqLoad  coremlReqKind = iota // 載入模型
	coremlReqInfer                      // 執行推論
	coremlReqClose                      // 釋放資源並結束 worker
)

// coremlRequest 是發送給 worker goroutine 的請求。
// 每個請求攜帶專屬的 resp channel，確保 caller 能收到對應的回應。
type coremlRequest struct {
	kind      coremlReqKind
	modelPath string             // coremlReqLoad: .mlmodelc 目錄完整路徑
	rgba      []byte             // coremlReqInfer: RGBA 影像原始位元組
	width     int                // coremlReqInfer: 影像寬度（pixels）
	height    int                // coremlReqInfer: 影像高度（pixels）
	resp      chan coremlResponse // caller 等待此 channel 接收回應
}

// coremlResponse 是 worker goroutine 的回應。
type coremlResponse struct {
	tensor RawTensor // 推論結果（僅 coremlReqInfer 有值）
	err    error     // 錯誤（nil 表示成功）
}

// ──────────────────────────────────────────────
// coremlEngine — InferenceEngine 實作
// ──────────────────────────────────────────────

// coremlEngine 是 macOS CoreML 推論引擎的正式實作。
//
// 內部持有一個 LockOSThread 的 worker goroutine（§14.6.2），
// 所有 CoreML C API 呼叫都在該 goroutine 上序列化執行。
//
// 生命週期：
//
//	NewInferenceEngine() → 啟動 worker
//	LoadModel()          → 透過 channel 請求 worker 載入模型
//	Infer()              → 透過 channel 請求 worker 推論
//	Close()              → 透過 channel 請求 worker 釋放資源並結束
type coremlEngine struct {
	reqCh  chan coremlRequest // 發送請求到 worker
	wg     sync.WaitGroup    // 等待 worker goroutine 結束
	mu     sync.Mutex        // 保護 closed / loaded 狀態
	closed bool              // 引擎是否已關閉
	loaded bool              // 模型是否已成功載入
}

// NewInferenceEngine 建立 macOS CoreML 推論引擎。
//
// 內部啟動一個 LockOSThread 的 worker goroutine（§14.6.2）。
// worker 在整個引擎生命週期內保持活躍，直到 Close() 被呼叫。
//
// 呼叫端應檢查 LoadModel() 的 error 來決定是否降級到 OpenCV-only。
// 使用完畢後必須呼叫 Close() 釋放 native 資源。
func NewInferenceEngine() InferenceEngine {
	e := &coremlEngine{
		reqCh: make(chan coremlRequest),
	}
	e.wg.Add(1)
	go e.worker()
	return e
}

// ──────────────────────────────────────────────
// Worker Goroutine
// ──────────────────────────────────────────────

// worker 是 CoreML 專屬的 worker goroutine。
//
// §14.6.2：runtime.LockOSThread() 確保所有 CoreML 操作在同一 OS thread 上執行。
// Apple 的 CoreML / Vision framework 對 thread 有特定要求：
//   - MLModel prediction 應在同一 thread 上呼叫以避免競爭
//   - VNImageRequestHandler 不是 thread-safe
//
// worker 持有 CoreMLHandle（不透明指標指向 Obj-C 的 CoreMLSession），
// 在收到 coremlReqClose 時釋放並結束。
func (e *coremlEngine) worker() {
	defer e.wg.Done()

	// §14.6.2：鎖定 OS thread，整個 worker 生命週期不釋放。
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var handle C.CoreMLHandle // CoreML session handle（nil = 未載入）

	for req := range e.reqCh {
		switch req.kind {

		case coremlReqLoad:
			e.handleLoad(req, &handle)

		case coremlReqInfer:
			e.handleInfer(req, handle)

		case coremlReqClose:
			// 釋放 CoreML 資源
			if handle != nil {
				C.CoreML_Close(handle)
				handle = nil
			}
			req.resp <- coremlResponse{}
			return // 結束 worker goroutine
		}
	}
}

// handleLoad 處理模型載入請求。
// 在 worker goroutine 內呼叫，已持有 LockOSThread。
//
// §14.6.3：C.CString / C.free / unsafe.Pointer 操作限制在本 bridge 檔案內。
func (e *coremlEngine) handleLoad(req coremlRequest, handle *C.CoreMLHandle) {
	// 將 Go string 轉為 C string
	cPath := C.CString(req.modelPath)
	var cErrMsg *C.char

	// 呼叫 Obj-C 端載入模型
	*handle = C.CoreML_LoadModel(cPath, &cErrMsg)

	// 立即釋放 C string（不用 defer，因為在 for-loop 內 defer 會延遲到函式結束）
	C.free(unsafe.Pointer(cPath))

	if *handle == nil {
		// CoreML 載入失敗 → wrap ErrInferenceUnavailable（使用者原則 #3）
		reason := "unknown CoreML error"
		if cErrMsg != nil {
			reason = C.GoString(cErrMsg)
			C.CoreML_FreeString(cErrMsg)
		}
		req.resp <- coremlResponse{
			err: fmt.Errorf("%w: CoreML model load failed — %s", ErrInferenceUnavailable, reason),
		}
		return
	}

	req.resp <- coremlResponse{} // 成功
}

// handleInfer 處理推論請求。
// 在 worker goroutine 內呼叫，已持有 LockOSThread。
//
// §14.6.3：所有 unsafe 操作（C pointer → Go slice 複製）在此函式內完成。
// 回傳的 RawTensor 完全在 Go heap 上，外部不持有任何 native pointer。
func (e *coremlEngine) handleInfer(req coremlRequest, handle C.CoreMLHandle) {
	// 前置檢查
	if handle == nil {
		req.resp <- coremlResponse{err: ErrModelNotLoaded}
		return
	}
	if len(req.rgba) == 0 {
		req.resp <- coremlResponse{
			err: fmt.Errorf("%w: empty RGBA image data", ErrInferenceUnavailable),
		}
		return
	}

	// ── 呼叫 CoreML 推論（C API）──
	var outData *C.float
	var outCount C.int
	var outShape *C.int
	var outDims C.int
	var cErrMsg *C.char

	rc := C.CoreML_Infer(
		handle,
		(*C.uint8_t)(unsafe.Pointer(&req.rgba[0])),
		C.int(req.width),
		C.int(req.height),
		&outData, &outCount,
		&outShape, &outDims,
		&cErrMsg,
	)

	if rc != 0 {
		// 推論失敗 → wrap ErrInferenceUnavailable（使用者原則 #3）
		reason := "unknown CoreML error"
		if cErrMsg != nil {
			reason = C.GoString(cErrMsg)
			C.CoreML_FreeString(cErrMsg)
		}
		req.resp <- coremlResponse{
			err: fmt.Errorf("%w: CoreML inference failed — %s", ErrInferenceUnavailable, reason),
		}
		return
	}

	// ── 從 C heap 複製到 Go heap（§14.6.3）──
	// 複製完成後立即釋放 C 端記憶體，確保不洩漏。
	count := int(outCount)
	dims := int(outDims)

	// 複製 float32 推論資料
	goData := make([]float32, count)
	if count > 0 {
		// unsafe.Slice 從 C pointer 建立 Go slice view（Go 1.17+），
		// 然後用 copy 將資料複製到 Go heap。
		cSlice := unsafe.Slice((*float32)(unsafe.Pointer(outData)), count)
		copy(goData, cSlice)
	}
	C.CoreML_FreeFloats(outData)

	// 複製 tensor shape 資料
	goShape := make([]int, dims)
	if dims > 0 {
		cShapeSlice := unsafe.Slice(outShape, dims)
		for i := 0; i < dims; i++ {
			goShape[i] = int(cShapeSlice[i])
		}
	}
	C.CoreML_FreeInts(outShape)

	// 回傳完全在 Go heap 上的 RawTensor
	req.resp <- coremlResponse{
		tensor: RawTensor{Data: goData, Shape: goShape},
	}
}

// ──────────────────────────────────────────────
// InferenceEngine 介面實作
// ──────────────────────────────────────────────

// LoadModel 載入 CoreML 模型（.mlmodelc 目錄格式）。
//
// 流程：
//  1. 檢查引擎狀態（已關閉 → ErrEngineAlreadyClosed）
//  2. TODO(§14.6.6): ModelVerifier.Verify() SHA256 驗證
//     — 待 model_hashes.json manifest 建立後啟用
//  3. 透過 channel 發送 load 請求到 LockOSThread worker
//  4. Worker 呼叫 CoreML_LoadModel()（Obj-C 端）
//  5. CoreML 載入失敗 → ErrInferenceUnavailable（使用者原則 #3）
func (e *coremlEngine) LoadModel(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return ErrEngineAlreadyClosed
	}

	// TODO(§14.6.6): 啟用 SHA256 模型完整性驗證。
	// 待 model_hashes.json manifest 建立後，取消以下註解：
	//
	//   verifier, vErr := NewModelVerifier(embeddedModelHashes)
	//   if vErr != nil {
	//       return fmt.Errorf("%w: manifest error — %v", ErrInferenceUnavailable, vErr)
	//   }
	//   if err := verifier.Verify(path); err != nil {
	//       return err // ErrModelIntegrityMismatch
	//   }

	// 發送 load 請求到 worker
	resp := make(chan coremlResponse, 1)
	e.reqCh <- coremlRequest{
		kind:      coremlReqLoad,
		modelPath: path,
		resp:      resp,
	}
	r := <-resp

	if r.err == nil {
		e.loaded = true
	}
	return r.err
}

// Infer 執行一次 CoreML 推論。
//
// 輸入：RGBA 原始位元組（4 bytes per pixel）+ 影像寬高。
// 輸出：RawTensor — 已從 native memory 複製到 Go heap 的浮點數陣列。
//
// Vision framework（Obj-C 端）自動處理：
//   - 影像縮放到模型輸入尺寸（通常 640×640）
//   - 色彩空間轉換
//
// 橋接層不做任何 YOLO 後處理 — 後處理在 yolo_postprocess.go 的
// DecodeYOLOOutput() 完成（BBox 解碼、NMS、score filter）。
func (e *coremlEngine) Infer(rgba []byte, width, height int) (RawTensor, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return RawTensor{}, ErrEngineAlreadyClosed
	}
	if !e.loaded {
		return RawTensor{}, ErrModelNotLoaded
	}

	// 發送 infer 請求到 worker
	resp := make(chan coremlResponse, 1)
	e.reqCh <- coremlRequest{
		kind:   coremlReqInfer,
		rgba:   rgba,
		width:  width,
		height: height,
		resp:   resp,
	}
	r := <-resp
	return r.tensor, r.err
}

// Close 釋放所有 CoreML native 資源。
//
// §14.6.2：透過 worker channel 發送 close 請求，
// 確保在 LockOSThread 的 worker 上釋放。
//
// 呼叫後引擎不可再使用：
//   - LoadModel() → ErrEngineAlreadyClosed
//   - Infer()     → ErrEngineAlreadyClosed
//
// 重複呼叫 Close() 安全（no-op）。
func (e *coremlEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil // 重複 Close 不報錯
	}
	e.closed = true
	e.loaded = false

	// 請求 worker 釋放資源並結束
	resp := make(chan coremlResponse, 1)
	e.reqCh <- coremlRequest{
		kind: coremlReqClose,
		resp: resp,
	}
	<-resp

	// 等待 worker goroutine 完全結束
	e.wg.Wait()
	return nil
}

// ──────────────────────────────────────────────
// Package-level 函式
// ──────────────────────────────────────────────

// InferenceBackendName 回傳此平台的推論後端名稱。
func InferenceBackendName() string {
	return "coreml"
}
