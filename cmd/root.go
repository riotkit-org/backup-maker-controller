package cmd

import (
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	controllers2 "github.com/riotkit-org/backup-maker-operator/pkg/controllers"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"github.com/riotkit-org/backup-maker-operator/pkg/integration"
	"github.com/riotkit-org/backup-maker-operator/pkg/locking"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func NewRootCommand() *cobra.Command {
	app := App{}
	command := &cobra.Command{
		Use:   "backup-maker-operator",
		Short: "Runs a controller that schedules and monitors Backup & Restore actions in the cluster",
		Run: func(command *cobra.Command, args []string) {
			err := app.Run()

			if err != nil {
				logrus.Errorf(err.Error())
				os.Exit(1)
			}
		},
	}

	command.Flags().BoolVarP(&app.debug, "debug", "v", true, "Increase verbosity to the debug level")
	command.Flags().StringVarP(&app.metricsBindAddress, "metrics-bind-address", "m", ":8080", "Host + Port on which to bind metrics endpoint to")
	command.Flags().StringVarP(&app.healthProbeBindAddress, "health-probe-bind-address", "p", ":8081", "Host + Port on which to bind healthcheck endpoint to")
	command.Flags().BoolVarP(&app.leaderElect, "leader-elect", "l", false, "Enable leader election? Use in HA mode")
	command.Flags().BoolVarP(&app.zapDevel, "zap-devel", "", false, "Enable development mode for Zap")
	command.Flags().StringVarP(&app.redisHost, "redis-host", "", "redis", "Redis hostname or IP address")
	command.Flags().IntVarP(&app.redisPort, "redis-port", "", 6379, "Redis port number")
	command.Flags().BoolVarP(&app.disableRedis, "disable-redis", "", false, "Disable redis and use in-memory locking mechanism (does not work for multiple instances of the controller)")

	return command
}

type App struct {
	debug                  bool
	metricsBindAddress     string
	healthProbeBindAddress string
	leaderElect            bool
	zapDevel               bool
	redisHost              string
	redisPort              int
	disableRedis           bool
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(riotkitorgv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func (a *App) Run() error {
	if a.debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	var locker locking.Locker
	if a.disableRedis {
		locker = locking.NewInMemoryLocker()
	} else {
		locker = locking.NewRedisDistributedLocker("tcp", a.redisHost, a.redisPort)
	}
	defer locker.Close()

	// zap logger is used by KubeBuilder
	ctrl.SetLogger(zap.New(
		zap.UseDevMode(a.zapDevel),
		zap.ConsoleEncoder(),
	))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		MetricsBindAddress:            a.metricsBindAddress,
		Port:                          9443,
		HealthProbeBindAddress:        a.healthProbeBindAddress,
		LeaderElection:                a.leaderElect,
		LeaderElectionID:              "7b93e0c3.riotkit.org",
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	recorder := mgr.GetEventRecorderFor("backup-maker-operator")
	kubeconfig, err := buildConfig(os.Getenv("KUBECONFIG"))
	if err != nil {
		panic(err.Error())
	}

	// check Kubernetes connection
	dynClient, clErr := dynamic.NewForConfig(kubeconfig)
	if clErr != nil {
		panic(clErr.Error())
	}
	brClient, clErr := v1alpha1.NewForConfig(kubeconfig)
	if clErr != nil {
		panic(clErr.Error())
	}
	integrations := integration.NewAllSupportedJobResourceTypes(kubeconfig)
	fetcher := factory.CachedFetcher{Cache: mgr.GetCache(), Client: brClient}

	if err = (&controllers2.ClusterBackupProcedureTemplateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  mgr.GetCache(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterBackupProcedureTemplate")
		return err
	}
	if err = (&controllers2.ScheduledBackupReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Cache:     mgr.GetCache(),
		BRClient:  brClient,
		RestCfg:   kubeconfig,
		DynClient: dynClient,
		Fetcher:   factory.CachedFetcher{Cache: mgr.GetCache(), Client: brClient},
		Recorder:  recorder,
		Locker:    locker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ScheduledBackup")
		return err
	}
	if err = (&controllers2.RequestedBackupActionReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Cache:     mgr.GetCache(),
		BRClient:  brClient,
		DynClient: dynClient,
		RestCfg:   kubeconfig,
		Fetcher:   fetcher,
		Recorder:  recorder,
		Locker:    locker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RequestedBackupAction")
		return err
	}
	// +kubebuilder:scaffold:builder
	if err = (&controllers2.JobsManagedByRequestedBackupActionObserver{
		Integrations: &integrations,
		Fetcher:      fetcher,
		BRClient:     brClient,
		Client:       mgr.GetClient(),
		Locker:       locker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "JobsManagedByRequestedBackupActionObserver")
		return err
	}
	if err = (&controllers2.JobsManagedByScheduledBackupObserver{
		Integrations: &integrations,
		Fetcher:      fetcher,
		BRClient:     brClient,
		Client:       mgr.GetClient(),
		Locker:       locker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "JobsManagedByScheduledBackupObserver")
		return err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
