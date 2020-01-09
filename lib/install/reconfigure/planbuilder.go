package reconfigure

import (
	"github.com/gravitational/gravity/lib/fsm"
	"github.com/gravitational/gravity/lib/install"
	"github.com/gravitational/gravity/lib/install/phases"
	"github.com/gravitational/gravity/lib/storage"
)

type PlanBuilder struct {
	*install.PlanBuilder
}

// TODO
func (b *PlanBuilder) AddFixPhase(plan *storage.OperationPlan) {
	plan.Phases = append(plan.Phases, storage.OperationPhase{
		ID:          "/fix",
		Description: "Fix everything",
		Requires:    fsm.RequireIfPresent(plan, phases.HealthPhase),
		Data: &storage.OperationPhaseData{
			Server: &b.Master,
		},
		Step: 4,
	})
}
