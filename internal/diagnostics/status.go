package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"

	"agent-recall/internal/config"
	"agent-recall/internal/store"
)

const (
	StateOK         = "ok"
	StateWarning    = "warning"
	StateError      = "error"
	StateDisabled   = "disabled"
	StateNotChecked = "not_checked"
)

type ComponentStatus struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

type Status struct {
	Components map[string]ComponentStatus `json:"components"`
	Store      store.StatusInfo           `json:"store"`
}

type Options struct {
	MCPState   string
	MCPMessage string
}

func Check(storeDir string, opts Options) Status {
	mcpState := opts.MCPState
	if mcpState == "" {
		mcpState = StateNotChecked
	}
	mcpMessage := opts.MCPMessage
	if mcpMessage == "" {
		if mcpState == StateOK {
			mcpMessage = "MCP server responded"
		} else {
			mcpMessage = "not checked from this command; use /agent-recall:memory-status or claude mcp list to verify MCP"
		}
	}

	out := Status{Components: map[string]ComponentStatus{
		"mcp": {State: mcpState, Message: mcpMessage},
	}}

	st, err := store.Status(storeDir)
	out.Store = st
	if err != nil {
		out.Components["store"] = ComponentStatus{State: StateError, Message: err.Error()}
		out.Components["hook"] = ComponentStatus{State: StateError, Message: "store is not readable; hook ingestion cannot be verified"}
	} else {
		out.Components["store"] = storeComponent(st)
		out.Components["hook"] = hookComponent(st)
	}
	out.Components["model"] = modelComponent()
	return out
}

func Write(w io.Writer, status Status, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}
	fmt.Fprintln(w, "agent-recall status")
	fmt.Fprintln(w)
	writeComponent(w, "MCP", status.Components["mcp"])
	writeComponent(w, "Hook", status.Components["hook"])
	writeComponent(w, "Store", status.Components["store"])
	writeComponent(w, "Model", status.Components["model"])
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Store details:")
	fmt.Fprintf(w, "  Store:       %s\n", status.Store.StoreDir)
	fmt.Fprintf(w, "  Events:      %d\n", status.Store.Events)
	fmt.Fprintf(w, "  Sessions:    %d\n", status.Store.Sessions)
	fmt.Fprintf(w, "  Transcripts: %d\n", status.Store.Transcripts)
	fmt.Fprintf(w, "  Bytes:       %d\n", status.Store.StoreBytes)
	if status.Store.BadLines > 0 {
		fmt.Fprintf(w, "  Bad lines:   %d\n", status.Store.BadLines)
	}
	if status.Store.CursorError != "" {
		fmt.Fprintf(w, "  Cursor error: %s\n", status.Store.CursorError)
	}
	if status.Store.LastIngested != "" {
		fmt.Fprintf(w, "  Last sync:   %s\n", status.Store.LastIngested)
	}
	return nil
}

func storeComponent(st store.StatusInfo) ComponentStatus {
	if st.CursorError != "" {
		return ComponentStatus{State: StateError, Message: "cursor is unreadable: " + st.CursorError}
	}
	if st.BadLines > 0 {
		return ComponentStatus{State: StateWarning, Message: fmt.Sprintf("store readable with %d malformed evidence line(s)", st.BadLines)}
	}
	return ComponentStatus{State: StateOK, Message: "store readable at " + st.StoreDir}
}

func hookComponent(st store.StatusInfo) ComponentStatus {
	if st.CursorError != "" {
		return ComponentStatus{State: StateError, Message: "cursor is unreadable; hook ingestion state cannot be trusted"}
	}
	if st.Events == 0 {
		return ComponentStatus{State: StateWarning, Message: "no ingested events yet; check Stop and PreCompact hooks"}
	}
	if st.LastIngested == "" {
		return ComponentStatus{State: StateWarning, Message: "events exist but no last sync timestamp was found"}
	}
	return ComponentStatus{State: StateOK, Message: "last sync " + st.LastIngested}
}

func modelComponent() ComponentStatus {
	cfg, err := config.ModelConfigFromEnv()
	if err != nil {
		return ComponentStatus{State: StateError, Message: err.Error()}
	}
	if !cfg.Enabled() {
		return ComponentStatus{State: StateDisabled, Message: "optional search_answer is not configured"}
	}
	return ComponentStatus{State: StateOK, Message: fmt.Sprintf("search_answer enabled via %s model %q", cfg.Provider, cfg.Model)}
}

func writeComponent(w io.Writer, label string, c ComponentStatus) {
	fmt.Fprintf(w, "%-6s %s - %s\n", label+":", c.State, c.Message)
}
