# Third-Party Notices

AI Console is licensed under the Apache License 2.0. It includes and/or depends
on the third-party open-source components listed below. We gratefully acknowledge
their authors and contributors. Each component remains under its own license.

This list is maintained by hand; the authoritative dependency sets are
`go.mod` / `go.sum` (backend) and `frontend/package-lock.json` (frontend).

All listed licenses are permissive (MIT / BSD / Apache-2.0 / ISC) except
`lightningcss`, which is MPL-2.0 — see the note at the end.

---

## Backend (Go) — direct & notable transitive dependencies

| Dependency | License |
|---|---|
| github.com/wailsapp/wails/v2 | MIT |
| github.com/wailsapp/go-webview2 | MIT |
| github.com/wailsapp/mimetype | MIT |
| github.com/go-ole/go-ole | MIT |
| github.com/google/uuid | BSD-3-Clause |
| github.com/gorilla/websocket | BSD-2-Clause |
| golang.org/x/net | BSD-3-Clause |
| golang.org/x/sys | BSD-3-Clause |
| golang.org/x/text | BSD-3-Clause |
| golang.org/x/crypto | BSD-3-Clause |
| github.com/labstack/echo/v4 | MIT |
| github.com/labstack/gommon | MIT |
| github.com/godbus/dbus/v5 | BSD-2-Clause |
| git.sr.ht/~jackmordaunt/go-toast/v2 | MIT |
| github.com/bep/debounce | MIT |
| github.com/jchv/go-winloader | MIT |
| github.com/leaanthony/go-ansi-parser | MIT |
| github.com/leaanthony/gosod | MIT |
| github.com/leaanthony/slicer | MIT |
| github.com/leaanthony/u | MIT |
| github.com/leaanthony/debme | MIT |
| github.com/mattn/go-colorable | MIT |
| github.com/mattn/go-isatty | MIT |
| github.com/pkg/browser | BSD-2-Clause |
| github.com/pkg/errors | BSD-2-Clause |
| github.com/rivo/uniseg | MIT |
| github.com/samber/lo | MIT |
| github.com/tkrajina/go-reflector | Apache-2.0 |
| github.com/valyala/bytebufferpool | MIT |
| github.com/valyala/fasttemplate | MIT |
| gopkg.in/yaml.v3 | MIT / Apache-2.0 |
| Go standard library | BSD-3-Clause |

> `go-webview2` loads the Microsoft WebView2 Runtime at runtime. The WebView2
> Runtime is a proprietary, redistributable Microsoft component governed by its
> own terms; it is not bundled in this repository.

---

## Frontend (JavaScript / TypeScript)

| Dependency | License |
|---|---|
| react | MIT |
| react-dom | MIT |
| zustand | MIT |
| vite | MIT |
| @vitejs/plugin-react | MIT |
| tailwindcss | MIT |
| @tailwindcss/cli | MIT |
| lightningcss (+ platform binaries) | **MPL-2.0** |
| @testing-library/react | MIT |
| @testing-library/jest-dom | MIT |
| @types/react, @types/react-dom | MIT |
| jsdom | MIT |
| vitest | MIT |

---

## Bundled / Optional ML Components

| Component | Role | License |
|---|---|---|
| YOLOX-S model (`assets/models/yolox_button_s.{onnx,mlmodelc}`) | UI element detection | Apache-2.0 |
| whisper.cpp `whisper-cli` runner | Speech-to-text (user-provided binary, not bundled) | MIT |
| OpenAI Whisper `ggml-base.bin` model | Speech-to-text weights | MIT |
| Kokoro voice pack | Text-to-speech (optional, not yet pinned) | Apache-2.0 |

> The project deliberately excludes copyleft ML components. The former
> Ultralytics YOLOv5 model (AGPL-3.0) has been removed and replaced with YOLOX
> (Apache-2.0). The build does not depend on espeak-ng or GPL-licensed Piper.

---

## Bundled Fonts (Font Presets — all SIL OFL 1.1)

These fonts back the panel "字體版型 / Font Preset" feature. They are **not**
committed to the repository; `scripts/fetch_fonts.sh` downloads them into
`frontend/public/fonts/` before build. See `FONT_SPEC.md` for the full mapping.
All are licensed under the SIL Open Font License 1.1, which permits royalty-free
embedding and redistribution provided the license is retained and the fonts are
not sold on their own.

| Font | Script | License | Upstream |
|---|---|---|---|
| Inter | Latin | OFL-1.1 | github.com/rsms/inter |
| Noto Sans TC / Noto Sans JP / Noto Serif TC | CJK | OFL-1.1 | github.com/notofonts |
| Caveat | Latin | OFL-1.1 | Google Fonts |
| Dancing Script | Latin | OFL-1.1 | Google Fonts |
| Fredoka | Latin | OFL-1.1 | Google Fonts |
| JetBrains Mono | Latin/mono | OFL-1.1 | github.com/JetBrains/JetBrainsMono |
| Klee One | JP | OFL-1.1 | Fontworks (Google Fonts) |
| Yuji Syuku | JP brush | OFL-1.1 | Fontworks (Google Fonts) |
| LXGW WenKai 霞鶩文楷 | CJK kai | OFL-1.1 | github.com/lxgw/LxgwWenKai |
| jf open 粉圓 (jf-openhuninn) | TC rounded | OFL-1.1 | github.com/justfont/open-huninn-font |

> The OFL 1.1 license text is reproduced in `frontend/src/assets/fonts/OFL.txt`
> and must ship with any binary distribution that bundles these fonts.

---

## Note on lightningcss (MPL-2.0)

`lightningcss` is a build-time CSS tool pulled in by Tailwind CSS v4. It is
licensed under the Mozilla Public License 2.0, a file-level weak copyleft:
obligations apply only if you modify lightningcss's own source files. This
project uses it unmodified and only at build time (it is not linked into the
shipped binary), so MPL-2.0 imposes no obligations on AI Console's own
Apache-2.0 source. The MPL-2.0 license text is available at
https://www.mozilla.org/MPL/2.0/.

---

## Notes

- This list covers core and notable dependencies. Transitive dependencies remain
  under their respective licenses as recorded in the lock files.
- If you believe a dependency is missing or misattributed, please open an issue.
