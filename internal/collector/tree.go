//go:build windows

package collector

import (
	"sort"

	"github.com/ersinkoc/WindowsTaskManager/internal/metrics"
)

// BuildProcessTree assembles a forest of ProcessNode roots from a flat list.
// Children are sorted by CPU descending. Processes whose parent does not
// exist (or whose parent's create time is later than the child's) are
// surfaced as orphan roots.
func BuildProcessTree(procs []metrics.ProcessInfo) []*metrics.ProcessNode {
	byPID := make(map[uint32]*metrics.ProcessNode, len(procs))
	for i := range procs {
		p := procs[i]
		byPID[p.PID] = &metrics.ProcessNode{Process: p}
	}

	var roots []*metrics.ProcessNode
	for _, n := range byPID {
		parent, ok := byPID[n.Process.ParentPID]
		if !ok || n.Process.ParentPID == 0 {
			roots = append(roots, n)
			continue
		}
		// Detect orphan: parent created after child means PID was reused.
		if parent.Process.CreateTime > n.Process.CreateTime && parent.Process.CreateTime > 0 && n.Process.CreateTime > 0 {
			n.IsOrphan = true
			roots = append(roots, n)
			continue
		}
		parent.Children = append(parent.Children, n)
	}

	for _, n := range byPID {
		sort.Slice(n.Children, func(i, j int) bool {
			return n.Children[i].Process.CPUPercent > n.Children[j].Process.CPUPercent
		})
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Process.CPUPercent > roots[j].Process.CPUPercent
	})

	for _, r := range roots {
		assignDepth(r, 0)
	}

	return roots
}

func assignDepth(n *metrics.ProcessNode, d int) {
	n.Depth = d
	for _, c := range n.Children {
		assignDepth(c, d+1)
	}
}
