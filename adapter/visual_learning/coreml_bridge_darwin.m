// coreml_bridge_darwin.m — CoreML Objective-C 實作（§14.6.1）。
//
// 規範依據：AI_Console_Spec_v4_2.md §14.6.1
//   「macOS CoreML bridge 的 Objective-C 程式碼必須在獨立 .m 檔案中，
//     不得 inline 進 .go 檔案。」
//
// 此檔案實作 coreml_bridge_darwin.h 定義的 C API。
// 使用 Apple Vision framework 處理影像輸入，自動處理格式轉換和縮放。
// 所有 Objective-C 物件在 @autoreleasepool 內管理記憶體。
//
// 推論流程：
//   RGBA bytes → CGImage → VNImageRequestHandler
//   → VNCoreMLRequest（自動縮放到模型輸入尺寸）
//   → MLMultiArray → float32 陣列複製到 C heap
//
// 檔案命名含 _darwin 後綴，Go build 系統只在 macOS 編譯此檔案。

#import <Foundation/Foundation.h>
#import <CoreML/CoreML.h>
#import <Vision/Vision.h>
#import <CoreGraphics/CoreGraphics.h>
#include <stdlib.h>
#include <string.h>
#include "coreml_bridge_darwin.h"

// ══════════════════════════════════════════════════
// 內部結構
// ══════════════════════════════════════════════════

// CoreMLSession 持有載入的模型和 Vision model wrapper。
// model 和 vnModel 使用 __bridge_retained 轉移給 C 管理，
// Close 時用 CFRelease 釋放。
typedef struct {
    void* model;     // MLModel*（__bridge_retained ownership）
    void* vnModel;   // VNCoreMLModel*（__bridge_retained ownership）
} CoreMLSession;

// ══════════════════════════════════════════════════
// 工具函式
// ══════════════════════════════════════════════════

// copyErrString 複製 NSError 的描述到 C heap。
// 回傳值必須由 caller 使用 CoreML_FreeString() 釋放。
static char* copyErrString(NSError* error) {
    if (!error) return NULL;
    const char* desc = [[error localizedDescription] UTF8String];
    if (!desc) return NULL;
    size_t len = strlen(desc);
    char* copy = (char*)malloc(len + 1);
    if (copy) memcpy(copy, desc, len + 1);
    return copy;
}

// makeCString 從 C 字串複製到新的 heap 分配。
// 回傳值必須由 caller 使用 CoreML_FreeString() 釋放。
static char* makeCString(const char* str) {
    if (!str) return NULL;
    size_t len = strlen(str);
    char* copy = (char*)malloc(len + 1);
    if (copy) memcpy(copy, str, len + 1);
    return copy;
}

// ══════════════════════════════════════════════════
// CoreML_LoadModel — 載入模型
// ══════════════════════════════════════════════════

CoreMLHandle CoreML_LoadModel(const char* modelDirPath, char** errMsg) {
    @autoreleasepool {
        // ── 參數檢查 ──
        if (!modelDirPath) {
            if (errMsg) *errMsg = makeCString("model path is NULL");
            return NULL;
        }

        NSString* path = [NSString stringWithUTF8String:modelDirPath];
        NSURL* url = [NSURL fileURLWithPath:path];
        NSError* error = nil;

        // ── MLModelConfiguration ──
        // computeUnits = All：讓 CoreML 自動選擇最佳裝置
        //   Apple Silicon → Neural Engine（最快）
        //   Intel Mac → GPU 或 CPU
        MLModelConfiguration* config = [[MLModelConfiguration alloc] init];
        config.computeUnits = MLComputeUnitsAll;

        // ── 載入 MLModel ──
        // .mlmodelc 是 CoreML 編譯後的目錄格式，
        // 包含模型權重和中繼資料。
        MLModel* model = [MLModel modelWithContentsOfURL:url
                                           configuration:config
                                                   error:&error];
        if (!model) {
            if (errMsg) *errMsg = copyErrString(error);
            return NULL;
        }

        // ── 建立 VNCoreMLModel ──
        // Vision framework wrapper，自動處理：
        //   - 影像格式轉換（RGBA → 模型期望的格式）
        //   - 影像縮放（任意尺寸 → 模型輸入尺寸，通常 640×640）
        VNCoreMLModel* vnModel = [VNCoreMLModel modelForMLModel:model error:&error];
        if (!vnModel) {
            if (errMsg) *errMsg = copyErrString(error);
            return NULL;
        }

        // ── 建立 Session ──
        CoreMLSession* session = (CoreMLSession*)calloc(1, sizeof(CoreMLSession));
        if (!session) {
            if (errMsg) *errMsg = makeCString("failed to allocate CoreML session");
            return NULL;
        }

        // __bridge_retained：ARC ownership 轉移給 C 管理。
        // 必須在 CoreML_Close() 中用 CFRelease() 釋放。
        session->model = (__bridge_retained void*)model;
        session->vnModel = (__bridge_retained void*)vnModel;

        return (CoreMLHandle)session;
    }
}

