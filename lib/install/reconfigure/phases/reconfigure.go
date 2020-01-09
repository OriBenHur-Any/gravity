package phases

import (
	"context"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/defaults"
	"github.com/gravitational/gravity/lib/fsm"
	"github.com/gravitational/gravity/lib/localenv"
	"github.com/gravitational/gravity/lib/ops"
	"github.com/gravitational/gravity/lib/pack"
	"github.com/gravitational/gravity/lib/storage"
	"github.com/gravitational/gravity/lib/utils"

	"github.com/gravitational/rigging"
	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func NewFix(p fsm.ExecutorParams, operator ops.Operator, packages pack.PackageService, client *kubernetes.Clientset) (*fixExecutor, error) {
	logger := &fsm.Logger{
		FieldLogger: logrus.WithFields(logrus.Fields{
			constants.FieldPhase: p.Phase.ID,
		}),
		Key:      opKey(p.Plan),
		Operator: operator,
		Server:   p.Phase.Data.Server,
	}
	return &fixExecutor{
		FieldLogger:    logger,
		ExecutorParams: p,
		LocalPackages:  packages,
		Client:         client,
	}, nil
}

type fixExecutor struct {
	// FieldLogger is used for logging.
	logrus.FieldLogger
	// ExecutorParams are common executor parameters.
	fsm.ExecutorParams
	// LocalPackages is the node-local package service.
	LocalPackages pack.PackageService
	// Client is the Kubernetes client.
	Client *kubernetes.Clientset
}

func (p *fixExecutor) Execute(ctx context.Context) error {
	// Remove old configuration/secrets packages.
	p.Progress.NextStep("Cleaning up local packages")
	err := pack.ForeachPackage(p.LocalPackages, func(e pack.PackageEnvelope) error {
		if val, ok := e.RuntimeLabels[pack.AdvertiseIPLabel]; ok {
			if val != p.Phase.Data.Server.AdvertiseIP {
				err := p.LocalPackages.DeletePackage(e.Locator)
				if err != nil {
					return trace.Wrap(err)
				}
				p.Progress.NextStep("Removed local package %v", e.Locator)
			}
		}
		return nil
	})
	if err != nil {
		return trace.Wrap(err)
	}

	p.Progress.NextStep("Updating cluster state")
	clusterEnv, err := localenv.NewClusterEnvironment()
	if err != nil {
		return trace.Wrap(err)
	}
	cluster, err := clusterEnv.Backend.GetLocalSite(defaults.SystemAccountID)
	if err != nil {
		return trace.Wrap(err)
	}
	cluster.ClusterState.Servers = storage.Servers{*p.Phase.Data.Server}
	_, err = clusterEnv.Backend.UpdateSite(*cluster)
	if err != nil {
		return trace.Wrap(err)
	}

	// Remove CNI interface.
	p.Progress.NextStep("Cleaning up network interfaces")
	ifaces, err := net.Interfaces()
	if err != nil {
		return trace.Wrap(err)
	}
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "cni") {
			err := utils.Exec(exec.Command("ip", "link", "del", iface.Name), os.Stdout)
			if err != nil {
				return trace.Wrap(err)
			}
			p.Progress.NextStep("Removed inteface %v", iface.Name)
		}
	}

	// Remove service account tokens.
	p.Progress.NextStep("Removing service account tokens")
	secrets, err := p.Client.CoreV1().Secrets(constants.AllNamespaces).List(metav1.ListOptions{})
	if err != nil {
		return rigging.ConvertError(err)
	}
	for _, secret := range secrets.Items {
		// Only remove service account tokens.
		if secret.Type != v1.SecretTypeServiceAccountToken {
			p.Progress.NextStep("Skipping secret %v/%v", secret.Namespace, secret.Name)
			continue
		}
		// Do not remove tokens for system controllers, Kubernetes will refresh those on its own.
		if secret.Namespace == metav1.NamespaceSystem && strings.Contains(secret.Name, "controller") {
			p.Progress.NextStep("Skipping secret %v/%v", secret.Namespace, secret.Name)
			continue
		}
		err := p.Client.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{})
		if err != nil {
			p.Progress.NextStep("Failed to remove secret %v/%v: %v", secret.Namespace, secret.Name, err)
		} else {
			p.Progress.NextStep("Removed secret %v/%v", secret.Namespace, secret.Name)
		}
	}

	// Remove Kubernetes node.
	p.Progress.NextStep("Removing Kubernetes node")
	nodes, err := utils.GetNodes(p.Client.CoreV1().Nodes())
	if err != nil {
		return trace.Wrap(err)
	}
	for ip, node := range nodes {
		if ip != p.Phase.Data.Server.AdvertiseIP {
			err := p.Client.CoreV1().Nodes().Delete(node.Name, &metav1.DeleteOptions{})
			if err != nil {
				return rigging.ConvertError(err)
			}
			p.Progress.NextStep("Removed node %v", node.Name)
		}
	}

	return trace.BadParameter("FAIL")
	return nil
}

// Rollback is no-op for this phase.
func (*fixExecutor) Rollback(ctx context.Context) error {
	return nil
}

// PreCheck is no-op for this phase.
func (*fixExecutor) PreCheck(ctx context.Context) error {
	return nil
}

// PostCheck is no-op for this phase.
func (*fixExecutor) PostCheck(ctx context.Context) error {
	return nil
}
