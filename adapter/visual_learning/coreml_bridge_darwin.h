// coreml_bridge_darwin.h — CoreML C API（§14.6.1）。
//
// Go 端（coreml_bridge_darwin.go）和 Obj-C 端（coreml_bridge_darwin.m）共用此 header。
// 此 header 定義純 C 介面，不含任何 Objective-C 語法。
//
// 記憶體規則：
//   - errMsg 由 callee 分配，caller 用 CoreML_FreeString() 釋放。
//   - outData 由 callee 分配，caller 用 CoreML_FreeFloats() 釋放。
//   - outShape 由 callee 分配，caller 用 CoreML_FreeInts() 釋放。

#ifndef COREML_BRIDGE_DARWIN_H
#define COREML_BRIDGE_DARWIN_H

#include <stdint.h>

// CoreMLHandle 是 CoreML session 的不透明指標。
// 內部指向 Objective-C 的 CoreMLSession 結構。
typedef void* CoreMLHandle;

// CoreML_LoadModel 從 .mlmodelc 目錄載入 CoreML 模型。
//
// modelDirPath: .mlmodelc 目錄的完整路徑。
// errMsg:       失敗時設定錯誤訊息（caller 負責 CoreML_FreeString）。
//
// 成功回傳 handle，失敗回傳 NULL。
CoreMLHandle CoreML_LoadModel(const char* modelDirPath, char** errMsg);

// CoreML_Infer 執行一次 CoreML 推論。
//
// 輸入：
//   handle:   LoadModel 回傳的 session handle。
//   rgbaData: RGBA 影像原始位元組（4 bytes per pixel）。
//   width:    影像寬度（pixels）。
//   height:   影像高度（pixels）。
//
// 輸出（caller 負責釋放）：
//   outData:      float32 陣列（推論結果）。
//   outCount:     outData 的元素總數。
//   outShape:     tensor shape 陣列（例如 [1, 25200, 85]）。
//   outShapeDims: outShape 的維度數。
//   errMsg:       失敗時設定錯誤訊息。
//
// 成功回傳 0，失敗回傳 -1。
int CoreML_Infer(CoreMLHandle handle,
                 const uint8_t* rgbaData, int width, int height,
                 float** outData, int* outCount,
                 int** outShape, int* outShapeDims,
                 char** errMsg);

// CoreML_Close 釋放 CoreML session 的所有資源。
// handle 為 NULL 時安全（no-op）。
void CoreML_Close(CoreMLHandle handle);

// ── 記憶體釋放 ──

void CoreML_FreeFloats(float* ptr);
void CoreML_FreeInts(int* ptr);
void CoreML_FreeString(char* str);

#endif // COREML_BRIDGE_DARWIN_H
