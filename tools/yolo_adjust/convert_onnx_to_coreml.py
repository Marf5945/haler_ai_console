#!/usr/bin/env python3
"""ONNX -> CoreML (.mlmodelc) converter for the YOLOX button detector.

為什麼是這條路：
  - coremltools 6.0+ 已移除 ONNX frontend（source='onnx' 不存在），
    舊的 onnx-coreml 依賴上古 nnssa，現代 Mac 上都跑不起來。
  - 唯一可行的現代路徑：ONNX --(onnx2torch)--> torch.nn.Module
    --(torch.jit.trace)--> TorchScript --(ct.convert, source='pytorch')--> CoreML。
  - .mlmodelc 不需要 Xcode 的 coremlcompiler CLI；用
    coremltools.models.utils.compile_model() 純 Python 就能編譯。

對齊 app 的關鍵設定（來自 adapter/visual_learning 原始碼）：
  - app.go 找的 base path 是 assets/models/yolox_button_s，macOS 自動加 .mlmodelc
    → 產物固定輸出成 assets/models/yolox_button_s.mlmodelc
  - coreml_bridge_darwin.m：輸入是 ImageType（CVPixelBuffer / 32BGRA），
    且「scale/bias 已寫在 .mlmodel 內」→ 正規化必須烤進模型。
  - yolo_postprocess.go：YOLOX-S，InputSize=640，單類別 button，
    解碼（stride 8/16/32、grid、exp）在 Go 端做 → 模型要輸出「未解碼 raw tensor」，
    所以當初匯出 ONNX 時 --decode-in-inference 必須是關的。
  - 座標還原用 stretch（非 letterbox）→ CoreML ImageType 直接縮放即可。

前處理（YOLOX 標準、非 legacy）：0~255、BGR、stretch resize 到 640x640，
  不做 /255、不做 mean/std。對應 ImageType: color_layout=BGR, scale=1.0, bias=0。
  若這顆權重是舊式（需 /255 + ImageNet mean/std），加 --legacy。

用法（在 Mac、乾淨 venv）：
  python3 -m venv /tmp/yolox-coreml && source /tmp/yolox-coreml/bin/activate
  pip install --upgrade pip
  pip install "coremltools>=8.0" onnx onnx2torch torch onnxruntime pillow numpy
  python tools/yolo_adjust/convert_onnx_to_coreml.py \
      --onnx assets/models/yolox_button_s.onnx \
      --out  assets/models/yolox_button_s.mlmodelc

跑完會印出 CoreML vs onnxruntime 的最大絕對誤差；< 1e-3 視為前處理/通道順序正確。
"""

from __future__ import annotations

import argparse
import os
import shutil
import sys
import tempfile
from pathlib import Path

import numpy as np


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Convert YOLOX ONNX to CoreML .mlmodelc")
    p.add_argument("--onnx", type=Path, required=True, help="輸入 ONNX 檔")
    p.add_argument("--out", type=Path, required=True,
                   help="輸出 .mlmodelc 目錄路徑（例如 assets/models/yolox_button_s.mlmodelc）")
    p.add_argument("--input-size", type=int, default=640, help="YOLOX 方形輸入邊長")
    p.add_argument("--color", choices=["BGR", "RGB"], default="BGR",
                   help="模型期望的通道順序；YOLOX 標準預設 BGR")
    p.add_argument("--legacy", action="store_true",
                   help="舊式權重：輸入 /255 後再套 ImageNet mean/std")
    p.add_argument("--convert-to", choices=["mlprogram", "neuralnetwork"], default="mlprogram",
                   help="CoreML 輸出格式；若 mlprogram 可驗證但 app 載入失敗，試 neuralnetwork")
    p.add_argument("--skip-verify", action="store_true", help="略過數值驗證")
    p.add_argument("--verify-tol", type=float, default=1e-3, help="驗證可接受最大絕對誤差")
    return p.parse_args()


def build_scale_bias(legacy: bool, color: str):
    """回傳 (scale, bias_list) 給 ct.ImageType。

    CoreML 對每個像素做： out = pixel * scale + bias（pixel 為 0~255）。

    非 legacy：YOLOX 標準直接吃 0~255 BGR → scale=1, bias=0。
    legacy：先 /255 得 0~1，再 (x-mean)/std。
      展開成 pixel 軸： out = pixel * (1/255/std) + (-mean/std)
      ImageNet mean=[.485,.456,.406], std=[.229,.224,.225]（RGB 順序）。
    """
    if not legacy:
        return 1.0, [0.0, 0.0, 0.0]

    mean_rgb = np.array([0.485, 0.456, 0.406])
    std_rgb = np.array([0.229, 0.224, 0.225])
    if color == "BGR":
        mean = mean_rgb[::-1]
        std = std_rgb[::-1]
    else:
        mean, std = mean_rgb, std_rgb
    # ct.ImageType 的 scale 是純量、bias 是長度 3 陣列。
    # legacy 三通道 std 不同 → scale 無法用單一純量精準表示。
    # 折衷：用平均 std 當 scale，bias 各通道單獨算（足夠通過寬鬆驗證；
    # 若驗證不過，請改走 --legacy 的精準版本，見檔尾 NOTE）。
    scale = float(1.0 / 255.0 / std.mean())
    bias = [float(-m / s) for m, s in zip(mean, std)]
    return scale, bias


def onnx_io_names(onnx_path: Path):
    import onnx
    m = onnx.load(str(onnx_path))
    inp = m.graph.input[0].name
    out = m.graph.output[0].name
    return inp, out


