package redact

import "testing"

func TestDefaultRedactor(t *testing.T) {
	r := Default()
	res := r.Redact("API_KEY=sk-ant-abc1234567890\nAuthorization: Bearer abcdefghijklmnopqrstuvwxyz")
	if !res.Applied {
		t.Fatalf("expected redaction to apply")
	}
	if len(res.Rules) == 0 {
		t.Fatalf("expected redaction rules")
	}
	if res.Text == "" || res.Text == "API_KEY=sk-ant-abc1234567890" {
		t.Fatalf("text was not redacted: %q", res.Text)
	}
}
