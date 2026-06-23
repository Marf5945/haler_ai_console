#!/usr/bin/env python3
"""Strip YOLOX head obj/cls sigmoid nodes from an ONNX export.

The app's Go postprocess contract expects raw logits for objectness and class
scores, then applies sigmoid in yolo_postprocess.go. Some YOLOX eval exports
already include sigmoid on obj/cls outputs even when decode_in_inference is off.
This script bypasses only those final head sigmoids, keeping SiLU activations
inside the backbone/head untouched.
"""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path

import onnx


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Bypass final YOLOX obj/cls sigmoid nodes")
    p.add_argument("--in", dest="input", type=Path, required=True, help="input ONNX")
    p.add_argument("--out", dest="output", type=Path, required=True, help="output ONNX")
    p.add_argument("--backup", type=Path, help="optional backup path before in-place overwrite")
    return p.parse_args()


def main() -> int:
    args = parse_args()
    if not args.input.is_file():
        raise SystemExit(f"input ONNX not found: {args.input}")

    if args.backup and args.input.resolve() == args.output.resolve():
        args.backup.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(args.input, args.backup)

    model = onnx.load(str(args.input))
    graph = model.graph

    bypass = {}
    remove_outputs = set()
    for node in graph.node:
        if node.op_type != "Sigmoid" or len(node.input) != 1 or len(node.output) != 1:
            continue
        src = node.input[0]
        dst = node.output[0]
        if src.startswith("/head/obj_preds.") or src.startswith("/head/cls_preds."):
            bypass[dst] = src
            remove_outputs.add(dst)

    if not bypass:
        raise SystemExit("no final YOLOX obj/cls sigmoid nodes found")

    rewritten_inputs = 0
    for node in graph.node:
        for i, value in enumerate(node.input):
            if value in bypass:
                node.input[i] = bypass[value]
                rewritten_inputs += 1

    kept = [node for node in graph.node if not (node.op_type == "Sigmoid" and node.output and node.output[0] in remove_outputs)]
    del graph.node[:]
    graph.node.extend(kept)

    onnx.checker.check_model(model)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    onnx.save(model, str(args.output))

    print(f"[ok] bypassed {len(bypass)} final sigmoid nodes; rewrote {rewritten_inputs} inputs")
    for dst, src in sorted(bypass.items()):
        print(f"  {dst} <- {src}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
