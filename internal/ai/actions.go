package ai

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Suggestion is one concrete operation the LLM proposes. The dashboard
// renders these as approve-buttons; nothing executes until the user clicks.
type Suggestion struct {
	ID     string          `json:"id"`               // stable hash so the UI can dedupe
	Type   string          `json:"type"`             // kill | suspend | protect | ignore | add_rule
	PID    uint32          `json:"pid,omitempty"`    // required for kill/suspend
	Name   string          `json:"name,omitempty"`   // executable name (used by protect/ignore)
	Reason string          `json:"reason,omitempty"` // free-form, <=80 chars
	Rule   *RuleSuggestion `json:"rule,omitempty"`   // populated for add_rule suggestions
	Policy *AutoPolicy     `json:"policy,omitempty"` // background auto-action evaluation
}

// AutoPolicy explains whether a suggestion would qualify for future
// background auto-execution. Phase 2 is still dry-run only: we surface
// deterministic eligibility, but do not execute automatically yet.
type AutoPolicy struct {
	Status              string `json:"status"` // disabled | blocked | needs_repeat | dry_run_eligible
	Reason              string `json:"reason,omitempty"`
	RepeatCount         int    `json:"repeat_count,omitempty"`
	RequiredRepeatCount int    `json:"required_repeat_count,omitempty"`
}

// RuleSuggestion is the wire shape for a rule proposed by the LLM. We keep
// durations as plain seconds ints (not Go time.Duration) so JSON encoding
// matches the rules API the dashboard already speaks. We also accept both
// `for`/`cooldown` and `for_seconds`/`cooldown_seconds` because the model
// picks whichever it thinks looks "nicer" despite the prompt telling it to
// use the explicit *_seconds form.
type RuleSuggestion struct {
	Name            string  `json:"name"`
	Enabled         bool    `json:"enabled"`
	Match           string  `json:"match"`
	Metric          string  `json:"metric"`
	Op              string  `json:"op"`
	Threshold       float64 `json:"threshold"`
	For             int     `json:"for,omitempty"`
	ForSeconds      int     `json:"for_seconds,omitempty"`
	Action          string  `json:"action"`
	Cooldown        int     `json:"cooldown,omitempty"`
	CooldownSeconds int     `json:"cooldown_seconds,omitempty"`
}

// AnalyzeResult is what the advisor hands back: the visible answer stripped
// of the raw <actions> block, plus the parsed structured actions.
type AnalyzeResult struct {
	Answer  string       `json:"answer"`
	Actions []Suggestion `json:"actions,omitempty"`
	Cached  bool         `json:"cached,omitempty"`
}

// parseActionsBlock scans `raw` for a single <actions>...</actions> block,
// decodes its JSON body, and returns the cleaned answer (with the block cut
// out) plus the parsed actions. On malformed JSON we keep the raw text and
// return zero actions — the user still sees the analysis.
func parseActionsBlock(raw string) (string, []Suggestion) {
	start := strings.Index(raw, "<actions>")
	if start < 0 {
		return strings.TrimSpace(raw), nil
	}
	end := strings.Index(raw[start:], "</actions>")
	if end < 0 {
		return strings.TrimSpace(raw), nil
	}
	end += start

	body := strings.TrimSpace(raw[start+len("<actions>") : end])
	cleaned := strings.TrimSpace(raw[:start] + raw[end+len("</actions>"):])

	// The model sometimes wraps the JSON in a fenced code block despite
	// our instructions; peel those off defensively.
	body = strings.TrimPrefix(body, "```json")
	body = strings.TrimPrefix(body, "```")
	body = strings.TrimSuffix(body, "```")
	body = strings.TrimSpace(body)

	if body == "" || body == "[]" {
		return cleaned, nil
	}

	var items []Suggestion
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		// Leave a footprint in the answer so the user can see what broke.
		return cleaned + "\n\n(actions block parse error: " + err.Error() + ")", nil
	}

	out := make([]Suggestion, 0, len(items))
	for _, it := range items {
		it.Type = strings.ToLower(strings.TrimSpace(it.Type))
		switch it.Type {
		case "kill", "suspend", "protect", "ignore", "add_rule":
		default:
			continue
		}
		it.ID = hashSuggestion(it)
		out = append(out, it)
	}
	return cleaned, out
}

// hashSuggestion returns a short stable ID so the UI can dedupe across
// repeated analyze calls.
func hashSuggestion(s Suggestion) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%d|%s", s.Type, s.PID, strings.ToLower(s.Name))
	if s.Rule != nil {
		fmt.Fprintf(h, "|%s|%s|%v", s.Rule.Name, s.Rule.Match, s.Rule.Threshold)
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)[:12]
}
