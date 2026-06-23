#!/usr/bin/env python3
"""Export the app's YOLOX button detector to ONNX with modern PyTorch."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import torch
from torch import nn


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Export YOLOX checkpoint to ONNX")
    parser.add_argument("--yolox-root", type=Path, required=True)
    parser.add_argument("-f", "--exp-file", type=Path, required=True)
    parser.add_argument("-c", "--ckpt", type=Path, required=True)
    parser.add_argument("--output-name", type=Path, required=True)
    parser.add_argument("--opset", type=int, default=11)
    parser.add_argument("--batch-size", type=int, default=1)
    parser.add_argument("--input", default="images")
    parser.add_argument("--output", default="output")
    parser.add_argument("--dynamic", action="store_true")
    parser.add_argument("--decode-in-inference", action="store_true")
    parser.add_argument("--no-onnxsim", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    sys.path.insert(0, str(args.yolox_root.resolve()))

    from yolox.exp import get_exp
    from yolox.models.network_blocks import SiLU
    from yolox.utils import replace_module

    exp = get_exp(str(args.exp_file), None)
    model = exp.get_model()
    checkpoint = torch.load(str(args.ckpt), map_location="cpu")
    if "model" in checkpoint:
        checkpoint = checkpoint["model"]
    model.load_state_dict(checkpoint)
    model.eval()
    model = replace_module(model, nn.SiLU, SiLU)
    model.head.decode_in_inference = args.decode_in_inference

    args.output_name.parent.mkdir(parents=True, exist_ok=True)
    dummy_input = torch.randn(args.batch_size, 3, exp.test_size[0], exp.test_size[1])
    dynamic_axes = None
    if args.dynamic:
        dynamic_axes = {args.input: {0: "batch"}, args.output: {0: "batch"}}

    torch.onnx.export(
        model,
        dummy_input,
        str(args.output_name),
        input_names=[args.input],
        output_names=[args.output],
        dynamic_axes=dynamic_axes,
        opset_version=args.opset,
    )

    if not args.no_onnxsim:
        import onnx
        from onnxsim import simplify

        onnx_model = onnx.load(str(args.output_name))
        simplified, ok = simplify(onnx_model)
        if not ok:
            raise RuntimeError("onnxsim could not validate the simplified model")
        onnx.save(simplified, str(args.output_name))

    print(f"generated ONNX model: {args.output_name}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
