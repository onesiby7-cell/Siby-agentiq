package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type SquadDashboard struct {
	orchestrator *Orchestrator
	mu           sync.RWMutex
}

func NewSquadDashboard(o *Orchestrator) *SquadDashboard {
	return &SquadDashboard{orchestrator: o}
}

func (d *SquadDashboard) Update() {
}

func (d *SquadDashboard) Render() string {
	var sb strings.Builder

	status := d.orchestrator.status

	d.orchestrator.mu.RLock()
	squads := d.orchestrator.squads
	d.orchestrator.mu.RUnlock()

	sb.WriteString(fmt.Sprintf(`
════════════════════════════════════════════════════════════════════════
                         🦂 SIBY-AGENTIQ DASHBOARD 🦂
════════════════════════════════════════════════════════════════════════

  Status: %s
  Squads: %d

════════════════════════════════════════════════════════════════════════
`, status, len(squads)))

	return sb.String()
}

func (d *SquadDashboard) RenderASCII() string {
	return d.Render()
}

func (d *SquadDashboard) GetStatus() string {
	return string(d.orchestrator.GetStatus())
}

type DashboardConfig struct {
	RefreshRate time.Duration
	ShowAgents  bool
	ShowMetrics bool
}

func (d *SquadDashboard) StartAutoRefresh(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				d.Update()
			}
		}
	}()
}
