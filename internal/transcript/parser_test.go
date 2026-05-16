package transcript

import "testing"

func TestParseTextAndToolUse(t *testing.T) {
	p, err := Parse([]byte(`{"type":"assistant","timestamp":"2026-05-16T10:02:00Z","uuid":"t1","sessionId":"s1","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"src/auth/callback.ts"}}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if p.Role != "assistant" || p.Kind != "tool_use" {
		t.Fatalf("unexpected parsed metadata: %#v", p)
	}
	if p.Text == "" {
		t.Fatalf("expected tool_use text")
	}
}
