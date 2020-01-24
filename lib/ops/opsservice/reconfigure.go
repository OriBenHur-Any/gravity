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
	o.Infof("%#v", req)

	cluster, err := o.openSite(req.SiteKey)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	operation := ops.SiteOperation{
		ID:            uuid.New(),
		AccountID:     req.AccountID,
		SiteDomain:    req.SiteDomain,
		Type:          ops.OperationReconfigure,
		Created:       cluster.clock().UtcNow(),
		CreatedBy:     storage.UserFromContext(ctx),
		Updated:       cluster.clock().UtcNow(),
		State:         ops.OperationReconfigureInProgress,
		Servers:       req.Servers,
		InstallExpand: req.InstallExpand,
	}

	_, err = cluster.newProvisioningToken(operation)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	// for _, profile := range cluster.app.Manifest.NodeProfiles {
	// 	agentURL, err := cluster.makeAgentURL(token, profile.Name)
	// 	if err != nil {
	// 		return nil, trace.Wrap(err)
	// 	}
	// 	operation.InstallExpand.Agents[profile.Name] = storage.AgentProfile{
	// 		AgentURL: agentURL,
	// 	}
	// }

	key, err := cluster.getOperationGroup().createSiteOperation(operation)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return key, nil
}
