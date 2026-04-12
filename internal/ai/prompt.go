package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ersinkoc/WindowsTaskManager/internal/anomaly"
	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// actionsSchemaDoc is appended to every system prompt. It teaches the LLM how
// to emit a structured actions block we can parse server-side into "suggested
// operations" that the user then approves one by one from the dashboard.
const actionsSchemaDoc = `

When you recommend an operation the user could take, append a single
<actions>...</actions> block at the very end of your reply containing a JSON
array. Never put actions inside prose; only inside one <actions> block. Omit
the block entirely when you have no concrete operation to suggest.

Each item must be one of:
  {"type":"kill","pid":<uint>,"name":"<exe>","reason":"<short>"}
  {"type":"suspend","pid":<uint>,"name":"<exe>","reason":"<short>"}
  {"type":"protect","name":"<exe>","reason":"<short>"}
  {"type":"ignore","name":"<exe>","reason":"<short>"}
  {"type":"add_rule","rule":{"name":"<id>","match":"<exe>","metric":"cpu_percent|memory_bytes|thread_count","op":">=|>|<=|<","threshold":<num>,"for_seconds":<int>,"action":"alert|kill|suspend","cooldown_seconds":<int>},"reason":"<short>"}

Rules:
- NEVER suggest kill or suspend for Windows system processes (svchost, csrss,
  lsass, wininit, winlogon, services, smss, explorer, dwm, MsMpEng,
  SecurityHealthService, RuntimeBroker, taskhostw, fontdrvhost, ctfmon,
  sihost, SearchHost, SearchIndexer, StartMenuExperienceHost, ShellExperienceHost,
  conhost, dllhost, WmiPrvSE, spoolsv, audiodg, System, Registry). For these,
  suggest "protect" or "ignore" instead.
- Always include PID when acting on a specific process.
- Keep reasons under 80 chars.
- Only recommend operations that are directly justified by the snapshot data.
- Do not wrap the <actions> block in markdown fences.`

// systemPromptEN is the English persona instruction.
const systemPromptEN = `You are a Windows system performance expert. Given the current snapshot
and active alerts, give the user concise, concrete, actionable advice in
3-5 sentences max. No speculation; reason from data only.` + actionsSchemaDoc

// BuildPrompt assembles the user message that we send to the LLM.
func BuildPrompt(language string, snap *metrics.SystemSnapshot, alerts []anomaly.Alert, includeTree, includePorts bool, userQuestion string) string {
	if snap == nil {
		return userQuestion
	}
	var b strings.Builder

	b.WriteString("## SYSTEM SNAPSHOT\n")
	fmt.Fprintf(&b, "CPU: %.1f%% (%d cores, %s)\n", snap.CPU.TotalPercent, snap.CPU.NumLogical, snap.CPU.Name)
	fmt.Fprintf(&b, "Memory: %.1f%% used (%s / %s)\n",
		snap.Memory.UsedPercent, humanBytes(snap.Memory.UsedPhys), humanBytes(snap.Memory.TotalPhys))
	if snap.GPU.Available {
		fmt.Fprintf(&b, "GPU: %s util %.0f%%, %s/%s VRAM\n",
			snap.GPU.Name, snap.GPU.Utilization, humanBytes(snap.GPU.VRAMUsed), humanBytes(snap.GPU.VRAMTotal))
	}

	if len(snap.Disk.Drives) > 0 {
		b.WriteString("Disks: ")
		for i, d := range snap.Disk.Drives {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s %.0f%% used", d.Letter, d.UsedPct)
		}
		b.WriteByte('\n')
	}

	b.WriteString("\n## TOP CPU PROCESSES\n")
	top := append([]metrics.ProcessInfo(nil), snap.Processes...)
	sort.Slice(top, func(i, j int) bool { return top[i].CPUPercent > top[j].CPUPercent })
	if len(top) > 8 {
		top = top[:8]
	}
	for _, p := range top {
		fmt.Fprintf(&b, "- PID %d %-25s CPU %5.1f%% MEM %s\n",
			p.PID, truncate(p.Name, 25), p.CPUPercent, humanBytes(p.WorkingSet))
	}

	b.WriteString("\n## TOP MEMORY PROCESSES\n")
	mem := append([]metrics.ProcessInfo(nil), snap.Processes...)
	sort.Slice(mem, func(i, j int) bool { return mem[i].WorkingSet > mem[j].WorkingSet })
	if len(mem) > 8 {
		mem = mem[:8]
	}
	for _, p := range mem {
		fmt.Fprintf(&b, "- PID %d %-25s MEM %s CPU %.1f%%\n",
			p.PID, truncate(p.Name, 25), humanBytes(p.WorkingSet), p.CPUPercent)
	}

	if len(alerts) > 0 {
		b.WriteString("\n## ACTIVE ALERTS\n")
		for _, a := range alerts {
			fmt.Fprintf(&b, "- [%s] %s — %s\n", a.Severity, a.Title, a.Description)
		}
	}

	if includeTree && len(snap.ProcessTree) > 0 {
		b.WriteString("\n## PROCESS TREE (roots)\n")
		for i, root := range snap.ProcessTree {
			if i >= 6 {
				break
			}
			fmt.Fprintf(&b, "- %s (PID %d, children=%d)\n", root.Process.Name, root.Process.PID, len(root.Children))
		}
	}

	if includePorts && len(snap.PortBindings) > 0 {
		listening := 0
		for _, pb := range snap.PortBindings {
			if pb.State == "listen" {
				listening++
			}
		}
		fmt.Fprintf(&b, "\nListening ports: %d, total endpoints: %d\n", listening, len(snap.PortBindings))
	}

	if strings.TrimSpace(userQuestion) != "" {
		b.WriteString("\n## USER QUESTION\n")
		b.WriteString(userQuestion)
		b.WriteByte('\n')
	} else {
		b.WriteString("\n## REQUEST\nGive a short health assessment and call out the most worrying processes, if any.\n")
	}

	return b.String()
}

// BuildChatPrompt assembles the multi-turn chat prompt using the same
// snapshot context as Analyze plus a short in-memory conversation window.
func BuildChatPrompt(language string, snap *metrics.SystemSnapshot, alerts []anomaly.Alert, includeTree, includePorts bool, history []chatTurn, userMessage string) string {
	var b strings.Builder
	b.WriteString(BuildPrompt(language, snap, alerts, includeTree, includePorts, ""))
	if len(history) > 0 {
		b.WriteString("\n## RECENT CHAT\n")
		for _, turn := range history {
			role := "Assistant"
			if strings.EqualFold(turn.Role, "user") {
				role = "User"
			}
			fmt.Fprintf(&b, "%s: %s\n", role, strings.TrimSpace(turn.Content))
		}
	}
	b.WriteString("\n## CHAT TASK\n")
	b.WriteString("Continue the conversation naturally. Answer the latest user message directly using the current snapshot.\n")
	b.WriteString("If the user asks for an action recommendation, keep using the <actions> block format.\n")
	b.WriteString("\n## LATEST USER MESSAGE\n")
	b.WriteString(strings.TrimSpace(userMessage))
	b.WriteByte('\n')
	return b.String()
}

// SystemPrompt returns the system prompt for the requested language.
// Only English is supported; the argument is kept for forward compatibility
// with older callers that still pass a language hint.
func SystemPrompt(_ string) string {
	return systemPromptEN
}

func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := uint64(unit), 0
	for n/div >= unit && exp < 4 {
		div *= unit
		exp++
	}
	suffixes := []string{"K", "M", "G", "T", "P"}
	return fmt.Sprintf("%.1f%s", float64(n)/float64(div), suffixes[exp])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
