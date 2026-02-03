package main

import (
	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	ca "github.com/operator-framework/operator-marketplace/pkg/certificateauthority"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiconfigv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/apiserver"

	"github.com/operator-framework/operator-marketplace/pkg/apis"
	configv1 "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	apiutils "github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	"github.com/operator-framework/operator-marketplace/pkg/controller"
	"github.com/operator-framework/operator-marketplace/pkg/controller/configmap"
	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/metrics"
	"github.com/operator-framework/operator-marketplace/pkg/signals"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	sourceCommit "github.com/operator-framework/operator-marketplace/pkg/version"

	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	// TODO(tflannag): Should this be configurable?
	defaultLeaderElectionConfigMapName = "marketplace-operator-lock"
	defaultRetryPeriod                 = 30 * time.Second
	defaultRenewDeadline               = 60 * time.Second
	defaultLeaseDuration               = 90 * time.Second
	defaultPprofPort                   = 6060
	healthPort                         = 8080
)

func init() {
	log.SetLogger(zap.New())
}

func printVersion() {
	logrus.Printf("Go Version: %s", runtime.Version())
	logrus.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
}

func setupScheme() *kruntime.Scheme {
	scheme := kruntime.NewScheme()

	utilruntime.Must(apis.AddToScheme(scheme))
	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	if configv1.IsAPIAvailable() {
		utilruntime.Must(apiconfigv1.AddToScheme(scheme))
	}

	return scheme
}