def main() -> int:
    args = parse_args()
    if not args.onnx.is_file():
        print(f"[ERR] ONNX 不存在：{args.onnx}", file=sys.stderr)
        return 2

    os.environ.setdefault("TMPDIR", "/private/tmp/")
    tempfile.tempdir = os.environ["TMPDIR"]

    import torch
    import coremltools as ct
    from coremltools.models.utils import compile_model
    from onnx2torch import convert as onnx2torch_convert

    S = args.input_size
    in_name, out_name = onnx_io_names(args.onnx)
    print(f"[info] ONNX input='{in_name}' output='{out_name}' size={S} color={args.color} legacy={args.legacy}")

    # 1) ONNX -> torch.nn.Module
    torch_model = onnx2torch_convert(str(args.onnx)).eval()

    # 2) TorchScript trace
    example = torch.rand(1, 3, S, S)
    with torch.no_grad():
        traced = torch.jit.trace(torch_model, example, check_trace=False)

    # 3) torch -> CoreML，輸入用 ImageType 並把正規化烤進模型
    scale, bias = build_scale_bias(args.legacy, args.color)
    color_layout = ct.colorlayout.BGR if args.color == "BGR" else ct.colorlayout.RGB
    image_input = ct.ImageType(
        name=in_name,
        shape=(1, 3, S, S),
        scale=scale,
        bias=bias,
        color_layout=color_layout,
        channel_first=True,
    )
    print(f"[info] ImageType scale={scale} bias={bias}")

    convert_kwargs = {
        "inputs": [image_input],
        "convert_to": args.convert_to,
        "compute_units": ct.ComputeUnit.ALL,
    }
    if args.convert_to == "mlprogram":
        convert_kwargs["compute_precision"] = ct.precision.FLOAT32
        convert_kwargs["minimum_deployment_target"] = ct.target.macOS13
    mlmodel = ct.convert(traced, **convert_kwargs)

    # 4) 存 mlpackage 再編譯成 .mlmodelc（不需要 coremlcompiler CLI）
    out_dir = args.out
    out_dir.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir="/private/tmp") as tmp:
        pkg = Path(tmp) / "model.mlpackage"
        mlmodel.save(str(pkg))
        compiled = compile_model(str(pkg), str(Path(tmp) / "compiled.mlmodelc"))
        compiled = Path(compiled)
        if out_dir.exists():
            shutil.rmtree(out_dir)
        shutil.copytree(compiled, out_dir)
    print(f"[ok] 已輸出 {out_dir}")

    # 5) 數值驗證：onnxruntime vs CoreML 在同一張圖上比對
    if args.skip_verify:
        print("[info] 跳過驗證（--skip-verify）")
        return 0

    try:
        import onnxruntime as ort
        from PIL import Image
    except ImportError:
        print("[warn] 缺 onnxruntime / pillow，跳過驗證；建議補裝後重跑驗證。")
        return 0

    rng = np.random.default_rng(0)
    rgb = rng.integers(0, 256, size=(S, S, 3), dtype=np.uint8)  # H,W,3 (RGB)
    pil = Image.fromarray(rgb, mode="RGB")

    # onnxruntime：依模型期望排通道（BGR 就反轉），0~255 -> NCHW float32。
    arr = rgb.astype(np.float32)
    if args.color == "BGR":
        arr = arr[:, :, ::-1]
    if args.legacy:
        arr = arr / 255.0
        m = np.array([0.485, 0.456, 0.406], np.float32)
        s = np.array([0.229, 0.224, 0.225], np.float32)
        if args.color == "BGR":
            m, s = m[::-1], s[::-1]
        arr = (arr - m) / s
    nchw = np.transpose(arr, (2, 0, 1))[None, ...].astype(np.float32)

    sess = ort.InferenceSession(str(args.onnx), providers=["CPUExecutionProvider"])
    onnx_out = sess.run([out_name], {in_name: nchw})[0]

    # CoreML：直接餵 PIL RGB 影像，color_layout/scale/bias 由模型內部處理
    spec = mlmodel.get_spec()
    cm_out_name = spec.description.output[0].name
    cm_out = mlmodel.predict({in_name: pil})[cm_out_name]
    cm_out = np.asarray(cm_out)

    if cm_out.shape != onnx_out.shape:
        print(f"[warn] 形狀不同 onnx={onnx_out.shape} coreml={cm_out.shape}")
    diff = float(np.max(np.abs(cm_out.reshape(-1)[: onnx_out.size]
                                - onnx_out.reshape(-1)[: cm_out.size])))
    print(f"[verify] max|coreml - onnx| = {diff:.6e}  (tol={args.verify_tol})")
    if diff <= args.verify_tol:
        print("[verify] PASS：前處理/通道順序正確。")
        return 0
    print("[verify] FAIL：誤差過大。常見原因：通道順序（試 --color RGB）、"
          "或正規化（這顆是舊式權重 → 試 --legacy）。見檔尾 NOTE。")
    return 1


# NOTE（legacy 三通道 std 不同的精準版）：
#   ct.ImageType 的 scale 只能是單一純量，無法精準表達三通道不同的 std。
#   若 --legacy 驗證誤差過大，請改用以下做法之一：
#     (a) 在 torch 模型最前面包一層固定的 Normalize（mean/std 寫死），
#         讓 ImageType 維持 scale=1/255、bias=0；或
#     (b) 確認這顆其實是非 legacy（多數自訓 yolox_button_s 都是），直接不要 --legacy。
if __name__ == "__main__":
    raise SystemExit(main())
