# WA3 for Voiceflow

Voiceflow is not a folder-skill install target. Treat WA3 as a tool/function
integration behind a Voiceflow assistant: Voiceflow collects guided answers and
calls a backend that runs the deterministic WA3 builder.

## Files

- `voiceflow-tool-spec.json` - tool contract sketch.
- `voiceflow-function.js` - example function wrapper that calls a backend WA3
  builder endpoint.

## Security Boundary

- Voiceflow may guide the user through template questions.
- Voiceflow must not decide `risk_class`, trust state, or canonical validity.
- The backend WA3 builder must run schema validation, secret scan, risk gate,
  canonicalization, lint, and trust classification.
- Do not store credentials inside `.tdy` or Voiceflow variables.

## Prompt To Install

```text
Use adapters/voiceflow/ as the Voiceflow integration sketch. Create a function
that sends guided answers to a backend WA3 builder. Do not embed credentials in
the function or generated .tdy file.
```
