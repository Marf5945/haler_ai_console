#!/usr/bin/env python3
import json
import os
import shutil
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path
from tkinter import BOTH, END, LEFT, RIGHT, X, Y, Button, Canvas, Entry, Frame, Label, Listbox, Scrollbar, StringVar, TclError, Text, Tk, filedialog, messagebox
from tkinter import ttk

from PIL import Image, ImageDraw, ImageFont, ImageTk


APP_DIR = Path.home() / "Desktop" / "yolo_adjust"
DATASET_DIR = APP_DIR / "dataset"
IMAGES_DIR = DATASET_DIR / "images" / "train"
LABELS_DIR = DATASET_DIR / "labels" / "train"
META_DIR = DATASET_DIR / "meta"
PREVIEW_DIR = DATASET_DIR / "previews"
SCRIPT_PATH = APP_DIR / "dom_candidate_script.js"
SETTINGS_PATH = APP_DIR / "settings.json"
DATA_YAML_PATH = DATASET_DIR / "data.yaml"


@dataclass
class Candidate:
    id: str
    rect: list[float]
    class_guess: str = "button"
    confidence_rule: str = "high"
    source: str = "dom_auto"
    review_status: str = "accepted"
    text: str = ""
    tag: str = ""
    role: str = ""
    disabled: bool = False
    meta: dict = field(default_factory=dict)


