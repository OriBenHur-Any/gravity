package reconfigure

import (
	"github.com/gravitational/gravity/lib/fsm"
	"github.com/gravitational/gravity/lib/install"
	"github.com/gravitational/gravity/lib/install/engine"
	"github.com/gravitational/gravity/lib/ops"
)

// NewFSMFactory returns a new factory that can create fsm for the reconfigure operation.
func NewFSMFactory(config install.Config) engine.FSMFactory {
	return &fsmFactory{Config: config}
}

// NewFSM creates a new fsm for the provided operator and operation.
func (f *fsmFactory) NewFSM(operator ops.Operator, operationKey ops.SiteOperationKey) (*fsm.FSM, error) {
	fsmConfig := install.NewFSMConfig(operator, operationKey, f.Config)
	fsmConfig.Spec = FSMSpec(fsmConfig)
	return install.NewFSM(fsmConfig)
}

type fsmFactory struct {
	install.Config
}
