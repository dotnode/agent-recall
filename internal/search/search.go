package search

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"agent-recall/internal/store"
)

type Query struct {
	Text      string `json:"query,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

type Hit struct {
	ID        string               `json:"id"`
	Score     float64              `json:"score"`
	Timestamp string               `json:"ts,omitempty"`
	SessionID string               `json:"session_id,omitempty"`
	CWD       string               `json:"cwd,omitempty"`
	Role      string               `json:"role,omitempty"`
	Kind      string               `json:"kind"`
	Text      string               `json:"text"`
	Source    store.EvidenceSource `json:"source"`
}

type Result struct {
	Notice string `json:"notice"`
	Query  string `json:"query,omitempty"`
	Items  []Hit  `json:"items"`
}

func Recall(dir string, q Query) (Result, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	records, err := store.LoadRecords(dir)
	if err != nil {
		return Result{}, err
	}
	terms := terms(q.Text)
	var hits []Hit
	for _, rec := range records {
		if q.SessionID != "" && rec.SessionID != q.SessionID {
			continue
		}
		if q.CWD != "" && rec.CWD != q.CWD {
			continue
		}
		if q.Kind != "" && rec.Kind != q.Kind {
			continue
		}
		score := score(rec, q.Text, terms)
		if q.Text != "" && score == 0 {
			continue
		}
		hits = append(hits, hit(rec, score))
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Source.ByteStart > hits[j].Source.ByteStart
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	return Result{Notice: store.Notice, Query: q.Text, Items: hits}, nil
}

func Timeline(dir string, q Query) (Result, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	records, err := store.LoadRecords(dir)
	if err != nil {
		return Result{}, err
	}
	var hits []Hit
	for i := len(records) - 1; i >= 0 && len(hits) < q.Limit; i-- {
		rec := records[i]
		if q.SessionID != "" && rec.SessionID != q.SessionID {
			continue
		}
		if q.CWD != "" && rec.CWD != q.CWD {
			continue
		}
		hits = append(hits, hit(rec, 1))
	}
	return Result{Notice: store.Notice, Query: q.Text, Items: hits}, nil
}

func Decisions(dir string, q Query) (Result, error) {
	q.Kind = "decision"
	return Recall(dir, q)
}

func WriteResult(w io.Writer, res Result, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	fmt.Fprintln(w, res.Notice)
	if res.Query != "" {
		fmt.Fprintf(w, "Query: %s\n\n", res.Query)
	}
	if len(res.Items) == 0 {
		fmt.Fprintln(w, "No evidence found.")
		return nil
	}
	for i, item := range res.Items {
		fmt.Fprintf(w, "%d. [%s %s score=%.1f] %s\n", i+1, item.Role, item.Kind, item.Score, oneLine(item.Text, 500))
		fmt.Fprintf(w, "   source: %s:%d\n", item.Source.TranscriptPath, item.Source.Line)
	}
	return nil
}

func score(rec store.EvidenceRecord, query string, toks []string) float64 {
	text := strings.ToLower(rec.Text)
	queryLower := strings.ToLower(query)
	var s float64
	for _, t := range toks {
		if strings.Contains(text, t) {
			s += 1
		}
	}
	if queryLower != "" && strings.Contains(text, queryLower) {
		s += 5
	}
	if rec.Kind == "decision" {
		s += 2
	}
	if rec.Role == "user" {
		s += 1
	}
	return s
}

func hit(rec store.EvidenceRecord, score float64) Hit {
	return Hit{ID: rec.ID, Score: score, Timestamp: rec.Timestamp, SessionID: rec.SessionID, CWD: rec.CWD, Role: rec.Role, Kind: rec.Kind, Text: rec.Text, Source: rec.Source}
}

func terms(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
	}) {
		f = strings.TrimSpace(f)
		if len([]rune(f)) < 2 || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func oneLine(text string, max int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= max {
		return text
	}
	return text[:max] + "…"
}
