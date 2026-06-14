# Shared Core vs Platform-Specific Layers

This project keeps product behavior shared while allowing macOS and Windows to
own their native runtime edges separately.

## Shared Core

Shared core code must compile and behave the same on macOS and Windows unless a
small build-tagged adapter is explicitly injected.

- Conversation, routing, and action-chain control flow:
  `data/conversation`, `shared/actionchain`, `orchestration/dag`,
  `orchestration/replan`, `orchestration/skill_flow`,
  `orchestration/skill_step`, `task_progress_binding.go`, and
  `task_loop_binding.go`.
- Memory and summarization:
  `data/memory`, `app_memory_expand.go`, summary/deep-memory wiring, and
  `RunSummarizationNow`.
- Built-in document/data tools:
  `builtin`, `dianliao_*`, `go_program_authoring_*`, and related tests.
- Web/local search and source trust:
  `shared/localsearch`, `shared/websearch`, `shared/safefetcher`,
  `domain/source_trust`, URL capability bindings, and egress tests.
- Frontend product UI and i18n:
  `frontend/src/locales`, reusable components, `frontend/src/lib/appShared.jsx`,
  chat/message behavior, settings behavior, and locale tests.
- Split app ownership:
  app-facing Go files should stay grouped by behavior, following the Windows
  split-file shape:
  `app_adapter.go`, `app_learning.go`, `app_model.go`, `app_search.go`,
  `app_skill.go`, `app_tool_routing.go`, plus narrow feature files such as
  `app_memory_expand.go`.

## Platform-Specific Layer

Platform code may differ, but must keep the same public app behavior and DTOs.

- macOS native input/capture:
  `*_darwin.go`, `*.m`, `*.h`, CoreGraphics/CoreML bridge code, and `.mlmodelc`
  packaging assumptions.
- Windows native input/capture:
  `*_windows.go`, Windows click-anchor capture/replay behavior, Windows drag
  behavior, and bundled Windows runtime assets.
- Linux fallback/native drag:
  `*_linux.go`, `*_gtk.go`, and non-macOS/non-Windows fallback files.
- Generated Wails bindings:
  `frontend/wailsjs` is generated per checkout/platform after Go method changes.
  Do not manually treat one platform's generated bindings as authoritative for
  the other platform.
- Packaged app/build outputs:
  `build`, `.test-appdata`, `.gocache`, runtime logs, and local generated
  outputs are not shared core.

## Merge Rules

- Prefer the split-file architecture from the Windows checkout for new app-level
  Go behavior.
- When porting features, move behavior into the shared core first, then add
  small platform adapters only where OS APIs differ.
- Never overwrite a platform-specific native file from another platform just
  because filenames look similar.
- After shared Go behavior changes, run platform-local tests and regenerate that
  platform's Wails bindings with its own build.
- Language display labels remain user-facing names such as `中文`, `English`, and
  `日文`; locale keys are implementation details.
