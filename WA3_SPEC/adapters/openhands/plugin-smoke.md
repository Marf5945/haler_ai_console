# OpenHands Plugin Smoke Check

Use this as a checklist after loading the WA3 plugin.

1. Confirm `.plugin/plugin.json` exists.
2. Confirm root `SKILL.md` exists.
3. Confirm `builder/answers.schema.json` exists.
4. Confirm `conformance/tools/wa3/main.go` exists.
5. Run:

```sh
cd conformance
go run ./tools/wa3 bundle-check
go run ./tools/wa3 build --answers ../builder/examples/board.answers.json --out /tmp/wa3-board.draft.tdy --mock-demo /tmp/wa3-board.mock-demo.json
go run ./tools/wa3 trust /tmp/wa3-board.draft.tdy
```

Pass criteria:

- `bundle check ok`
- draft build exits successfully
- trust output has `state` equal to `unsigned_draft`
