package store

import "testing"

func TestStableID(t *testing.T) {
	a := StableID("agent", "session", "line")
	b := StableID("agent", "session", "line")
	c := StableID("agent", "session", "other")
	if a != b {
		t.Fatalf("stable id changed")
	}
	if a == c {
		t.Fatalf("stable id collision for different inputs")
	}
}
