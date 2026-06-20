# App structure notes

This folder owns the top-level React application shell.

## Current shape

- `../App.jsx` is only a compatibility re-export so existing imports keep working.
- `App.jsx` contains the current application shell, state, effects, handlers, and several top-level UI sections.
- App-local dialogs and modal components live under `components/`.
- Feature-scoped components live under `../features/`, starting with `onboarding/` and `readiness-gate/`.
- Cross-cutting frontend helpers that are shared by `app/` and `features/` live under `../lib/`.
- Existing generic leaf components still live under `../components/`.
- Global CSS stays at `../style.css`, `../tailwind.css`, and component-specific CSS stays next to its component.

## Dependency rules for future splits

Move code in this order:

1. Pure display components that already receive all data through props.
2. Dialogs and panels whose local `useState` does not need App-level state.
3. Domain hooks that own one feature area, such as voice, onboarding, scheduler, reference files, or tool flows.
4. App-level orchestration only after the above boundaries are stable.

Before moving a component out of `App.jsx`, check for:

- Wails bindings imported from `../../wailsjs/go/main/App` or `../../wailsjs/runtime/runtime`.
- Module-level helpers or constants in `App.jsx`, especially i18n helpers, style helpers, persona/avatar helpers, and reference-file helpers.
- CSS class names that are defined in the global stylesheets.
- Dynamic imports and `new URL(..., import.meta.url)` asset references, because relative paths change when files move.
- Closure dependencies on App state, setters, refs, or handlers. If a component reads these directly, pass them as props first or keep it in `App.jsx`.

## Safe next extraction candidates

Already extracted:

- `components/`: app-local dialog, modal, toast, and confirmation components.
- `components/`: app-local tool popup, project management, package confirmation, session-close, avatar upload, recording catalog, and Go program catalog components.
- `../features/onboarding/`: first-run onboarding overlay.
- `../features/readiness-gate/`: floating candidates, confirmation tiers, retrieval transparency, and long-press confirmation UI.
- `../lib/`: shared Wails-call, external-link, DAG/tool-tab, and small helper utilities.

Good next targets:

- `Sidebar`
- `SkillActivityCard`
- `SkillFirstUseCard`
- `DraftSandboxStopDialog`
- `TrustedSessionExpiredDialog`
- `StateTokenLegend`
- `SettingSelect`
- `SettingPopupSelect`

Large sections that need a dependency pass before moving:

- `TopConsole`
- `ConversationPanel`
- `ReviewPanel`
- `RightRail`
- `SchedulerPanel`
- `SettingsMenu` / `SettingsWorkspace`
- `PersonaSettingsDrawer`
- `BrowserSettingsSection`

These large sections are likely movable, but they depend on many helpers and handlers. Move one feature group at a time and run `npm run build` plus focused Vitest checks after each group.
