#!/usr/bin/env python3
"""Small Tkinter GUI for caption_tagger.py."""

from __future__ import annotations

import queue
import threading
import tkinter as tk
from pathlib import Path
from tkinter import filedialog, messagebox, scrolledtext, ttk

from caption_tagger import (
    DEFAULT_WD14_MODEL,
    WD14Options,
    WD14Tagger,
    build_caption,
    discover_images,
    merge_caption,
    normalize_tag,
    read_existing,
    split_tags,
    write_caption,
)


class CaptionTaggerApp:
    def __init__(self, root: tk.Tk) -> None:
        self.root = root
        self.root.title("Image Caption Tagger")
        self.root.geometry("820x680")
        self.root.minsize(760, 600)

        self.log_queue: queue.Queue[str] = queue.Queue()
        self.worker: threading.Thread | None = None

        self.folder_var = tk.StringVar()
        self.token_var = tk.StringVar(value="my_style_token")
        self.base_tags_var = tk.StringVar()
        self.model_var = tk.StringVar(value=DEFAULT_WD14_MODEL)
        self.threshold_var = tk.StringVar(value="0.35")
        self.character_threshold_var = tk.StringVar(value="0.85")

        self.wd14_var = tk.BooleanVar(value=True)
        self.recursive_var = tk.BooleanVar(value=False)
        self.overwrite_var = tk.BooleanVar(value=False)
        self.append_var = tk.BooleanVar(value=False)
        self.include_ratings_var = tk.BooleanVar(value=False)
        self.keep_underscores_var = tk.BooleanVar(value=False)
        self.dry_run_var = tk.BooleanVar(value=False)

        self._build_ui()
        self._poll_log()

    def _build_ui(self) -> None:
        outer = ttk.Frame(self.root, padding=16)
        outer.pack(fill=tk.BOTH, expand=True)
        outer.columnconfigure(0, weight=1)
        outer.rowconfigure(5, weight=1)

        folder_frame = ttk.LabelFrame(outer, text="圖片資料夾", padding=12)
        folder_frame.grid(row=0, column=0, sticky="ew")
        folder_frame.columnconfigure(0, weight=1)

        folder_entry = ttk.Entry(folder_frame, textvariable=self.folder_var)
        folder_entry.grid(row=0, column=0, sticky="ew", padx=(0, 8))
        ttk.Button(folder_frame, text="選擇資料夾", command=self.choose_folder).grid(
            row=0, column=1
        )

        caption_frame = ttk.LabelFrame(outer, text="標註設定", padding=12)
        caption_frame.grid(row=1, column=0, sticky="ew", pady=(12, 0))
        caption_frame.columnconfigure(1, weight=1)

        ttk.Label(caption_frame, text="觸發詞").grid(row=0, column=0, sticky="w")
        ttk.Entry(caption_frame, textvariable=self.token_var).grid(
            row=0, column=1, sticky="ew", padx=(8, 0)
        )

        ttk.Label(caption_frame, text="固定 tags").grid(
            row=1, column=0, sticky="w", pady=(8, 0)
        )
        ttk.Entry(caption_frame, textvariable=self.base_tags_var).grid(
            row=1, column=1, sticky="ew", padx=(8, 0), pady=(8, 0)
        )

        options_frame = ttk.LabelFrame(outer, text="選項", padding=12)
        options_frame.grid(row=2, column=0, sticky="ew", pady=(12, 0))
        options_frame.columnconfigure(0, weight=1)
        options_frame.columnconfigure(1, weight=1)
        options_frame.columnconfigure(2, weight=1)

        ttk.Checkbutton(options_frame, text="使用 WD14", variable=self.wd14_var).grid(
            row=0, column=0, sticky="w"
        )
        ttk.Checkbutton(options_frame, text="包含子資料夾", variable=self.recursive_var).grid(
            row=0, column=1, sticky="w"
        )
        ttk.Checkbutton(options_frame, text="Dry run 不寫檔", variable=self.dry_run_var).grid(
            row=0, column=2, sticky="w"
        )
        ttk.Checkbutton(options_frame, text="覆蓋既有 txt", variable=self.overwrite_var).grid(
            row=1, column=0, sticky="w", pady=(8, 0)
        )
        ttk.Checkbutton(options_frame, text="追加到既有 txt", variable=self.append_var).grid(
            row=1, column=1, sticky="w", pady=(8, 0)
        )
        ttk.Checkbutton(
            options_frame,
            text="保留底線 long_hair",
            variable=self.keep_underscores_var,
        ).grid(row=1, column=2, sticky="w", pady=(8, 0))

        wd14_frame = ttk.LabelFrame(outer, text="WD14 進階", padding=12)
        wd14_frame.grid(row=3, column=0, sticky="ew", pady=(12, 0))
        wd14_frame.columnconfigure(1, weight=1)

        ttk.Label(wd14_frame, text="模型").grid(row=0, column=0, sticky="w")
        ttk.Entry(wd14_frame, textvariable=self.model_var).grid(
            row=0, column=1, columnspan=3, sticky="ew", padx=(8, 0)
        )

        ttk.Label(wd14_frame, text="一般 threshold").grid(
            row=1, column=0, sticky="w", pady=(8, 0)
        )
        ttk.Entry(wd14_frame, textvariable=self.threshold_var, width=8).grid(
            row=1, column=1, sticky="w", padx=(8, 24), pady=(8, 0)
        )
        ttk.Label(wd14_frame, text="角色 threshold").grid(
            row=1, column=2, sticky="w", pady=(8, 0)
        )
        ttk.Entry(wd14_frame, textvariable=self.character_threshold_var, width=8).grid(
            row=1, column=3, sticky="w", padx=(8, 0), pady=(8, 0)
        )
        ttk.Checkbutton(
            wd14_frame,
            text="包含 rating tags",
            variable=self.include_ratings_var,
        ).grid(row=2, column=0, columnspan=2, sticky="w", pady=(8, 0))

        button_frame = ttk.Frame(outer)
        button_frame.grid(row=4, column=0, sticky="ew", pady=(14, 8))
        button_frame.columnconfigure(0, weight=1)

        self.run_button = ttk.Button(
            button_frame, text="產生標註檔", command=self.start_generation
        )
        self.run_button.grid(row=0, column=1, sticky="e")

        self.log_text = scrolledtext.ScrolledText(outer, height=16, wrap=tk.WORD)
        self.log_text.grid(row=5, column=0, sticky="nsew")
        self.log_text.configure(state=tk.DISABLED)

    def choose_folder(self) -> None:
        selected = filedialog.askdirectory(title="選擇圖片資料夾")
        if selected:
            self.folder_var.set(selected)

    def start_generation(self) -> None:
        if self.worker and self.worker.is_alive():
            return

        folder = self.folder_var.get().strip()
        if not folder:
            messagebox.showerror("缺少資料夾", "請先選擇圖片資料夾。")
            return
        if self.overwrite_var.get() and self.append_var.get():
            messagebox.showerror("選項衝突", "覆蓋和追加只能選一個。")
            return

        try:
            threshold = float(self.threshold_var.get())
            character_threshold = float(self.character_threshold_var.get())
        except ValueError:
            messagebox.showerror("數值錯誤", "threshold 必須是數字。")
            return

        self.log_text.configure(state=tk.NORMAL)
        self.log_text.delete("1.0", tk.END)
        self.log_text.configure(state=tk.DISABLED)
        self.run_button.configure(state=tk.DISABLED)

        self.worker = threading.Thread(
            target=self._run_generation,
            kwargs={
                "folder": folder,
                "threshold": threshold,
                "character_threshold": character_threshold,
            },
            daemon=True,
        )
        self.worker.start()

    def _run_generation(
        self, folder: str, threshold: float, character_threshold: float
    ) -> None:
        try:
            target = Path(folder).expanduser().resolve()
            token = normalize_tag(self.token_var.get())
            base_tags = split_tags(self.base_tags_var.get())
            mode = "append" if self.append_var.get() else "write"

            self._log(f"Scanning: {target}")
            images = discover_images(target, recursive=self.recursive_var.get())
            if not images:
                self._log("No supported images found.")
                return

            wd14_tagger = None
            if self.wd14_var.get():
                self._log("Loading WD14 model. First run may download model files.")
                wd14_tagger = WD14Tagger(
                    WD14Options(
                        enabled=True,
                        model_id=self.model_var.get().strip() or DEFAULT_WD14_MODEL,
                        threshold=threshold,
                        character_threshold=character_threshold,
                        include_ratings=self.include_ratings_var.get(),
                        keep_underscores=self.keep_underscores_var.get(),
                        cache_dir=str(Path(__file__).resolve().parent / ".wd14-cache"),
                    )
                )

            written = 0
            skipped = 0
            for index, image_path in enumerate(images, start=1):
                caption_path = image_path.with_suffix(".txt")
                existing = read_existing(caption_path)
                if existing and not self.overwrite_var.get() and not self.append_var.get():
                    skipped += 1
                    self._log(f"[{index}/{len(images)}] SKIP existing: {caption_path.name}")
                    continue

                generated = build_caption(
                    image_path,
                    token=token,
                    base_tags=base_tags,
                    wd14_tagger=wd14_tagger,
                )
                caption = merge_caption(existing, generated, mode=mode)
                write_caption(caption_path, caption, dry_run=self.dry_run_var.get())
                written += 1
                action = "DRY-RUN" if self.dry_run_var.get() else "WRITE"
                self._log(f"[{index}/{len(images)}] {action}: {caption_path.name}")

            self._log(f"Done. written={written} skipped={skipped} total_images={len(images)}")
        except Exception as exc:  # GUI boundary: report instead of crashing silently.
            self._log(f"ERROR: {exc}")
        finally:
            self.log_queue.put("__DONE__")

    def _log(self, message: str) -> None:
        self.log_queue.put(message)

    def _poll_log(self) -> None:
        while True:
            try:
                message = self.log_queue.get_nowait()
            except queue.Empty:
                break
            if message == "__DONE__":
                self.run_button.configure(state=tk.NORMAL)
                continue
            self.log_text.configure(state=tk.NORMAL)
            self.log_text.insert(tk.END, message + "\n")
            self.log_text.see(tk.END)
            self.log_text.configure(state=tk.DISABLED)
        self.root.after(100, self._poll_log)


def main() -> int:
    root = tk.Tk()
    CaptionTaggerApp(root)
    root.mainloop()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
