package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"

	apiconfigv1 "github.com/openshift/api/config/v1"

	"github.com/operator-framework/operator-marketplace/pkg/apis"
	configv1 "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	olmv1alpha1 "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/controller"
	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/metrics"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	"github.com/operator-framework/operator-marketplace/pkg/signals"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	sourceCommit "github.com/operator-framework/operator-marketplace/pkg/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// TODO(tflannag): Should this be configurable?
	leaderElectionConfigMapName = "marketplace-operator-lock"
	duration                    = 60 * time.Second
	retryPeriod                 = 2 * time.Second
)

func printVersion() {
	logrus.Printf("Go Version: %s", runtime.Version())
	logrus.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Printf("operator-sdk Version: %v", sdkVersion.Version)
}

func setupScheme() *kruntime.Scheme {
	scheme := kruntime.NewScheme()

	utilruntime.Must(apis.AddToScheme(scheme))
	utilruntime.Must(olmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))

	if configv1.IsAPIAvailable() {
		utilruntime.Must(apiconfigv1.AddToScheme(scheme))
	}

	return scheme
}

func main() {
	printVersion()

	var (
		clusterOperatorName string
		tlsKeyPath          string
		tlsCertPath         string
		version             bool
	)
	flag.StringVar(&clusterOperatorName, "clusterOperatorName", "", "the name of the OpenShift ClusterOperator that should reflect this operator's status, or the empty string to disable ClusterOperator updates")
	flag.StringVar(&defaults.Dir, "defaultsDir", "", "the directory where the default CatalogSources are stored")
	flag.BoolVar(&version, "version", false, "displays marketplace source commit info.")
	flag.StringVar(&tlsKeyPath, "tls-key", "", "Path to use for private key (requires tls-cert)")
	flag.StringVar(&tlsCertPath, "tls-cert", "", "Path to use for certificate (requires tls-key)")
	flag.Parse()

	logger := logrus.New()

	// Check if version flag was set
	if version {
		logger.Infof("%s", sourceCommit.String())
		os.Exit(0)
	}

	// set TLS to serve metrics over a secure channel if cert is provided
	// cert is provided by default by the marketplace-trusted-ca volume mounted as part of the marketplace-operator deployment
	err := metrics.ServePrometheus(tlsCertPath, tlsKeyPath)
	if err != nil {
		logger.Fatalf("failed to serve prometheus metrics: %s", err)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logger.Fatalf("failed to get watch namespace: %v", err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logger.Fatal(err)
	}

	// Set OpenShift config API availability
	err = configv1.SetConfigAPIAvailability(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("setting up scheme")
	scheme := setupScheme()

	// Even though we are asking to watch all namespaces, we only handle events
	// from the operator's namespace. The reason for watching all namespaces is
	// watch for CatalogSources in targetNamespaces being deleted and recreate
	// them.
	//
	// Note(tflannag): Setting the `MetricsBindAddress` to `0` here disables the
	// metrics listener from controller-runtime. Previously, this was disabled by
	// default in <v0.2.0, but it's now enabled by default and the default port
	// conflicts with the same port we bind for the health checks.
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          "",
		MetricsBindAddress: "0",
		Scheme:             scheme,
	})
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Registering Components.")
	ctx := signals.Context()
	stopCh := ctx.Done()

	var statusReporter status.Reporter = &status.NoOpReporter{}
	if clusterOperatorName != "" {
		statusReporter, err = status.NewReporter(cfg, mgr, namespace, clusterOperatorName, os.Getenv("RELEASE_VERSION"), stopCh)
		if err != nil {
			logger.Fatal(err)
		}
	}

	// Populate the global default OperatorSources definition and config
	err = defaults.PopulateGlobals()
	if err != nil {
		logger.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, options.ControllerOptions{}); err != nil {
		logger.Fatal(err)
	}

	logger.Info("Setting up health checks")
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		logger.Fatal(fmt.Errorf("failed to initialize the kubernetes client config: %v", err))
	}
	client, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		logger.Fatal(fmt.Errorf("failed to initialize the kubernetes clientset: %v", err))
	}
	// Wait until this instance becomes the leader.
	logger.Info("Waiting to become leader.")
	leader, err := setupLeaderElection(logger, client, leaderElectionConfigMapName, namespace, mgr.GetEventRecorderFor(mgr.GetScheme().Name()))
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("Elected leader")
	go leader.Run(ctx)

	logger.Info("Starting the Cmd.")

	// Handle the defaults
	err = ensureDefaults(cfg, mgr.GetScheme())
	if err != nil {
		logger.Fatal(err)
	}

	// statusReportingDoneCh will be closed after the operator has successfully stopped reporting ClusterOperator status.
	statusReportingDoneCh := statusReporter.StartReporting()

	// Start the Cmd
	err = mgr.Start(stopCh)

	// Wait for ClusterOperator status reporting routine to close the statusReportingDoneCh channel.
	<-statusReportingDoneCh

	exit(err)
}

// exit stops the reporting of ClusterOperator status and exits with the proper exit code.
func exit(err error) {
	// If an error exists then exit with status set to 1
	if err != nil {
		logrus.Fatalf("The operator encountered an error, exit code 1: %v", err)
	}

	// No error, graceful termination
	logrus.Info("The operator exited gracefully, exit code 0")
	os.Exit(0)
}

// ensureDefaults ensures that all the default OperatorSources are present on
// the cluster
func ensureDefaults(cfg *rest.Config, scheme *kruntime.Scheme) error {
	// The default client serves read requests from the cache which only gets
	// initialized after mgr.Start(). So we need to instantiate a new client
	// for the defaults handler.
	clientForDefaults, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		logrus.Errorf("Error initializing client for handling defaults - %v", err)
		return err
	}

	if configv1.IsAPIAvailable() {
		// Check if the cluster OperatorHub config resource is present.
		operatorHubCluster := &apiconfigv1.OperatorHub{}
		err = clientForDefaults.Get(context.TODO(), client.ObjectKey{Name: operatorhub.DefaultName}, operatorHubCluster)

		// The default OperatorHub config resource is present which will take care of ensuring defaults
		if err == nil {
			return nil
		}
	}

	// Ensure that the default OperatorSources are present based on the definitions
	// in the defaults directory
	result := defaults.New(defaults.GetGlobals()).EnsureAll(clientForDefaults)
	if len(result) != 0 {
		return fmt.Errorf("[defaults] Error ensuring default OperatorSource(s) - %v", result)
	}

	return nil
}

// TODO(tflannag): Investigate making this a custom type instead

func setupLeaderElection(
	logger logrus.FieldLogger,
	client kubernetes.Interface,
	name,
	namespace string,
	recorder resourcelock.EventRecorder,
) (*leaderelection.LeaderElector, error) {
	id, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	rl, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		namespace,
		name,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating lock %v", err)
	}

	// TODO(tflannag): Properly handle signals
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	leader, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: duration,
		RenewDeadline: duration / 2,
		RetryPeriod:   retryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				logger.WithFields(logrus.Fields{
					"lock":     rl.Describe(),
					"identify": rl.Identity(),
					"type":     resourcelock.ConfigMapsResourceLock,
				}).Info("elected leader")
			},
			OnStoppedLeading: func() {
				logger.Warn("leader election lost")
				cancel()
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating leader elector: %v", err)
	}

	return leader, nil
}
