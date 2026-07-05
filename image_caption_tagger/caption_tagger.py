#!/usr/bin/env python3
"""
Generate editable .txt caption/tag files next to images for LoRA training.

This script intentionally avoids downloading vision models. Its suggestions are
basic seed tags from filenames and simple image properties, meant to be edited.
"""

from __future__ import annotations

import argparse
import csv
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

try:
    from PIL import Image, ImageStat
except ImportError:  # pragma: no cover
    Image = None
    ImageStat = None


IMAGE_EXTENSIONS = {
    ".avif",
    ".bmp",
    ".gif",
    ".jpeg",
    ".jpg",
    ".png",
    ".tif",
    ".tiff",
    ".webp",
}

DEFAULT_WD14_MODEL = "SmilingWolf/wd-v1-4-swinv2-tagger-v2"

STOPWORDS = {
    "image",
    "img",
    "photo",
    "picture",
    "pic",
    "copy",
    "edit",
    "final",
    "new",
    "untitled",
    "screenshot",
    "screen",
    "shot",
}


@dataclass
class ImageInfo:
    path: Path
    width: int | None = None
    height: int | None = None
    mode: str | None = None
    brightness: float | None = None
    saturation: float | None = None
    warmth: float | None = None
    has_alpha: bool = False


@dataclass
class WD14Options:
    enabled: bool = False
    model_id: str = DEFAULT_WD14_MODEL
    threshold: float = 0.35
    character_threshold: float = 0.85
    include_ratings: bool = False
    keep_underscores: bool = False
    cache_dir: str | None = None


@dataclass
class WD14Tag:
    name: str
    category: int


def split_tags(value: str | None) -> list[str]:
    if not value:
        return []
    parts = re.split(r"[,;\n]+", value)
    return [normalize_tag(part) for part in parts if normalize_tag(part)]


def normalize_tag(tag: str) -> str:
    tag = tag.strip().lower()
    tag = re.sub(r"\s+", " ", tag)
    tag = tag.strip(" ,;")
    return tag


