// TODO: License
package cli

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/gravitational/gravity/lib/constants"
	"github.com/gravitational/gravity/lib/defaults"
	"github.com/gravitational/gravity/lib/install"
	installerclient "github.com/gravitational/gravity/lib/install/client"
	"github.com/gravitational/gravity/lib/install/reconfigure"
	"github.com/gravitational/gravity/lib/loc"
	"github.com/gravitational/gravity/lib/localenv"
	"github.com/gravitational/gravity/lib/ops"
	"github.com/gravitational/gravity/lib/storage"
	"github.com/gravitational/gravity/lib/storage/keyval"
	"github.com/gravitational/gravity/lib/system/signals"
	"github.com/gravitational/gravity/lib/utils"
	"github.com/gravitational/gravity/lib/utils/cli"

	"github.com/gravitational/trace"
)

func reconfigureCluster(env *localenv.LocalEnvironment, config InstallConfig) error {
	env.PrintStep("Starting reconfigurator")
	// Determine existing cluster name.
	// TODO(r0mant): Do something smarter.
	repos, err := env.Packages.GetRepositories()
	if err != nil {
		return trace.Wrap(err)
	}
	if len(repos) != 2 {
		return trace.BadParameter("expected 2 repositories: %v", repos)
	}
	var clusterName string
	for _, repo := range repos {
		if repo != defaults.SystemAccountOrg {
			clusterName = repo
		}
	}
	env.PrintStep("Cluster name: %v", clusterName)
	config.SiteDomain = clusterName
	if err := config.CheckAndSetDefaults(); err != nil {
		return trace.Wrap(err)
	}
	if config.FromService {
		err := startReconfiguratorFromService(env, config)
		if utils.IsContextCancelledError(err) {
			return trace.Wrap(err, "installer interrupted")
		}
		return trace.Wrap(err)
	}
	strategy, err := newReconfiguratorConnectStrategy(env, config, cli.CommandArgs{
		Parser: cli.ArgsParserFunc(parseArgs),
	})
	if err != nil {
		return trace.Wrap(err)
	}
	err = InstallerClient(env, installerclient.Config{
		ConnectStrategy: strategy,
		Lifecycle: &installerclient.AutomaticLifecycle{
			Aborter:            AborterForMode(config.Mode, env),
			Completer:          InstallerCompleteOperation(env),
			DebugReportPath:    DebugReportPath(),
			LocalDebugReporter: InstallerGenerateLocalReport(env),
		},
	})
	if utils.IsContextCancelledError(err) {
		// We only end up here if the initialization has not been successful - clean up the state
		InstallerCleanup()
		return trace.Wrap(err, "installer interrupted")
	}
	return trace.Wrap(err)
}

func startReconfiguratorFromService(env *localenv.LocalEnvironment, config InstallConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	interrupt := signals.NewInterruptHandler(ctx, cancel, InterruptSignals)
	defer interrupt.Close()
	go TerminationHandler(interrupt, env)
	listener, err := NewServiceListener()
	if err != nil {
		return trace.Wrap(utils.NewPreconditionFailedError(err))
	}
	defer func() {
		if err != nil {
			listener.Close()
		}
	}()
	installerConfig, err := newInstallerConfig(ctx, env, config)
	if err != nil {
		return trace.Wrap(utils.NewPreconditionFailedError(err))
	}
	installer, err := newReconfigurator(ctx, installerConfig)
	if err != nil {
		return trace.Wrap(utils.NewPreconditionFailedError(err))
	}
	interrupt.AddStopper(installer)
	return trace.Wrap(installer.Run(listener))
}

func newReconfigurator(ctx context.Context, config *install.Config) (*install.Installer, error) {
	installOperation, err := getInstallOperation(config)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	engine, err := reconfigure.NewEngine(reconfigure.Config{
		Operator:         config.Operator,
		InstallOperation: installOperation,
		// AdvertiseAddr: config.AdvertiseAddr,
		// Token:         config.Token.Token,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	config.LocalAgent = false
	installer, err := install.New(ctx, install.RuntimeConfig{
		Config:         *config,
		Planner:        reconfigure.NewPlanner(config),
		FSMFactory:     install.NewFSMFactory(*config),
		ClusterFactory: install.NewClusterFactory(*config),
		Engine:         engine,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return installer, nil
}

func getInstallOperation(config *install.Config) (*storage.Site, *ops.SiteOperation, error) {
	_, reader, err := config.LocalPackages.ReadPackage(loc.Locator{
		Repository: config.SiteDomain,
		Name:       constants.SiteExportPackage,
		Version:    "0.0.1",
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer reader.Close()
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer os.Remove(tempFile.Name())
	_, err = io.Copy(tempFile, reader)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	backend, err := keyval.NewBolt(keyval.BoltConfig{
		Path: tempFile.Name(),
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer backend.Close()
	cluster, err := backend.GetSite(config.SiteDomain)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	operations, err := storage.GetOperations(backend)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	for _, operation := range operations {
		if operation.Type == ops.OperationInstall {
			opsOperation := (ops.SiteOperation)(operation)
			return &opsOperation, nil
		}
	}
	return nil, trace.NotFound("install operation not found")
}
