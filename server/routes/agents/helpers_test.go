package agents

import "testing"

func TestNormalizeAgentType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{name: "chat", input: "chat", expected: "chat", ok: true},
		{name: "chat with spacing", input: " Chat ", expected: "chat", ok: true},
		{name: "invalid", input: "assistant", expected: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentType, ok := normalizeAgentType(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if agentType != tt.expected {
				t.Fatalf("expected agent type %q, got %q", tt.expected, agentType)
			}
		})
	}
}