def unique_tags(tags: Iterable[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for tag in tags:
        normalized = normalize_tag(tag)
        if not normalized or normalized in seen:
            continue
        seen.add(normalized)
        result.append(normalized)
    return result


def filename_tags(path: Path) -> list[str]:
    stem = path.stem.lower()
    stem = re.sub(r"\([^)]+\)", " ", stem)
    stem = re.sub(r"\[[^\]]+\]", " ", stem)
    pieces = re.split(r"[^a-z0-9\u4e00-\u9fff]+", stem)
    tags: list[str] = []
    for piece in pieces:
        piece = piece.strip("_- ")
        if not piece or piece in STOPWORDS:
            continue
        if piece.isdigit():
            continue
        if len(piece) == 1 and piece.isascii():
            continue
        tags.append(piece.replace("_", " "))
    return tags


def format_wd14_tag(name: str, keep_underscores: bool) -> str:
    if keep_underscores:
        return name
    return name.replace("_", " ")


class WD14Tagger:
    def __init__(self, options: WD14Options):
        self.options = options
        self.session = None
        self.input_name = ""
        self.input_size = 448
        self.channels_last = True
        self.tags: list[WD14Tag] = []

        try:
            import numpy as np
            import onnxruntime as ort
            from huggingface_hub import hf_hub_download
        except ImportError as exc:
            raise SystemExit(
                "WD14 mode requires extra packages. Install them with:\n"
                "  python3 -m pip install onnxruntime huggingface_hub numpy pillow\n"
                f"Missing import: {exc.name}"
            ) from exc

        self.np = np
        model_path = hf_hub_download(
            repo_id=options.model_id,
            filename="model.onnx",
            cache_dir=options.cache_dir,
        )
        tags_path = hf_hub_download(
            repo_id=options.model_id,
            filename="selected_tags.csv",
            cache_dir=options.cache_dir,
        )

        providers = ["CPUExecutionProvider"]
        self.session = ort.InferenceSession(model_path, providers=providers)
        model_input = self.session.get_inputs()[0]
        self.input_name = model_input.name
        self._configure_input_shape(model_input.shape)
        self.tags = self._load_tags(Path(tags_path))

    def _configure_input_shape(self, shape: list[object]) -> None:
        if len(shape) != 4:
            return
        if shape[-1] == 3:
            self.channels_last = True
            if isinstance(shape[1], int):
                self.input_size = shape[1]
        elif shape[1] == 3:
            self.channels_last = False
            if isinstance(shape[2], int):
                self.input_size = shape[2]

    def _load_tags(self, tags_path: Path) -> list[WD14Tag]:
        tags: list[WD14Tag] = []
        with tags_path.open("r", encoding="utf-8", newline="") as handle:
            reader = csv.DictReader(handle)
            for row in reader:
                name = row.get("name", "").strip()
                category_text = row.get("category", "").strip()
                if not name or not category_text:
                    continue
                try:
                    category = int(category_text)
                except ValueError:
                    continue
                tags.append(WD14Tag(name=name, category=category))
        return tags

    def suggest(self, path: Path) -> list[str]:
        image_array = self._prepare_image(path)
        assert self.session is not None
        probabilities = self.session.run(None, {self.input_name: image_array})[0][0]

        tags: list[str] = []
        for tag, probability in zip(self.tags, probabilities):
            probability = float(probability)
            if tag.category == 9 and not self.options.include_ratings:
                continue
            if tag.category == 4:
                threshold = self.options.character_threshold
            else:
                threshold = self.options.threshold
            if probability < threshold:
                continue
            tags.append(format_wd14_tag(tag.name, self.options.keep_underscores))
        return tags

    def _prepare_image(self, path: Path):
        if Image is None:
            raise SystemExit("Pillow is required for WD14 mode.")

        with Image.open(path) as image:
            image = image.convert("RGBA")
            background = Image.new("RGBA", image.size, (255, 255, 255, 255))
            image = Image.alpha_composite(background, image).convert("RGB")

            width, height = image.size
            side = max(width, height)
            square = Image.new("RGB", (side, side), (255, 255, 255))
            square.paste(image, ((side - width) // 2, (side - height) // 2))
            square = square.resize((self.input_size, self.input_size), Image.Resampling.LANCZOS)

        array = self.np.asarray(square, dtype=self.np.float32)
        array = array[:, :, ::-1]  # WD taggers are commonly exported for BGR input.
        if self.channels_last:
            return array[None, :, :, :]
        return self.np.transpose(array, (2, 0, 1))[None, :, :, :]


def inspect_image(path: Path) -> ImageInfo:
    info = ImageInfo(path=path)
    if Image is None or ImageStat is None:
        return info

    try:
        with Image.open(path) as image:
            info.width, info.height = image.size
            info.mode = image.mode
            info.has_alpha = image.mode in {"LA", "RGBA"} or (
                image.mode == "P" and "transparency" in image.info
            )

            sample = image.convert("RGB")
            sample.thumbnail((128, 128))
            stat = ImageStat.Stat(sample)
            r, g, b = stat.mean[:3]
            info.brightness = (r + g + b) / 3.0
            info.warmth = r - b

            hsv = sample.convert("HSV")
            hsv_stat = ImageStat.Stat(hsv)
            info.saturation = hsv_stat.mean[1]
    except Exception as exc:
        print(f"WARN: cannot inspect {path}: {exc}", file=sys.stderr)

    return info


def property_tags(info: ImageInfo) -> list[str]:
    tags: list[str] = []

    if info.width and info.height:
        width = info.width
        height = info.height
        megapixels = (width * height) / 1_000_000

        if abs(width - height) <= max(width, height) * 0.05:
            tags.append("square image")
        elif height > width:
            tags.append("portrait")
        else:
            tags.append("landscape")

        if megapixels >= 2.0:
            tags.append("high resolution")
        elif megapixels < 0.5:
            tags.append("low resolution")

    if info.has_alpha:
        tags.append("transparent background")

    if info.brightness is not None:
        if info.brightness >= 185:
            tags.append("bright image")
        elif info.brightness <= 70:
            tags.append("dark image")

    if info.saturation is not None:
        if info.saturation <= 45:
            tags.append("muted colors")
        elif info.saturation >= 130:
            tags.append("vivid colors")

    if info.warmth is not None:
        if info.warmth >= 18:
            tags.append("warm tone")
        elif info.warmth <= -18:
            tags.append("cool tone")

    return tags


def discover_images(target: Path, recursive: bool) -> list[Path]:
    if target.is_file():
        if target.suffix.lower() in IMAGE_EXTENSIONS:
            return [target]
        raise SystemExit(f"Not a supported image file: {target}")

    if not target.is_dir():
        raise SystemExit(f"Path does not exist: {target}")

    iterator = target.rglob("*") if recursive else target.iterdir()
    return sorted(
        path
        for path in iterator
        if path.is_file() and path.suffix.lower() in IMAGE_EXTENSIONS
    )


def build_caption(
    path: Path,
    token: str | None,
    base_tags: list[str],
    wd14_tagger: WD14Tagger | None,
) -> str:
    info = inspect_image(path)
    tags = []
    if token:
        tags.append(token)
    tags.extend(base_tags)
    if wd14_tagger is not None:
        tags.extend(wd14_tagger.suggest(path))
    tags.extend(filename_tags(path))
    tags.extend(property_tags(info))
    return ", ".join(unique_tags(tags))


def read_existing(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8").strip()
    except FileNotFoundError:
        return ""


def merge_caption(existing: str, generated: str, mode: str) -> str:
    if mode == "append" and existing:
        return ", ".join(unique_tags(split_tags(existing) + split_tags(generated)))
    return generated


def review_caption(image_path: Path, caption: str) -> str | None:
    print()
    print(f"Image: {image_path}")
    print(f"Suggested: {caption}")
    answer = input("Add tags, Enter accept, !edit <caption>, !skip, !quit: ").strip()

    if not answer:
        return caption
    if answer == "!skip":
        return None
    if answer == "!quit":
        raise KeyboardInterrupt
    if answer.startswith("!edit "):
        edited = answer[len("!edit ") :].strip()
        return ", ".join(unique_tags(split_tags(edited)))

    return ", ".join(unique_tags(split_tags(caption) + split_tags(answer)))


def write_caption(path: Path, caption: str, dry_run: bool) -> None:
    if dry_run:
        print(f"DRY-RUN write {path}: {caption}")
        return
    path.write_text(caption + "\n", encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Create editable .txt image caption/tag files for LoRA training."
    )
    parser.add_argument(
        "target",
        nargs="?",
        help="Image file or image directory. If omitted, the script will ask.",
    )
    parser.add_argument(
        "--token",
        default="",
        help="Trigger token to put first, for example style_token_v1.",
    )
    parser.add_argument(
        "--base-tags",
        default="",
        help="Comma-separated tags to add to every caption.",
    )
    parser.add_argument(
        "--recursive",
        action="store_true",
        help="Scan image folders recursively.",
    )
    parser.add_argument(
        "--review",
        action="store_true",
        help="Review each caption interactively before writing.",
    )
    parser.add_argument(
        "--overwrite",
        action="store_true",
        help="Overwrite existing .txt caption files.",
    )
    parser.add_argument(
        "--append",
        action="store_true",
        help="Append generated tags to existing .txt caption files.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print actions without writing files.",
    )
    parser.add_argument(
        "--wd14",
        action="store_true",
        help="Use WD14 anime tagger suggestions from Hugging Face.",
    )
    parser.add_argument(
        "--wd14-model",
        default=DEFAULT_WD14_MODEL,
        help=f"WD14 Hugging Face model id. Default: {DEFAULT_WD14_MODEL}",
    )
    parser.add_argument(
        "--wd14-threshold",
        type=float,
        default=0.35,
        help="Minimum probability for WD14 general tags.",
    )
    parser.add_argument(
        "--wd14-character-threshold",
        type=float,
        default=0.85,
        help="Minimum probability for WD14 character tags.",
    )
    parser.add_argument(
        "--wd14-include-ratings",
        action="store_true",
        help="Include WD14 rating tags.",
    )
    parser.add_argument(
        "--wd14-keep-underscores",
        action="store_true",
        help="Keep Danbooru underscores instead of converting them to spaces.",
    )
    parser.add_argument(
        "--wd14-cache-dir",
        default=None,
        help="Optional Hugging Face cache directory for WD14 files.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.overwrite and args.append:
        raise SystemExit("Choose only one of --overwrite or --append.")

    target_text = args.target or input("Image file or folder: ").strip()
    target = Path(target_text).expanduser().resolve()
    token = normalize_tag(args.token)
    base_tags = split_tags(args.base_tags)
    mode = "append" if args.append else "write"
    wd14_tagger = None
    if args.wd14:
        wd14_tagger = WD14Tagger(
            WD14Options(
                enabled=True,
                model_id=args.wd14_model,
                threshold=args.wd14_threshold,
                character_threshold=args.wd14_character_threshold,
                include_ratings=args.wd14_include_ratings,
                keep_underscores=args.wd14_keep_underscores,
                cache_dir=args.wd14_cache_dir,
            )
        )

    images = discover_images(target, recursive=args.recursive)
    if not images:
        print("No supported images found.")
        return 0

    created = 0
    skipped = 0
    for image_path in images:
        caption_path = image_path.with_suffix(".txt")
        existing = read_existing(caption_path)
        if existing and not args.overwrite and not args.append:
            print(f"SKIP existing caption: {caption_path}")
            skipped += 1
            continue

        generated = build_caption(
            image_path,
            token=token,
            base_tags=base_tags,
            wd14_tagger=wd14_tagger,
        )
        caption = merge_caption(existing, generated, mode=mode)

        if args.review:
            try:
                reviewed = review_caption(image_path, caption)
            except KeyboardInterrupt:
                print("\nStopped.")
                break
            if reviewed is None:
                print(f"SKIP by review: {image_path}")
                skipped += 1
                continue
            caption = reviewed

        write_caption(caption_path, caption, dry_run=args.dry_run)
        if not args.dry_run:
            print(f"WRITE {caption_path}")
        created += 1

    print(f"Done. written={created} skipped={skipped} total_images={len(images)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
