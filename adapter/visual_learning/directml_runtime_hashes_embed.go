package visual_learning

import _ "embed"

// embeddedDirectMLRuntimeHashes pins the optional DirectML DLL payload.
// Missing or mismatched entries intentionally keep replay on OpenCV-only.
//
//go:embed directml_runtime_hashes.json
var embeddedDirectMLRuntimeHashes []byte