class YoloAdjustApp:
    def __init__(self) -> None:
        self.root = Tk()
        self.root.title("YOLO Adjust - 半自動標註")
        self.root.geometry("860x720")
        self.root.minsize(700, 620)

        self.current_stem: str | None = None
        self.current_image: Path | None = None
        self.current_preview: Path | None = None
        self.candidates: list[Candidate] = []
        self.preview_tk = None

        self.browser = StringVar(value="Google Chrome")
        self.window_x = StringVar(value="100")
        self.window_y = StringVar(value="100")
        self.viewport_w = StringVar(value="1280")
        self.viewport_h = StringVar(value="720")
        self.top_offset = StringVar(value="88")
        self.left_offset = StringVar(value="0")
        self.dom_offset_x = StringVar(value="0")
        self.dom_offset_y = StringVar(value="0")
        self.dom_scale = StringVar(value="")
        self.status = StringVar(value="")
        self.current_page: dict = {}

        self.edit_id = StringVar(value="")
        self.edit_x = StringVar(value="")
        self.edit_y = StringVar(value="")
        self.edit_w = StringVar(value="")
        self.edit_h = StringVar(value="")

        self.ensure_dirs()
        self.load_settings()
        self.bind_settings_autosave()
        self.build_ui()
        self.refresh_yolo_status()
        self.root.protocol("WM_DELETE_WINDOW", self.on_close)

    def ensure_dirs(self) -> None:
        for path in [APP_DIR, IMAGES_DIR, LABELS_DIR, META_DIR, PREVIEW_DIR]:
            path.mkdir(parents=True, exist_ok=True)

    def settings_vars(self) -> dict[str, StringVar]:
        return {
            "browser": self.browser,
            "window_x": self.window_x,
            "window_y": self.window_y,
            "viewport_w": self.viewport_w,
            "viewport_h": self.viewport_h,
            "top_offset": self.top_offset,
            "left_offset": self.left_offset,
            "dom_offset_x": self.dom_offset_x,
            "dom_offset_y": self.dom_offset_y,
            "dom_scale": self.dom_scale,
        }

    def load_settings(self) -> None:
        if not SETTINGS_PATH.exists():
            return
        try:
            settings = json.loads(SETTINGS_PATH.read_text(encoding="utf-8"))
        except Exception:
            return
        for key, var in self.settings_vars().items():
            if key in settings:
                var.set(str(settings[key]))

    def save_settings(self, *_args) -> None:
        try:
            payload = {key: var.get() for key, var in self.settings_vars().items()}
            SETTINGS_PATH.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
        except Exception:
            pass

    def bind_settings_autosave(self) -> None:
        for var in self.settings_vars().values():
            var.trace_add("write", self.save_settings)

    def on_close(self) -> None:
        self.save_settings()
        self.root.destroy()

    def build_ui(self) -> None:
        outer = Frame(self.root, padx=12, pady=12)
        outer.pack(fill=BOTH, expand=True)

        header = Frame(outer)
        header.pack(fill=X)
        Label(header, text="YOLO Adjust", font=("Helvetica", 18, "bold")).pack(side=LEFT)
        Button(header, text="重新檢查 YOLO", command=self.refresh_yolo_status).pack(side=RIGHT)

        Label(outer, textvariable=self.status, anchor="w", justify=LEFT, fg="#24506d").pack(fill=X, pady=(6, 8))

        main = ttk.PanedWindow(outer, orient="horizontal")
        main.pack(fill=BOTH, expand=True)

        left = Frame(main, padx=6, pady=6, width=430)
        left.pack_propagate(False)
        right = Frame(main, padx=6, pady=6)
        main.add(left, weight=0)
        main.add(right, weight=1)

        self.build_scrollable_controls(left)
        self.build_preview(right)

    def build_scrollable_controls(self, parent: Frame) -> None:
        control_canvas = Canvas(parent, highlightthickness=0)
        scrollbar = Scrollbar(parent, orient="vertical", command=control_canvas.yview)
        scroll_frame = Frame(control_canvas)
        window_id = control_canvas.create_window((0, 0), window=scroll_frame, anchor="nw")
        control_canvas.configure(yscrollcommand=scrollbar.set)

        control_canvas.pack(side=LEFT, fill=BOTH, expand=True)
        scrollbar.pack(side=RIGHT, fill=Y)

        def update_scroll_region(_event=None) -> None:
            control_canvas.configure(scrollregion=control_canvas.bbox("all"))
            control_canvas.itemconfigure(window_id, width=control_canvas.winfo_width())

        def on_mousewheel(event) -> None:
            control_canvas.yview_scroll(int(-1 * (event.delta / 120)), "units")

        scroll_frame.bind("<Configure>", update_scroll_region)
        control_canvas.bind("<Configure>", update_scroll_region)
        control_canvas.bind("<Enter>", lambda _event: control_canvas.bind_all("<MouseWheel>", on_mousewheel))
        control_canvas.bind("<Leave>", lambda _event: control_canvas.unbind_all("<MouseWheel>"))
        self.build_controls(scroll_frame)

    def build_controls(self, parent: Frame) -> None:
        browser_frame = ttk.LabelFrame(parent, text="截圖")
        browser_frame.pack(fill=X)

        row = Frame(browser_frame)
        row.pack(fill=X, padx=8, pady=6)
        Label(row, text="瀏覽器").pack(side=LEFT)
        browser_combo = ttk.Combobox(
            row,
            textvariable=self.browser,
            values=["Google Chrome", "Safari", "Microsoft Edge", "Firefox"],
            width=18,
            state="readonly",
        )
        browser_combo.pack(side=RIGHT)

        grid = Frame(browser_frame)
        grid.pack(fill=X, padx=8)
        fields = [
            ("X", self.window_x),
            ("Y", self.window_y),
            ("寬", self.viewport_w),
            ("高", self.viewport_h),
            ("上偏移", self.top_offset),
            ("左偏移", self.left_offset),
            ("DOM X", self.dom_offset_x),
            ("DOM Y", self.dom_offset_y),
            ("Scale", self.dom_scale),
        ]
        for i, (label, var) in enumerate(fields):
            Label(grid, text=label).grid(row=i // 2, column=(i % 2) * 2, sticky="w", pady=2)
            Entry(grid, textvariable=var, width=8).grid(row=i // 2, column=(i % 2) * 2 + 1, sticky="w", padx=(4, 12), pady=2)

        Label(
            browser_frame,
            text="截圖前請把 DevTools 關掉，或改成獨立視窗。停靠在右側會一起被截進預覽圖。",
            anchor="w",
            justify=LEFT,
            fg="#8a5a00",
            wraplength=260,
        ).pack(fill=X, padx=8, pady=(6, 0))
        self.compact_button(browser_frame, "1. 調整視窗並截圖新編號", self.capture_screen)
        self.compact_button(browser_frame, "選擇 label 並套用", self.choose_label_file, pady=(0, 8))

        dom_frame = ttk.LabelFrame(parent, text="DOM 候選框 JSON")
        dom_frame.pack(fill=X, pady=(10, 0))
        self.compact_button(dom_frame, "2. 只複製 DOM 擷取腳本（不截圖）", self.copy_dom_script, pady=(8, 4))
        self.compact_button(dom_frame, "3. 從剪貼簿貼上 JSON", self.paste_dom_json, pady=(0, 4))
        self.dom_text = Text(dom_frame, width=44, height=8, wrap="word")
        self.dom_text.pack(fill=X, padx=8, pady=4)
        self.compact_button(dom_frame, "確定 / 產生預覽圖", self.confirm_preview, pady=(4, 8))

        edit_frame = ttk.LabelFrame(parent, text="修框")
        edit_frame.pack(fill=X, pady=(10, 0))
        for label, var in [("ID", self.edit_id), ("X", self.edit_x), ("Y", self.edit_y), ("W", self.edit_w), ("H", self.edit_h)]:
            row = Frame(edit_frame)
            row.pack(fill=X, padx=8, pady=1)
            Label(row, text=label, width=4, anchor="w").pack(side=LEFT)
            Entry(row, textvariable=var, width=22).pack(side=RIGHT)
        buttons = Frame(edit_frame)
        buttons.pack(fill=X, padx=8, pady=8)
        Button(buttons, text="更新", command=self.update_selected).pack(side=LEFT, fill=X, expand=True)
        Button(buttons, text="新增", command=self.add_manual_box).pack(side=LEFT, fill=X, expand=True, padx=4)
        Button(buttons, text="刪除", command=self.delete_selected).pack(side=LEFT, fill=X, expand=True)

        self.compact_button(parent, "輸出 YOLO labels", self.export_yolo_labels, pady=(10, 0), padx=0)

    def compact_button(self, parent: Frame, text: str, command, pady=8, padx=8) -> None:
        holder = Frame(parent)
        holder.pack(fill=X, padx=padx, pady=pady)
        Button(holder, text=text, command=command, width=24).pack(side=LEFT)

    def build_preview(self, parent: Frame) -> None:
        info = Frame(parent)
        info.pack(fill=X)
        Label(info, text=f"輸出資料夾：{APP_DIR}", anchor="w").pack(side=LEFT)
        Button(info, text="打開資料夾", command=self.open_app_dir).pack(side=RIGHT)

        list_frame = Frame(parent)
        list_frame.pack(fill=X, pady=(8, 8))
        columns = ("id", "accepted", "rule", "rect", "text")
        self.tree = ttk.Treeview(list_frame, columns=columns, show="headings", height=6)
        for col, width in [("id", 95), ("accepted", 72), ("rule", 70), ("rect", 160), ("text", 380)]:
            self.tree.heading(col, text=col)
            self.tree.column(col, width=width, anchor="w")
        self.tree.pack(side=LEFT, fill=X, expand=True)
        self.tree.bind("<<TreeviewSelect>>", self.on_select_candidate)
        Button(list_frame, text="接受/取消", command=self.toggle_selected).pack(side=RIGHT, fill=Y, padx=(6, 0))

        canvas_frame = ttk.LabelFrame(parent, text="預覽")
        canvas_frame.pack(fill=BOTH, expand=True)
        self.canvas = Canvas(canvas_frame, bg="#f2f4f5")
        self.canvas.pack(fill=BOTH, expand=True)

    def refresh_yolo_status(self) -> None:
        # 本工具只負責「半自動標註」並輸出 YOLO txt 格式（images/ + labels/ + data.yaml）。
        # 不綁定任何訓練框架。訓練請改用 YOLOX 官方 repo（Apache-2.0）。
        # 重要：請勿用 Ultralytics 的 yolo CLI / ultralytics 套件來訓練，
        #       那會讓匯出的權重落入 AGPL-3.0，與本專案 Apache-2.0 授權衝突。
        n_images = len(list(IMAGES_DIR.glob("*.png"))) if IMAGES_DIR.exists() else 0
        dataset_state = "資料集就緒" if (IMAGES_DIR.exists() and LABELS_DIR.exists()) else "資料集尚未建立"
        self.status.set(
            "標註工具（框架無關，輸出 YOLO txt 格式）\n"
            f"{dataset_state}：已標註影像 {n_images} 張\n"
            "訓練請用 YOLOX(Apache-2.0)，勿用 ultralytics\n"
            f"資料會存到：{APP_DIR}"
        )

    def next_stem(self) -> str:
        existing = sorted(IMAGES_DIR.glob("page_*.png"))
        max_id = 0
        for path in existing:
            try:
                max_id = max(max_id, int(path.stem.split("_")[-1]))
            except ValueError:
                pass
        return f"page_{max_id + 1:03d}"

    def int_field(self, var: StringVar, name: str) -> int:
        try:
            return int(var.get())
        except ValueError as exc:
            raise ValueError(f"{name} 必須是整數") from exc

    def capture_screen(self) -> None:
        try:
            x = self.int_field(self.window_x, "X")
            y = self.int_field(self.window_y, "Y")
            vw = self.int_field(self.viewport_w, "寬")
            vh = self.int_field(self.viewport_h, "高")
            top = self.int_field(self.top_offset, "上偏移")
            left = self.int_field(self.left_offset, "左偏移")
        except ValueError as exc:
            messagebox.showerror("欄位錯誤", str(exc))
            return

        stem = self.next_stem()
        image_path = IMAGES_DIR / f"{stem}.png"
        outer_w = vw + left
        outer_h = vh + top
        app_name = self.browser.get()

        script = f'''
tell application "{app_name}"
  activate
  if (count of windows) is 0 then error "沒有可用視窗"
  set bounds of front window to {{{x}, {y}, {x + outer_w}, {y + outer_h}}}
end tell
delay 1.8
'''
        try:
            subprocess.run(["osascript", "-e", script], check=True, text=True, capture_output=True, timeout=10)
            region = f"{x + left},{y + top},{vw},{vh}"
            subprocess.run(["screencapture", "-x", "-R", region, str(image_path)], check=True, timeout=10)
        except subprocess.CalledProcessError as exc:
            messagebox.showerror("截圖失敗", exc.stderr or str(exc))
            return
        except Exception as exc:
            messagebox.showerror("截圖失敗", str(exc))
            return

        self.current_stem = stem
        self.current_image = image_path
        self.current_page = {}
        self.candidates = []
        self.refresh_tree()
        self.show_image(image_path)
        self.status.set(f"{self.status.get()}\n目前截圖：{image_path.name}")

    def copy_dom_script(self) -> None:
        if not SCRIPT_PATH.exists():
            messagebox.showerror("找不到腳本", f"缺少 {SCRIPT_PATH}")
            return
        text = SCRIPT_PATH.read_text(encoding="utf-8")
        self.root.clipboard_clear()
        self.root.clipboard_append(text)
        messagebox.showinfo("已複製", "請到瀏覽器 DevTools Console 貼上執行。結果會印出 JSON，也會嘗試自動 copy。")

    def paste_dom_json(self) -> None:
        try:
            text = self.root.clipboard_get()
        except TclError:
            messagebox.showerror("剪貼簿沒有文字", "請先在 Chrome Console 執行 copy(__YOLO_DOM_JSON__)。")
            return

        if not text.strip():
            messagebox.showerror("剪貼簿是空的", "請先在 Chrome Console 執行 copy(__YOLO_DOM_JSON__)。")
            return

        self.dom_text.delete("1.0", END)
        self.dom_text.insert("1.0", text)

        try:
            json.loads(self.extract_json_text(text))
            messagebox.showinfo("已貼上", "已從剪貼簿貼上完整 JSON，可以按「確定 / 產生預覽圖」。")
        except Exception:
            messagebox.showwarning("已貼上，但可能不是 JSON", "內容已貼上，但不像完整 JSON。請確認 Chrome Console 已執行 copy(__YOLO_DOM_JSON__)。")

    def parse_candidates(self, raw: str) -> tuple[dict, list[Candidate]]:
        data = json.loads(self.extract_json_text(raw))
        page = {}
        items = data
        if isinstance(data, dict):
            page = data.get("page", {})
            items = data.get("candidates", data.get("annotations", []))
        if not isinstance(items, list):
            raise ValueError("DOM 內容需要是 JSON array，或包含 candidates 的 JSON object")

        candidates = []
        for index, item in enumerate(items, start=1):
            if not isinstance(item, dict):
                continue
            rect = item.get("rect") or item.get("box")
            if isinstance(rect, dict):
                rect = [rect.get("x", rect.get("left", 0)), rect.get("y", rect.get("top", 0)), rect.get("width", 0), rect.get("height", 0)]
            if not isinstance(rect, list) or len(rect) < 4:
                continue
            x, y, w, h = [float(v) for v in rect[:4]]
            if w < 4 or h < 4:
                continue
            confidence_rule = str(item.get("confidence_rule") or "low")
            candidates.append(
                Candidate(
                    id=str(item.get("id") or f"dom_{index:03d}"),
                    rect=[x, y, w, h],
                    class_guess=str(item.get("class_guess") or item.get("class") or "button"),
                    confidence_rule=confidence_rule,
                    source=str(item.get("source") or "dom_auto"),
                    review_status="accepted" if item.get("visible", True) and confidence_rule == "high" else "pending",
                    text=str(item.get("text") or "")[:120],
                    tag=str(item.get("tag") or ""),
                    role=str(item.get("role") or ""),
                    disabled=bool(item.get("disabled", False)),
                    meta=item,
                )
            )
        return page, candidates

    def extract_json_text(self, raw: str) -> str:
        text = raw.strip()
        if text.startswith("```"):
            lines = text.splitlines()
            if len(lines) >= 3:
                text = "\n".join(lines[1:-1]).strip()

        if text.startswith("{") or text.startswith("["):
            return text

        starts = [pos for pos in (text.find("{"), text.find("[")) if pos >= 0]
        ends = [pos for pos in (text.rfind("}"), text.rfind("]")) if pos >= 0]
        if starts and ends:
            start = min(starts)
            end = max(ends)
            if end > start:
                return text[start : end + 1]

        raise ValueError("貼上的內容不是完整 JSON。請從 Console 複製整段輸出，內容開頭應該是 { 或 [。")

    def confirm_preview(self) -> None:
        if not self.current_image or not self.current_stem:
            messagebox.showwarning("還沒有截圖", "請先按「調整視窗並截圖編號」。")
            return

        raw = self.dom_text.get("1.0", END).strip()
        page = {}
        if raw:
            try:
                page, self.candidates = self.parse_candidates(raw)
                self.current_page = page
            except Exception as exc:
                messagebox.showerror("DOM JSON 解析失敗", str(exc))
                return
        elif self.candidates:
            page = self.current_page
        else:
            messagebox.showwarning("沒有候選框", "請先貼上 DOM JSON、選擇 label，或手動新增框。")
            return

        self.save_meta(page)
        self.make_preview()
        self.refresh_tree()

    def save_meta(self, page: dict) -> None:
        assert self.current_stem is not None
        if not page and self.current_page:
            page = self.current_page
        raw_path = META_DIR / f"{self.current_stem}.dom_candidates.json"
        review_path = META_DIR / f"{self.current_stem}.review.json"
        payload = {
            "page": page,
            "image": str(self.current_image),
            "candidates": [self.candidate_to_dict(c) for c in self.candidates],
        }
        raw_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
        review_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")

    def candidate_to_dict(self, candidate: Candidate) -> dict:
        data = dict(candidate.meta)
        data.update(
            {
                "id": candidate.id,
                "rect": candidate.rect,
                "class_guess": candidate.class_guess,
                "confidence_rule": candidate.confidence_rule,
                "source": candidate.source,
                "review_status": candidate.review_status,
                "text": candidate.text,
                "tag": candidate.tag,
                "role": candidate.role,
                "disabled": candidate.disabled,
            }
        )
        return data

    def make_preview(self) -> None:
        assert self.current_image is not None
        assert self.current_stem is not None
        image = Image.open(self.current_image).convert("RGB")
        draw = ImageDraw.Draw(image)
        iw, ih = image.size
        font = ImageFont.load_default()

        for candidate in self.candidates:
            if candidate.review_status != "accepted":
                continue
            x, y, w, h = self.candidate_rect_to_image(candidate, iw, ih)
            color = "#22a06b" if candidate.confidence_rule == "high" else "#d99a00"
            if candidate.source == "human":
                color = "#d64545"
            if candidate.source == "label_import":
                color = "#7c5cff"
            xy = [x, y, x + w, y + h]
            draw.rectangle(xy, outline=color, width=3)
            label = f"{candidate.id} {candidate.class_guess}"
            text_box = draw.textbbox((xy[0], xy[1]), label, font=font)
            draw.rectangle([text_box[0] - 2, text_box[1] - 2, text_box[2] + 2, text_box[3] + 2], fill=color)
            draw.text((xy[0], xy[1]), label, fill="white", font=font)

        preview_path = PREVIEW_DIR / f"{self.current_stem}.preview.png"
        image.save(preview_path)
        self.current_preview = preview_path
        self.show_image(preview_path)
        messagebox.showinfo("完成", f"已產生預覽圖：{preview_path.name}")

    def dom_to_image_transform(self, image_width: int, image_height: int) -> tuple[float, float, float, float, float]:
        viewport = self.current_page.get("viewport", {}) if self.current_page else {}
        try:
            vw = float(viewport.get("width") or self.viewport_w.get() or image_width)
        except (TypeError, ValueError):
            vw = float(image_width)
        try:
            vh = float(viewport.get("height") or self.viewport_h.get() or image_height)
        except (TypeError, ValueError):
            vh = float(image_height)
        try:
            ox = float(self.dom_offset_x.get() or 0)
            oy = float(self.dom_offset_y.get() or 0)
        except ValueError:
            ox = 0
            oy = 0

        # DOM rects use CSS pixels. The screenshot may include extra vertical UI
        # like the macOS Dock, so use the horizontal CSS-pixel scale for both axes.
        try:
            scale = float(self.dom_scale.get()) if self.dom_scale.get().strip() else image_width / vw
        except ValueError:
            scale = image_width / vw if vw else 1.0
        return scale, ox, oy, vw, vh

    def candidate_rect_to_image(self, candidate: Candidate, image_width: int, image_height: int) -> list[float]:
        x, y, w, h = candidate.rect
        if candidate.meta.get("coordinate_space") == "image":
            _scale, ox, oy, _vw, _vh = self.dom_to_image_transform(image_width, image_height)
            return [x + ox, y + oy, w, h]
        scale, ox, oy, _vw, _vh = self.dom_to_image_transform(image_width, image_height)
        return [x * scale + ox, y * scale + oy, w * scale, h * scale]

    def show_image(self, path: Path) -> None:
        image = Image.open(path)
        self.canvas.update_idletasks()
        max_w = max(self.canvas.winfo_width(), 800)
        max_h = max(self.canvas.winfo_height(), 500)
        image.thumbnail((max_w - 24, max_h - 24))
        self.preview_tk = ImageTk.PhotoImage(image)
        self.canvas.delete("all")
        self.canvas.create_image(12, 12, anchor="nw", image=self.preview_tk)

    def refresh_tree(self) -> None:
        for item in self.tree.get_children():
            self.tree.delete(item)
        for candidate in self.candidates:
            rect = ", ".join(str(round(v, 1)) for v in candidate.rect)
            accepted = "yes" if candidate.review_status == "accepted" else "no"
            self.tree.insert("", END, iid=candidate.id, values=(candidate.id, accepted, candidate.confidence_rule, rect, candidate.text))

    def selected_candidate(self) -> Candidate | None:
        selected = self.tree.selection()
        if not selected:
            return None
        selected_id = selected[0]
        return next((c for c in self.candidates if c.id == selected_id), None)

    def on_select_candidate(self, _event=None) -> None:
        candidate = self.selected_candidate()
        if not candidate:
            return
        x, y, w, h = candidate.rect
        self.edit_id.set(candidate.id)
        self.edit_x.set(str(x))
        self.edit_y.set(str(y))
        self.edit_w.set(str(w))
        self.edit_h.set(str(h))

    def toggle_selected(self) -> None:
        candidate = self.selected_candidate()
        if not candidate:
            return
        candidate.review_status = "pending" if candidate.review_status == "accepted" else "accepted"
        self.refresh_tree()
        self.make_preview()

    def update_selected(self) -> None:
        candidate = self.selected_candidate()
        if not candidate:
            messagebox.showwarning("沒有選取", "請先選一個框。")
            return
        try:
            candidate.rect = [float(self.edit_x.get()), float(self.edit_y.get()), float(self.edit_w.get()), float(self.edit_h.get())]
        except ValueError:
            messagebox.showerror("欄位錯誤", "X/Y/W/H 必須是數字")
            return
        candidate.source = "human"
        candidate.review_status = "accepted"
        self.refresh_tree()
        self.make_preview()

    def add_manual_box(self) -> None:
        try:
            rect = [float(self.edit_x.get()), float(self.edit_y.get()), float(self.edit_w.get()), float(self.edit_h.get())]
        except ValueError:
            messagebox.showerror("欄位錯誤", "X/Y/W/H 必須是數字")
            return
        new_id = self.edit_id.get().strip() or f"human_{len(self.candidates) + 1:03d}"
        if any(c.id == new_id for c in self.candidates):
            new_id = f"{new_id}_{len(self.candidates) + 1:03d}"
        self.candidates.append(Candidate(id=new_id, rect=rect, source="human", confidence_rule="human", review_status="accepted"))
        self.refresh_tree()
        self.make_preview()

    def delete_selected(self) -> None:
        candidate = self.selected_candidate()
        if not candidate:
            return
        self.candidates = [c for c in self.candidates if c.id != candidate.id]
        self.refresh_tree()
        self.make_preview()

    def choose_label_file(self) -> None:
        if not self.current_image or not self.current_stem:
            messagebox.showwarning("還沒有截圖", "請先截圖，再選擇 label。")
            return

        selected = filedialog.askopenfilename(
            title="選擇 YOLO label",
            initialdir=str(LABELS_DIR),
            filetypes=[("YOLO label", "*.txt"), ("All files", "*.*")],
        )
        if not selected:
            return
        self.import_label_file(Path(selected))

    def import_label_file(self, label_path: Path) -> None:
        if not self.current_image or not self.current_stem:
            messagebox.showwarning("還沒有截圖", "請先截圖，再選擇 label。")
            return
        if not label_path.exists():
            messagebox.showwarning("找不到 label", str(label_path))
            return

        raw_lines = [line.strip() for line in label_path.read_text(encoding="utf-8").splitlines() if line.strip()]
        image = Image.open(self.current_image)
        iw, ih = image.size
        imported: list[Candidate] = []
        output_lines: list[str] = []

        for index, line in enumerate(raw_lines, start=1):
            parts = line.split()
            if len(parts) != 5:
                continue
            try:
                class_id = int(float(parts[0]))
                x_center, y_center, box_w, box_h = [float(value) for value in parts[1:]]
            except ValueError:
                continue

            w = box_w * iw
            h = box_h * ih
            x = (x_center - box_w / 2) * iw
            y = (y_center - box_h / 2) * ih
            imported.append(
                Candidate(
                    id=f"label_{index:03d}",
                    rect=[x, y, w, h],
                    class_guess="button",
                    confidence_rule="label",
                    source="label_import",
                    review_status="accepted",
                    text=f"from {label_path.name}",
                    meta={"coordinate_space": "image", "class_id": class_id, "imported_from": str(label_path)},
                )
            )
            output_lines.append(f"{class_id} {x_center:.6f} {y_center:.6f} {box_w:.6f} {box_h:.6f}")

        if not imported:
            messagebox.showwarning("沒有可匯入的框", f"{label_path.name} 沒有有效 YOLO label。")
            return

        self.candidates = imported
        current_label = LABELS_DIR / f"{self.current_stem}.txt"
        current_label.write_text("\n".join(output_lines) + "\n", encoding="utf-8")
        self.write_data_yaml()
        self.current_page = {"label_imported_from": str(label_path)}
        self.save_meta(self.current_page)
        self.refresh_tree()
        self.make_preview()
        messagebox.showinfo("已匯入", f"已匯入 {label_path.name} 到 {current_label.name}\n並已重產 data.yaml。")

    def write_data_yaml(self) -> None:
        DATA_YAML_PATH.write_text(
            "\n".join(
                [
                    f"path: {DATASET_DIR}",
                    "train: images/train",
                    "val: images/train",
                    "names:",
                    "  0: button",
                    "",
                ]
            ),
            encoding="utf-8",
        )

    def export_yolo_labels(self) -> None:
        if not self.current_image or not self.current_stem:
            messagebox.showwarning("還沒有截圖", "請先截圖。")
            return
        image = Image.open(self.current_image)
        iw, ih = image.size
        lines = []
        accepted = [c for c in self.candidates if c.review_status == "accepted"]
        for candidate in accepted:
            px, py, pw, ph = self.candidate_rect_to_image(candidate, iw, ih)
            x_center = (px + pw / 2) / iw
            y_center = (py + ph / 2) / ih
            yolo_w = pw / iw
            yolo_h = ph / ih
            lines.append(f"0 {x_center:.6f} {y_center:.6f} {yolo_w:.6f} {yolo_h:.6f}")
        label_path = LABELS_DIR / f"{self.current_stem}.txt"
        label_path.write_text("\n".join(lines) + ("\n" if lines else ""), encoding="utf-8")
        self.write_data_yaml()
        self.save_meta({})
        messagebox.showinfo("已輸出", f"YOLO label：{label_path}\ndata.yaml：{DATA_YAML_PATH}\naccepted boxes：{len(accepted)}")

    def open_app_dir(self) -> None:
        subprocess.run(["open", str(APP_DIR)], check=False)

    def run(self) -> None:
        self.root.mainloop()


if __name__ == "__main__":
    try:
        YoloAdjustApp().run()
    except KeyboardInterrupt:
        sys.exit(0)