func main() {
	printVersion()

	var (
		clusterOperatorName     string
		tlsKeyPath              string
		tlsCertPath             string
		leaderElectionNamespace string
		pprofAddress            string
		version                 bool
		loglvl                  string
	)
	flag.StringVar(&clusterOperatorName, "clusterOperatorName", "", "configures the name of the OpenShift ClusterOperator that should reflect this operator's status, or the empty string to disable ClusterOperator updates")
	flag.StringVar(&defaults.Dir, "defaultsDir", "", "configures the directory where the default CatalogSources are stored")
	flag.BoolVar(&version, "version", false, "displays marketplace source commit info.")
	flag.StringVar(&pprofAddress, "pprof-address", fmt.Sprintf(":%d", defaultPprofPort), "Address to serve pprof endpoints on.")
	flag.StringVar(&tlsKeyPath, "tls-key", "", "Path to use for private key (requires tls-cert)")
	flag.StringVar(&tlsCertPath, "tls-cert", "", "Path to use for certificate (requires tls-key)")
	flag.StringVar(&leaderElectionNamespace, "leader-namespace", "openshift-marketplace", "configures the namespace that will contain the leader election lock")
	flag.StringVar(&loglvl, "level", "info", "Sets level of logger with default verbosity info level. See https://github.com/sirupsen/logrus for other verbosity levels.")
	flag.Parse()
	logger := logrus.New()

	// Set verbosity level
	parsedLevel, err := logrus.ParseLevel(loglvl)
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	logger.SetLevel(parsedLevel)

	// Check if version flag was set
	if version {
		logger.Infof("%s", sourceCommit.String())
		os.Exit(0)
	}

	namespace, err := apiutils.GetWatchNamespace()
	if err != nil {
		logger.Fatalf("failed to get watch namespace: %v", err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logger.Fatal(err)
	}

	// Set OpenShift config API availability
	if err := configv1.SetConfigAPIAvailability(cfg); err != nil {
		logger.Fatal(err)
	}

	logger.Info("setting up scheme")
	scheme := setupScheme()

	// Note(tflannag): Setting the `MetricsBindAddress` to `0` here disables the
	// metrics listener from controller-runtime. Previously, this was disabled by
	// default in <v0.2.0, but it's now enabled by default and the default port
	// conflicts with the same port we bind for the health checks.
	mgr, err := manager.New(cfg, manager.Options{
		Metrics:          metricsserver.Options{BindAddress: "0"},
		PprofBindAddress: pprofAddress,
		Scheme:           scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&corev1.ConfigMap{}: {
					Namespaces: map[string]cache.Config{
						namespace: {
							FieldSelector: fields.SelectorFromSet(fields.Set{
								"metadata.name": ca.TrustedCaConfigMapName,
							}),
						},
						configmap.ClientCANamespace: {
							FieldSelector: fields.SelectorFromSet(fields.Set{
								"metadata.name": configmap.ClientCAConfigMapName,
							}),
						},
					},
				},
				// Restrict CatalogSource watch to the operator's namespace only.
				// The operator only manages default CatalogSources (redhat-operators,
				// certified-operators, community-operators) in this namespace.
				&olmv1alpha1.CatalogSource{}: {
					Namespaces: map[string]cache.Config{
						namespace: {},
					},
				},
			},
		},
	})
	if err != nil {
		logger.Fatal(err)
	}

	// Setup APIServer TLS configuration for HTTPS servers
	apiServerTLSQuerier, apiServerFactory, err := apiserver.SetupAPIServerTLSConfig(logger, cfg)
	if err != nil {
		logger.Fatalf("error setting up APIServer TLS configuration: %v", err)
	}

	clientCAStore := ca.NewClientCAStore(x509.NewCertPool())
	// Best effort attempt to fetch client rootCA
	// Should not fail if this does not immediately succeed, the configmap controller will continue to
	// watch for the right configmap for updating this certpool as soon as it is created.
	caData, err := configmap.GetClientCAFromConfigMap(context.TODO(), mgr.GetClient(), types.NamespacedName{Name: configmap.ClientCAConfigMapName, Namespace: configmap.ClientCANamespace})
	if err == nil && len(caData) > 0 {
		clientCAStore.Update(caData)
	} else if err != nil {
		logger.Warn("failed to initialize client CA certPool for the metrics endpoint: %w", err)
	} else if len(caData) == 0 {
		logger.Warn("could not find client CA to initialize client rootCA certpool, the clientCA configMap may not be initialized properly yet")
	}

	// set TLS to serve metrics over a secure channel if cert is provided
	// cert is provided by default by the marketplace-trusted-ca volume mounted as part of the marketplace-operator deployment
	if err := metrics.ServePrometheus(tlsCertPath, tlsKeyPath, clientCAStore, apiServerTLSQuerier); err != nil {
		logger.Fatalf("failed to serve prometheus metrics: %s", err)
	}

	logger.Info("setting up health checks")
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", healthPort), nil)

	ctx, cancel := context.WithCancel(signals.Context())
	defer cancel()

	run := func(ctx context.Context) {
		stopCh := ctx.Done()
		logger.Info("registering components")
		var statusReporter status.Reporter = &status.NoOpReporter{}
		if clusterOperatorName != "" {
			logger.Info("setting up the marketplace clusteroperator status reporter")
			statusReporter, err = status.NewReporter(cfg, mgr, namespace, clusterOperatorName, os.Getenv("RELEASE_VERSION"), stopCh)
			if err != nil {
				logger.Fatal(err)
			}
		}

		// Populate the global default CatalogSource definitions and config
		if err := defaults.PopulateGlobals(); err != nil {
			logger.Fatal(err)
		}

		logger.Info("setting up controllers")
		if err := controller.AddToManager(mgr, options.ControllerOptions{ClientCAStore: clientCAStore}); err != nil {
			logger.Fatal(err)
		}

		// Start APIServer TLS informer factory if on OpenShift
		if apiServerFactory != nil {
			apiServerFactory.Start(ctx.Done())
		}

		// start reporting the marketplace clusteroperator status reporting before
		// starting the manager instance as mgr.Start is blocking
		logger.Info("starting the marketplace clusteroperator status reporter")
		statusReportingDoneCh := statusReporter.StartReporting()

		logger.Info("starting manager")
		if err := mgr.Start(ctx); err != nil {
			logger.WithError(err).Error("unable to run manager")
		}

		// Wait for ClusterOperator status reporting routine to close the statusReportingDoneCh channel.
		<-statusReportingDoneCh
	}

	client, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		logger.Fatal(fmt.Errorf("failed to initialize the kubernetes clientset: %v", err))
	}

	id := os.Getenv("POD_NAME")
	if id == "" {
		logger.Warn("failed to determine $POD_NAME falling back to hostname")
		id, err = os.Hostname()
		if err != nil {
			logger.Fatal(err)
		}
	}

	rl, err := resourcelock.New(resourcelock.LeasesResourceLock, leaderElectionNamespace, defaultLeaderElectionConfigMapName, client.CoreV1(), client.CoordinationV1(), resourcelock.ResourceLockConfig{
		Identity:      id,
		EventRecorder: record.NewBroadcaster().NewRecorder(scheme, corev1.EventSource{Component: id}),
	})
	if err != nil {
		logger.Fatal(err)
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            rl,
		ReleaseOnCancel: true,
		LeaseDuration:   defaultLeaseDuration,
		RenewDeadline:   defaultRenewDeadline,
		RetryPeriod:     defaultRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.Infof("became leader: %s", id)
				run(ctx)
			},
			OnStoppedLeading: func() {
				logger.Warnf("leader election lost for %s identity", id)
				// Stop the controller just in case this doesn't coincide with container stop
				// e.g. scale > 1 (which we don't support today and would require the ability
				// to start/stop reconciliation dynamically)
				cancel()
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					return
				}
				logger.Infof("current leader: %s", identity)
			},
		},
	})
}
