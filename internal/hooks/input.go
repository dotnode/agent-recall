package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

type Input struct {
	HookEventName      string         `json:"hook_event_name,omitempty"`
	SessionID          string         `json:"session_id,omitempty"`
	TranscriptPath     string         `json:"transcript_path,omitempty"`
	CWD                string         `json:"cwd,omitempty"`
	Trigger            string         `json:"trigger,omitempty"`
	CustomInstructions string         `json:"custom_instructions,omitempty"`
	Raw                map[string]any `json:"-"`
}

func ReadInput(r io.Reader) (Input, error) {
	var raw map[string]any
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return Input{}, err
	}
	in := Input{Raw: raw}
	in.HookEventName, _ = raw["hook_event_name"].(string)
	in.SessionID, _ = raw["session_id"].(string)
	in.TranscriptPath, _ = raw["transcript_path"].(string)
	in.CWD, _ = raw["cwd"].(string)
	in.Trigger, _ = raw["trigger"].(string)
	in.CustomInstructions, _ = raw["custom_instructions"].(string)
	if in.TranscriptPath == "" {
		return in, fmt.Errorf("missing transcript_path")
	}
	return in, nil
}
