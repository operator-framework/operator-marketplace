package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

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
	"github.com/operator-framework/operator-sdk/pkg/leader"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	leaderElectionConfigMapName = "marketplace-operator-lock"
)

func printVersion() {
	logrus.Printf("Go Version: %s", runtime.Version())
	logrus.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Printf("operator-sdk Version: %v", sdkVersion.Version)
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

	// Check if version flag was set
	if version {
		logrus.Infof("%s", sourceCommit.String())
		os.Exit(0)
	}

	// set TLS to serve metrics over a secure channel if cert is provided
	// cert is provided by default by the marketplace-trusted-ca volume mounted as part of the marketplace-operator deployment
	err := metrics.ServePrometheus(tlsCertPath, tlsKeyPath)
	if err != nil {
		logrus.Fatalf("failed to serve prometheus metrics: %s", err)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logrus.Fatal(err)
	}

	// Set OpenShift config API availability
	err = configv1.SetConfigAPIAvailability(cfg)
	if err != nil {
		logrus.Fatal(err)
	}

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
	})
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Registering Components.")
	logrus.Info("setting up scheme")
	// Setup Scheme for all defined resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.Fatal(err)
	}
	// Add external resource to scheme
	if err := olmv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.Fatal(err)
	}
	if err := v1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.Fatal(err)
	}
	// If the config API is available add the config resources to the scheme
	if configv1.IsAPIAvailable() {
		if err := apiconfigv1.AddToScheme(mgr.GetScheme()); err != nil {
			logrus.Fatal(err)
		}
	}

	stopCh := signals.Context().Done()

	var statusReporter status.Reporter = &status.NoOpReporter{}
	if clusterOperatorName != "" {
		statusReporter, err = status.NewReporter(cfg, mgr, namespace, clusterOperatorName, os.Getenv("RELEASE_VERSION"), stopCh)
		if err != nil {
			logrus.Fatal(err)
		}
	}

	// Populate the global default OperatorSources definition and config
	err = defaults.PopulateGlobals()
	if err != nil {
		logrus.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, options.ControllerOptions{}); err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Setting up health checks")
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	// Wait until this instance becomes the leader.
	logrus.Info("Waiting to become leader.")
	err = leader.Become(context.TODO(), leaderElectionConfigMapName)
	if err != nil {
		logrus.Fatal(err, "Failed to retry for leader lock")
	}
	logrus.Info("Elected leader.")

	logrus.Info("Starting the Cmd.")

	// Handle the defaults
	err = ensureDefaults(cfg, mgr.GetScheme())
	if err != nil {
		logrus.Fatal(err)
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
