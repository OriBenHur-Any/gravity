package reconfigure

import (
	"strings"

	"github.com/gravitational/gravity/lib/fsm"
	"github.com/gravitational/gravity/lib/httplib"
	"github.com/gravitational/gravity/lib/install"
	installphases "github.com/gravitational/gravity/lib/install/phases"
	"github.com/gravitational/gravity/lib/install/reconfigure/phases"

	"github.com/gravitational/trace"
)

// FSMSpec returns a function that returns an appropriate phase executor
// based on the provided parameters.
func FSMSpec(config install.FSMConfig) fsm.FSMSpecFunc {
	return func(p fsm.ExecutorParams, remote fsm.Remote) (fsm.PhaseExecutor, error) {
		switch {
		case p.Phase.ID == installphases.ChecksPhase:
			return installphases.NewChecks(p,
				config.Operator,
				config.OperationKey)

		case p.Phase.ID == installphases.ConfigurePhase:
			return installphases.NewConfigure(p,
				config.Operator)

		case strings.HasPrefix(p.Phase.ID, installphases.PullPhase):
			return installphases.NewPull(p,
				config.Operator,
				config.Packages,
				config.LocalPackages,
				config.Apps,
				config.LocalApps,
				remote)

		// TODO(r0mant): Reconfiguration is only currently supported for
		// single-node clusters so only "/masters" phase can be present.
		case strings.HasPrefix(p.Phase.ID, installphases.MastersPhase):
			return installphases.NewSystem(p,
				config.Operator,
				remote)

		case p.Phase.ID == installphases.WaitPhase:
			client, _, err := httplib.GetClusterKubeClient(p.Plan.DNSConfig.Addr())
			if err != nil {
				return nil, trace.Wrap(err)
			}
			return installphases.NewWait(p,
				config.Operator,
				client)

		case p.Phase.ID == installphases.HealthPhase:
			return installphases.NewHealth(p,
				config.Operator)

		case p.Phase.ID == ReconfigurePhase:
			client, _, err := httplib.GetClusterKubeClient(p.Plan.DNSConfig.Addr())
			if err != nil {
				return nil, trace.Wrap(err)
			}
			return phases.NewFix(p,
				config.Operator,
				config.LocalPackages,
				client)

		default:
			return nil, trace.BadParameter("unknown phase %q", p.Phase.ID)
		}
	}
}

const (
	ReconfigurePhase = "/reconfigure"
)
