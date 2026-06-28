# WA3 for LangGraph

LangGraph is not a folder-skill install target in the same sense as Codex or
OpenHands. Treat WA3 as an assistant/graph integration: the graph collects guided
answers, runs the deterministic WA3 builder, and returns draft/trust artifacts.

## Files

- `langgraph.json` - example graph manifest.
- `wa3_builder_graph.py` - minimal graph-shaped example for wrapping the WA3 CLI.

## Integration Boundary

- The graph may ask questions and collect answers.
- The graph must not decide `risk_class`, trust state, or canonical validity.
- The graph calls the WA3 CLI or equivalent deterministic implementation for:
  - build
  - secret scan
  - risk gate
  - canonicalization
  - trust enum

## Prompt To Install

```text
Load adapters/langgraph/README.md and use langgraph.json as the assistant
integration sketch. Do not treat this as a native skill install. Wire the graph
to call conformance/tools/wa3 for build and trust checks.
```
