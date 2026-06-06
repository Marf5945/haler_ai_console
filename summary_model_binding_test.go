package main

import (
	"os"
	"path/filepath"
	"testing"

	"ui_console/adapter/adapter_registry"
)

func TestParseOllamaListOutputFiltersNonGenerativeModels(t *testing.T) {
	out := `NAME                                                      ID              SIZE      MODIFIED
nomic-embed-text:latest                                   0a109f422b47    274 MB    17 hours ago
hf.co/second-state/Breeze-7B-Instruct-v1_0-GGUF:Q4_K_M    baf51edfecf3    4.5 GB    13 seconds ago
mxbai-embed-large:latest                                  468836162de7    669 MB    2 days ago
qwen3:4b                                                  abcdef123456    2.5 GB    1 day ago
`
	options := parseOllamaListOutput(out)
	got := modelOptionIDs(options)
	want := []string{
		"hf.co/second-state/Breeze-7B-Instruct-v1_0-GGUF:Q4_K_M",
		"qwen3:4b",
	}
	if !sameStrings(got, want) {
		t.Fatalf("parseOllamaListOutput IDs = %#v, want %#v", got, want)
	}
}

func TestScanOllamaModelLibraryFiltersNonGenerativeModels(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "blobs"),
		filepath.Join(root, "manifests", "registry.ollama.ai", "library", "nomic-embed-text"),
		filepath.Join(root, "manifests", "registry.ollama.ai", "library", "llama3.2"),
		filepath.Join(root, "manifests", "hf.co", "second-state", "Breeze-7B-Instruct-v1_0-GGUF"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{
		filepath.Join(root, "manifests", "registry.ollama.ai", "library", "nomic-embed-text", "latest"),
		filepath.Join(root, "manifests", "registry.ollama.ai", "library", "llama3.2", "3b"),
		filepath.Join(root, "manifests", "hf.co", "second-state", "Breeze-7B-Instruct-v1_0-GGUF", "Q4_K_M"),
	} {
		if err := os.WriteFile(file, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := modelOptionIDs(scanOllamaModelLibrary(root))
	want := []string{
		"llama3.2:3b",
		"hf.co/second-state/Breeze-7B-Instruct-v1_0-GGUF:Q4_K_M",
	}
	if !sameStrings(got, want) {
		t.Fatalf("scanOllamaModelLibrary IDs = %#v, want %#v", got, want)
	}
}

func TestIsOllamaGenerativeModelID(t *testing.T) {
	generative := []string{
		"qwen3:4b",
		"llama3.2:3b",
		"hf.co/second-state/Breeze-7B-Instruct-v1_0-GGUF:Q4_K_M",
		"mistral:7b-instruct",
	}
	for _, id := range generative {
		if !isOllamaGenerativeModelID(id) {
			t.Fatalf("expected generative model %q to pass", id)
		}
	}

	nonGenerative := []string{
		"nomic-embed-text:latest",
		"mxbai-embed-large:latest",
		"bge-m3:latest",
		"jina/jina-embeddings-v2-base-en:latest",
		"some-reranker:latest",
	}
	for _, id := range nonGenerative {
		if isOllamaGenerativeModelID(id) {
			t.Fatalf("expected non-generative model %q to be filtered", id)
		}
	}
}

func TestShouldExposeAdapterFiltersRegisteredOllamaEmbeddingAdapter(t *testing.T) {
	if shouldExposeAdapter(adapter_registry.Adapter{
		ID:       "local-ollama-nomic-embed-text-latest",
		Name:     "Ollama - nomic-embed-text:latest",
		Kind:     "local",
		Endpoint: "http://localhost:11434/v1",
		Model:    "nomic-embed-text:latest",
	}) {
		t.Fatal("registered Ollama embedding adapter should not be exposed")
	}

	if !shouldExposeAdapter(adapter_registry.Adapter{
		ID:       "local-ollama-breeze-7b-instruct-v1-0-gguf-q4-k-m",
		Name:     "Ollama - hf.co/second-state/Breeze-7B-Instruct-v1_0-GGUF:Q4_K_M",
		Kind:     "local",
		Endpoint: "http://localhost:11434/v1",
		Model:    "hf.co/second-state/Breeze-7B-Instruct-v1_0-GGUF:Q4_K_M",
	}) {
		t.Fatal("registered Ollama chat adapter should be exposed")
	}
}

func modelOptionIDs(options []SummaryModelOption) []string {
	ids := make([]string, 0, len(options))
	for _, option := range options {
		ids = append(ids, option.ID)
	}
	return ids
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := map[string]int{}
	for _, value := range got {
		counts[value]++
	}
	for _, value := range want {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
