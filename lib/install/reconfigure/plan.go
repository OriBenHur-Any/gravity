package reconfigure

import (
	"github.com/gravitational/gravity/lib/install"
	"github.com/gravitational/gravity/lib/ops"
	"github.com/gravitational/gravity/lib/storage"

	"github.com/gravitational/trace"
)

func NewPlanner(getter install.PlanBuilderGetter) *Planner {
	return &Planner{
		PlanBuilderGetter: getter,
	}
}

func (p *Planner) GetOperationPlan(operator ops.Operator, cluster ops.Site, operation ops.SiteOperation) (*storage.OperationPlan, error) {
	// The "reconfigure" operation reuses a lot of the install fsm phases.
	installBuilder, err := p.GetPlanBuilder(operator, cluster, operation)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	builder := &PlanBuilder{
		PlanBuilder: installBuilder,
	}

	plan := &storage.OperationPlan{
		OperationID:   operation.ID,
		OperationType: operation.Type,
		AccountID:     operation.AccountID,
		ClusterName:   operation.SiteDomain,
		Servers:       append(builder.Masters, builder.Nodes...),
		DNSConfig:     cluster.DNSConfig,
	}

	// TODO(r0mant): Add checks phase?
	builder.AddConfigurePhase(plan)
	builder.AddPullPhase(plan)
	builder.AddMastersPhase(plan)
	builder.AddWaitPhase(plan)
	builder.AddHealthPhase(plan)
	builder.AddFixPhase(plan)

	return plan, nil
}

type Planner struct {
	install.PlanBuilderGetter
}
