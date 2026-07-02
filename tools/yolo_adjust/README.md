# YOLO Adjust

Local helper for reviewing and adjusting YOLO labels used by the visual-learning pipeline.

## Run

```bash
cd tools/yolo_adjust
python3 yolo_adjust_app.py
```

On macOS, you can also run:

```bash
./run_yolo_adjust.command
```

## Output Layout

```text
tools/yolo_adjust/dataset/
  images/train/page_001.png
  labels/train/page_001.txt
  meta/page_001.dom_candidates.json
  meta/page_001.review.json
  previews/page_001.preview.png
```

## Notes

Keep local datasets, screenshots, generated labels, checkpoints, and exported models out of Git unless they are intentionally sanitized release assets.
