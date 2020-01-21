package opsservice

import (
	"context"

	"github.com/gravitational/gravity/lib/ops"
	"github.com/gravitational/gravity/lib/storage"

	"github.com/gravitational/trace"
	"github.com/pborman/uuid"
)

//
func (o *Operator) CreateClusterReconfigureOperation(ctx context.Context, req ops.CreateClusterReconfigureOperationRequest) (*ops.SiteOperationKey, error) {
	// err := req.Check()
	// if err != nil {
	// 	return nil, trace.Wrap(err)
	// }

	cluster, err := o.openSite(req.SiteKey)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	op := ops.SiteOperation{
		ID:         uuid.New(),
		AccountID:  req.AccountID,
		SiteDomain: req.SiteDomain,
		Type:       ops.OperationReconfigure,
		Created:    cluster.clock().UtcNow(),
		CreatedBy:  storage.UserFromContext(ctx),
		Updated:    cluster.clock().UtcNow(),
		State:      ops.OperationReconfigureInProgress,
	}

	key, err := cluster.getOperationGroup().createSiteOperation(op)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return key, nil
}
