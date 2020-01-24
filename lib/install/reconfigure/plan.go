package reconfigure

import (
	"github.com/gravitational/gravity/lib/app"
	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/fsm"
	"github.com/gravitational/gravity/lib/install"
	"github.com/gravitational/gravity/lib/ops"
	"github.com/gravitational/gravity/lib/storage"

	"github.com/gravitational/trace"
)

func NewPlanner(getter install.PlanBuilderGetter, cluster storage.Site) *Planner {
	return &Planner{
		PlanBuilderGetter: getter,
		Cluster:           cluster,
	}
}

func (p *Planner) GetOperationPlan(operator ops.Operator, cluster ops.Site, operation ops.SiteOperation) (*storage.OperationPlan, error) {
	masters, _ := fsm.SplitServers(operation.Servers)
	if len(masters) == 0 {
		return nil, trace.BadParameter(
			"at least one master server is required: %v", operation.Servers)
	}

	teleportPackage, err := cluster.App.Manifest.Dependencies.ByName(
		constants.TeleportPackage)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// The "reconfigure" operation reuses a lot of the install fsm phases.
	builder := &PlanBuilder{
		PlanBuilder: &install.PlanBuilder{
			Cluster:   ops.ConvertOpsSite(cluster),
			Operation: operation,
			Application: app.Application{
				Package:         cluster.App.Package,
				PackageEnvelope: cluster.App.PackageEnvelope,
				Manifest:        cluster.App.Manifest,
			},
			Masters:         masters,
			Master:          masters[0],
			ServiceUser:     p.Cluster.ServiceUser,
			TeleportPackage: *teleportPackage,
		},
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
	Cluster storage.Site
}
