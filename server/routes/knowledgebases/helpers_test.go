package knowledgebases

import "testing"

func TestNormalizeKnowledgebaseProvider(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{name: "milvus", input: "Milvus", expected: "milvus", ok: true},
		{name: "mixedbread alias", input: "mixedbread-ai", expected: "mixedbread", ok: true},
		{name: "zero entropy spacing", input: "zero_entropy", expected: "zeroentropy", ok: true},
		{name: "algolia", input: "algolia", expected: "algolia", ok: true},
		{name: "qdrant", input: "Qdrant", expected: "qdrant", ok: true},
		{name: "shilp skipped", input: "shilp", expected: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, ok := normalizeKnowledgebaseProvider(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if provider != tt.expected {
				t.Fatalf("expected provider %q, got %q", tt.expected, provider)
			}
		})
	}
}

func TestMergeKnowledgebaseAuth(t *testing.T) {
	existing := map[string]interface{}{"api_key": "old-key", "token": "keep-me"}
	newAPIKey := "new-key"
	merged := mergeKnowledgebaseAuth(existing, &newAPIKey, nil)

	if merged["api_key"] != "new-key" {
		t.Fatalf("expected api_key to be updated, got %#v", merged["api_key"])
	}
	if merged["token"] != "keep-me" {
		t.Fatalf("expected token to be preserved, got %#v", merged["token"])
	}

	emptyAPIKey := ""
	merged = mergeKnowledgebaseAuth(merged, &emptyAPIKey, nil)
	if _, ok := merged["api_key"]; ok {
		t.Fatalf("expected api_key to be removed when empty")
	}
}
