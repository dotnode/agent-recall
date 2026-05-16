package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
)

const MaxText = 16 * 1024

type Parsed struct {
	SessionID string
	Timestamp string
	UUID      string
	Role      string
	Kind      string
	Text      string
}

func Parse(raw []byte) (Parsed, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return Parsed{}, err
	}
	p := Parsed{
		SessionID: stringValue(obj["sessionId"]),
		Timestamp: stringValue(obj["timestamp"]),
		UUID:      stringValue(obj["uuid"]),
		Kind:      "message",
	}
	if t := stringValue(obj["type"]); t != "" {
		p.Kind = t
	}
	message, _ := obj["message"].(map[string]any)
	if message != nil {
		p.Role = stringValue(message["role"])
		text, kind := contentText(message["content"])
		p.Text = text
		if kind != "" {
			p.Kind = kind
		}
	}
	if p.Role == "" {
		p.Role = stringValue(obj["role"])
	}
	if p.Text == "" {
		text, kind := contentText(obj["content"])
		p.Text = text
		if kind != "" {
			p.Kind = kind
		}
	}
	if p.Role == "" {
		p.Role = "unknown"
	}
	p.Text = strings.TrimSpace(p.Text)
	if len(p.Text) > MaxText {
		p.Text = p.Text[:MaxText] + "\n[TRUNCATED]"
	}
	return p, nil
}

func contentText(v any) (string, string) {
	switch c := v.(type) {
	case string:
		return c, "message"
	case []any:
		var parts []string
		kind := "message"
		for _, item := range c {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			typ := stringValue(m["type"])
			switch typ {
			case "text":
				parts = append(parts, stringValue(m["text"]))
			case "tool_use":
				kind = "tool_use"
				parts = append(parts, fmt.Sprintf("tool_use %s %s", stringValue(m["name"]), compactJSON(m["input"])))
			case "tool_result":
				kind = "tool_result"
				if value := stringValue(m["content"]); value != "" {
					parts = append(parts, value)
				} else if value, _ := contentText(m["content"]); value != "" {
					parts = append(parts, value)
				}
			default:
				if txt := stringValue(m["text"]); txt != "" {
					parts = append(parts, txt)
				}
			}
		}
		return strings.Join(parts, "\n"), kind
	case map[string]any:
		return compactJSON(c), "message"
	default:
		return "", ""
	}
}

func compactJSON(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
