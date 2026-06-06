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
#import <CoreVideo/CoreVideo.h>
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

        // ── 載入 MLModel ──
        // .mlmodelc 是 CoreML 編譯後的目錄格式，
        // 包含模型權重和中繼資料。
        MLModel* model = nil;
        if (@available(macOS 10.14, *)) {
            // ── MLModelConfiguration ──
            // computeUnits = All：讓 CoreML 自動選擇最佳裝置
            //   Apple Silicon → Neural Engine（最快）
            //   Intel Mac → GPU 或 CPU
            MLModelConfiguration* config = [[MLModelConfiguration alloc] init];
            config.computeUnits = MLComputeUnitsAll;
            model = [MLModel modelWithContentsOfURL:url
                                      configuration:config
                                              error:&error];
        } else {
            model = [MLModel modelWithContentsOfURL:url error:&error];
        }
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
        MLModel* model = (__bridge MLModel*)session->model;

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
        // Step 2: CGImage → CVPixelBuffer → MLModel prediction
        // ════════════════════════════════════════
        // 直接呼叫 MLModel，避開 VNCoreMLRequest 對部分匯出模型的
        // VNCoreMLTransform 限制。模型的 ImageType input 會接收
        // CVPixelBuffer，scale/bias 已寫在 .mlmodel 內。
        NSString* inputName = model.modelDescription.inputDescriptionsByName.allKeys.firstObject;
        if (!inputName) {
            CGImageRelease(cgImage);
            if (errMsg) *errMsg = makeCString("model has no input feature");
            return -1;
        }

        MLFeatureDescription* inputDesc = model.modelDescription.inputDescriptionsByName[inputName];
        NSInteger targetWidth = 416;
        NSInteger targetHeight = 416;
        if (inputDesc.imageConstraint) {
            if (inputDesc.imageConstraint.pixelsWide > 0) {
                targetWidth = inputDesc.imageConstraint.pixelsWide;
            }
            if (inputDesc.imageConstraint.pixelsHigh > 0) {
                targetHeight = inputDesc.imageConstraint.pixelsHigh;
            }
        }

        NSDictionary* attrs = @{
            (NSString*)kCVPixelBufferCGImageCompatibilityKey: @YES,
            (NSString*)kCVPixelBufferCGBitmapContextCompatibilityKey: @YES
        };
        CVPixelBufferRef pixelBuffer = NULL;
        CVReturn cvErr = CVPixelBufferCreate(
            kCFAllocatorDefault,
            (size_t)targetWidth,
            (size_t)targetHeight,
            kCVPixelFormatType_32BGRA,
            (__bridge CFDictionaryRef)attrs,
            &pixelBuffer);
        if (cvErr != kCVReturnSuccess || !pixelBuffer) {
            CGImageRelease(cgImage);
            if (errMsg) *errMsg = makeCString("failed to create CVPixelBuffer");
            return -1;
        }

        CVPixelBufferLockBaseAddress(pixelBuffer, 0);
        void* baseAddress = CVPixelBufferGetBaseAddress(pixelBuffer);
        size_t bytesPerRow = CVPixelBufferGetBytesPerRow(pixelBuffer);
        CGColorSpaceRef pixelColorSpace = CGColorSpaceCreateDeviceRGB();
        CGContextRef pixelCtx = CGBitmapContextCreate(
            baseAddress,
            (size_t)targetWidth,
            (size_t)targetHeight,
            8,
            bytesPerRow,
            pixelColorSpace,
            kCGImageAlphaPremultipliedFirst | kCGBitmapByteOrder32Little);
        if (pixelColorSpace) {
            CGColorSpaceRelease(pixelColorSpace);
        }
        if (!pixelCtx) {
            CVPixelBufferUnlockBaseAddress(pixelBuffer, 0);
            CVPixelBufferRelease(pixelBuffer);
            CGImageRelease(cgImage);
            if (errMsg) *errMsg = makeCString("failed to create pixel buffer bitmap context");
            return -1;
        }
        CGContextDrawImage(pixelCtx, CGRectMake(0, 0, targetWidth, targetHeight), cgImage);
        CGContextRelease(pixelCtx);
        CVPixelBufferUnlockBaseAddress(pixelBuffer, 0);
        CGImageRelease(cgImage);

        NSError* featureError = nil;
        MLFeatureValue* imageValue = [MLFeatureValue featureValueWithPixelBuffer:pixelBuffer];
        MLDictionaryFeatureProvider* provider = [[MLDictionaryFeatureProvider alloc]
            initWithDictionary:@{ inputName: imageValue }
                          error:&featureError];
        if (!provider || featureError) {
            CVPixelBufferRelease(pixelBuffer);
            if (errMsg) *errMsg = copyErrString(featureError);
            return -1;
        }

        NSError* predError = nil;
        id<MLFeatureProvider> prediction = [model predictionFromFeatures:provider error:&predError];
        CVPixelBufferRelease(pixelBuffer);
        if (!prediction || predError) {
            if (errMsg) *errMsg = copyErrString(predError);
            return -1;
        }

        MLMultiArray* outputArray = nil;
        for (NSString* featureName in prediction.featureNames) {
            MLFeatureValue* value = [prediction featureValueForName:featureName];
            if (value.type == MLFeatureTypeMultiArray) {
                outputArray = value.multiArrayValue;
                break;
            }
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

        } else if (@available(macOS 12.0, *)) {
            if (dtype == MLMultiArrayDataTypeFloat16) {
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

// VisionOCR_RecognizeImage uses macOS Apple Vision OCR. This is an optional
// helper for visual anchors: it may describe text inside a small cropped button
// image, but replay code must still use YOLO/layout/candidate IDs for clicks.
int VisionOCR_RecognizeImage(const uint8_t* imageData,
                             int imageLen,
                             char** outJSON,
                             char** errMsg) {
    @autoreleasepool {
        if (!imageData || imageLen <= 0 || !outJSON) {
            if (errMsg) *errMsg = makeCString("invalid OCR image parameters");
            return -1;
        }

        if (@available(macOS 10.15, *)) {
            NSData* data = [NSData dataWithBytes:imageData length:(NSUInteger)imageLen];
            if (!data) {
                if (errMsg) *errMsg = makeCString("failed to create OCR image data");
                return -1;
            }

            __block NSError* requestError = nil;
            VNRecognizeTextRequest* request =
                [[VNRecognizeTextRequest alloc] initWithCompletionHandler:
                 ^(VNRequest* req, NSError* error) {
                    requestError = error;
                 }];
            request.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
            request.usesLanguageCorrection = YES;

            VNImageRequestHandler* handler =
                [[VNImageRequestHandler alloc] initWithData:data options:@{}];
            NSError* performError = nil;
            BOOL ok = [handler performRequests:@[request] error:&performError];
            if (!ok || performError || requestError) {
                if (errMsg) *errMsg = copyErrString(performError ?: requestError);
                return -1;
            }

            NSMutableArray* rows = [NSMutableArray array];
            for (VNRecognizedTextObservation* observation in request.results) {
                NSArray<VNRecognizedText*>* candidates = [observation topCandidates:1];
                VNRecognizedText* best = candidates.firstObject;
                if (!best || best.string.length == 0) continue;

                CGRect box = observation.boundingBox;
                [rows addObject:@{
                    @"text": best.string ?: @"",
                    @"confidence": @(best.confidence),
                    @"bbox": @[@(box.origin.x), @(box.origin.y), @(box.size.width), @(box.size.height)]
                }];
            }

            NSError* jsonError = nil;
            NSData* jsonData = [NSJSONSerialization dataWithJSONObject:rows
                                                               options:0
                                                                 error:&jsonError];
            if (!jsonData) {
                if (errMsg) *errMsg = copyErrString(jsonError);
                return -1;
            }
            NSString* jsonString = [[NSString alloc] initWithData:jsonData
                                                         encoding:NSUTF8StringEncoding];
            *outJSON = makeCString(jsonString.UTF8String);
            return *outJSON ? 0 : -1;
        }

        if (errMsg) *errMsg = makeCString("Apple Vision text recognition requires macOS 10.15 or newer");
        return -1;
    }
}

void CoreML_FreeFloats(float* ptr) {
    free(ptr);
}

void CoreML_FreeInts(int* ptr) {
    free(ptr);
}

void CoreML_FreeString(char* str) {
    free(str);
}