// ══════════════════════════════════════════════════
// CoreML_Infer — 執行推論
// ══════════════════════════════════════════════════

int CoreML_Infer(CoreMLHandle handle,
                 const uint8_t* rgbaData, int width, int height,
                 float** outData, int* outCount,
                 int** outShape, int* outShapeDims,
                 char** errMsg) {
    @autoreleasepool {
        // ── 參數檢查 ──
        if (!handle || !rgbaData || width <= 0 || height <= 0) {
            if (errMsg) *errMsg = makeCString("invalid inference parameters");
            return -1;
        }

        CoreMLSession* session = (CoreMLSession*)handle;
        VNCoreMLModel* vnModel = (__bridge VNCoreMLModel*)session->vnModel;

        // ════════════════════════════════════════
        // Step 1: RGBA bytes → CGImage
        // ════════════════════════════════════════
        // kCGImageAlphaPremultipliedLast = RGBA layout，alpha 在最後一個 byte。
        // kCGBitmapByteOrder32Big = 大端序（RGBA 標準排列）。
        CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
        if (!colorSpace) {
            if (errMsg) *errMsg = makeCString("failed to create color space");
            return -1;
        }

        CGContextRef ctx = CGBitmapContextCreate(
            (void*)rgbaData,
            (size_t)width,
            (size_t)height,
            8,                  // bits per component
            (size_t)(width * 4), // bytes per row (RGBA = 4 bytes)
            colorSpace,
            kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);

        if (!ctx) {
            CGColorSpaceRelease(colorSpace);
            if (errMsg) *errMsg = makeCString("failed to create bitmap context");
            return -1;
        }

        CGImageRef cgImage = CGBitmapContextCreateImage(ctx);
        CGContextRelease(ctx);
        CGColorSpaceRelease(colorSpace);

        if (!cgImage) {
            if (errMsg) *errMsg = makeCString("failed to create CGImage from RGBA data");
            return -1;
        }

        // ════════════════════════════════════════
        // Step 2: Vision framework 推論
        // ════════════════════════════════════════
        // VNCoreMLRequest 自動：
        //   - 將 CGImage 縮放到模型輸入尺寸
        //   - 轉換色彩空間
        //   - 執行 MLModel prediction
        __block MLMultiArray* outputArray = nil;
        __block NSError* inferError = nil;

        VNCoreMLRequest* request = [[VNCoreMLRequest alloc]
            initWithModel:vnModel
            completionHandler:^(VNRequest* req, NSError* err) {
                if (err) {
                    inferError = err;
                    return;
                }
                // 從結果中找到第一個 MLMultiArray 輸出。
                // YOLOv5 nano 通常只有一個輸出 tensor。
                for (VNCoreMLFeatureValueObservation* obs in req.results) {
                    if ([obs isKindOfClass:[VNCoreMLFeatureValueObservation class]] &&
                        obs.featureValue.type == MLFeatureTypeMultiArray) {
                        outputArray = obs.featureValue.multiArrayValue;
                        break;
                    }
                }
            }];

        // ScaleFill：填滿整個輸入區域，不留黑邊。
        // YOLO 期望完整影像，letterbox padding 在模型轉換時已處理。
        request.imageCropAndScaleOption = VNImageCropAndScaleOptionScaleFill;

        VNImageRequestHandler* handler = [[VNImageRequestHandler alloc]
            initWithCGImage:cgImage options:@{}];

        NSError* runError = nil;
        BOOL success = [handler performRequests:@[request] error:&runError];

        // CGImage 不再需要，立即釋放
        CGImageRelease(cgImage);

        if (!success || runError) {
            if (errMsg) *errMsg = copyErrString(runError);
            return -1;
        }
        if (inferError) {
            if (errMsg) *errMsg = copyErrString(inferError);
            return -1;
        }
        if (!outputArray) {
            if (errMsg) *errMsg = makeCString("no MLMultiArray output found in model results");
            return -1;
        }

        // ════════════════════════════════════════
        // Step 3: MLMultiArray → C float32 陣列
        // ════════════════════════════════════════
        // 將推論結果從 CoreML managed memory 複製到 C heap，
        // 讓 Go 端可以安全地接收（§14.6.3：unsafe 限制在 bridge 內）。
        NSArray<NSNumber*>* shape = outputArray.shape;
        NSInteger dims = (NSInteger)shape.count;
        NSInteger totalElements = 1;

        // 複製 shape
        int* shapeArr = (int*)malloc(sizeof(int) * (size_t)dims);
        if (!shapeArr) {
            if (errMsg) *errMsg = makeCString("failed to allocate shape array");
            return -1;
        }
        for (NSInteger i = 0; i < dims; i++) {
            shapeArr[i] = shape[i].intValue;
            totalElements *= shape[i].integerValue;
        }

        // 分配輸出 float32 陣列
        float* dataArr = (float*)malloc(sizeof(float) * (size_t)totalElements);
        if (!dataArr) {
            free(shapeArr);
            if (errMsg) *errMsg = makeCString("failed to allocate output data array");
            return -1;
        }

        // 根據 MLMultiArray 的資料型別轉換。
        // YOLOv5 CoreML 模型通常輸出 Float32 或 Float16。
        MLMultiArrayDataType dtype = outputArray.dataType;
        const void* srcPtr = outputArray.dataPointer;

        if (dtype == MLMultiArrayDataTypeFloat32) {
            // Float32：直接 memcpy（最快路徑）
            memcpy(dataArr, srcPtr, sizeof(float) * (size_t)totalElements);

        } else if (dtype == MLMultiArrayDataTypeDouble) {
            // Float64 → Float32：逐元素轉換
            const double* src = (const double*)srcPtr;
            for (NSInteger i = 0; i < totalElements; i++) {
                dataArr[i] = (float)src[i];
            }

        } else if (dtype == MLMultiArrayDataTypeFloat16) {
            // Float16 → Float32 轉換。
            // Apple Silicon 原生支援 __fp16 型別（ARM NEON）。
            // Intel Mac 使用 NSNumber fallback（較慢但正確）。
            #if defined(__arm64__) || defined(__aarch64__)
            const __fp16* src = (const __fp16*)srcPtr;
            for (NSInteger i = 0; i < totalElements; i++) {
                dataArr[i] = (float)src[i];
            }
            #else
            // Intel Mac fallback：透過 NSNumber 逐元素轉換。
            // 效能較差，但 Intel Mac 上 CoreML 推論本身就較慢，
            // 這個開銷相對可以接受。
            for (NSInteger i = 0; i < totalElements; i++) {
                NSNumber* num = [outputArray objectAtIndexedSubscript:i];
                dataArr[i] = num.floatValue;
            }
            #endif

        } else {
            // 不支援的資料型別
            free(dataArr);
            free(shapeArr);
            if (errMsg) *errMsg = makeCString("unsupported MLMultiArray data type");
            return -1;
        }

        // ── 寫入輸出參數 ──
        *outData = dataArr;
        *outCount = (int)totalElements;
        *outShape = shapeArr;
        *outShapeDims = (int)dims;

        return 0; // 成功
    }
}

// ══════════════════════════════════════════════════
// CoreML_Close — 釋放資源
// ══════════════════════════════════════════════════

void CoreML_Close(CoreMLHandle handle) {
    if (!handle) return;

    @autoreleasepool {
        CoreMLSession* session = (CoreMLSession*)handle;

        // CFRelease 對應 __bridge_retained 的 ownership 轉移。
        // 這會觸發 ARC 釋放 MLModel 和 VNCoreMLModel。
        if (session->vnModel) {
            CFRelease(session->vnModel);
            session->vnModel = NULL;
        }
        if (session->model) {
            CFRelease(session->model);
            session->model = NULL;
        }

        free(session);
    }
}

// ══════════════════════════════════════════════════
// 記憶體釋放
// ══════════════════════════════════════════════════

void CoreML_FreeFloats(float* ptr) {
    free(ptr);
}

void CoreML_FreeInts(int* ptr) {
    free(ptr);
}

void CoreML_FreeString(char* str) {
    free(str);
}
