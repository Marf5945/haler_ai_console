# Codex Adapter

For the full runnable builder package, install the `../../` folder itself. The
root `SKILL.md` sits beside `builder/`, `conformance/`, `WA3-SPEC.md`, and
`skill.json`, so build/trust/bundle-check commands can run after install.

The reduced Codex adapter lives at `../../skills/wa3-spec/`. Use it only when a
host wants a small spec-guidance skill without the builder/conformance toolchain.

For local development, keep `../../skills/wa3-spec/references/WA3-SPEC.md`
identical to `../../WA3-SPEC.md`. The reduced skill should stay concise and
point back to the full specification instead of duplicating it.
