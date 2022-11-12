/*
Copyright 2022 Riotkit.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	riotkitorgv1alpha1 "github.com/riotkit-org/backup-maker-operator/pkg/apis/riotkit/v1alpha1"
	"github.com/riotkit-org/backup-maker-operator/pkg/client/clientset/versioned/typed/riotkit/v1alpha1"
	controllers2 "github.com/riotkit-org/backup-maker-operator/pkg/controllers"
	"github.com/riotkit-org/backup-maker-operator/pkg/factory"
	"github.com/riotkit-org/backup-maker-operator/pkg/integration"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(riotkitorgv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// turn on debug also globally in the logrus
	//if opts.Level.Enabled(zapcore.DebugLevel) {
	logrus.SetLevel(logrus.DebugLevel)
	//}

	// our CRD client
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		MetricsBindAddress:            metricsAddr,
		Port:                          9443,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
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
		os.Exit(1)
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
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ScheduledBackup")
		os.Exit(1)
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
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RequestedBackupAction")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder
	if err = (&controllers2.JobsManagedByRequestedBackupActionObserver{
		Integrations: &integrations,
		Fetcher:      fetcher,
		BRClient:     brClient,
		Client:       mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "JobsManagedByRequestedBackupActionObserver")
		os.Exit(1)
	}
	if err = (&controllers2.JobsManagedByScheduledBackupObserver{
		Integrations: &integrations,
		Fetcher:      fetcher,
		BRClient:     brClient,
		Client:       mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "JobsManagedByScheduledBackupObserver")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
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
