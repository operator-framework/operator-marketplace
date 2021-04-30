package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"

	apiconfigv1 "github.com/openshift/api/config/v1"

	"github.com/operator-framework/operator-marketplace/pkg/apis"
	configv1 "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	olmv1alpha1 "github.com/operator-framework/operator-marketplace/pkg/apis/olm/v1alpha1"
	"github.com/operator-framework/operator-marketplace/pkg/builders"
	"github.com/operator-framework/operator-marketplace/pkg/controller"
	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/metrics"
	"github.com/operator-framework/operator-marketplace/pkg/migrator"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	"github.com/operator-framework/operator-marketplace/pkg/signals"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	sourceCommit "github.com/operator-framework/operator-marketplace/pkg/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
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
		logrus.Infof(sourceCommit.String())
		os.Exit(0)
	}

	// set TLS to serve metrics over a secure channel if cert is provided
	// cert is provided by default by the marketplace-trusted-ca volume mounted as part of the marketplace-operator deployment
	err := metrics.ServePrometheus(tlsCertPath, tlsKeyPath)
	if err != nil {
		logrus.Errorf("failed to serve prometheus metrics: %s", err)
		os.Exit(1)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Errorf("failed to get watch namespace: %v", err)
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	// Set OpenShift config API availability
	err = configv1.SetConfigAPIAvailability(cfg)
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	// Even though we are asking to watch all namespaces, we only handle events
	// from the operator's namespace. The reason for watching all namespaces is
	// watch for CatalogSources in targetNamespaces being deleted and recreate
	// them.
	mgr, err := manager.New(cfg, manager.Options{Namespace: ""})
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	logrus.Info("Registering Components.")
	logrus.Info("setting up scheme")
	// Setup Scheme for all defined resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	// Add external resource to scheme
	if err := olmv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	if err := v1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
	// If the config API is available add the config resources to the scheme
	if configv1.IsAPIAvailable() {
		if err := apiconfigv1.AddToScheme(mgr.GetScheme()); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
	}

	stopCh := signals.Context().Done()

	var statusReporter status.Reporter = &status.NoOpReporter{}
	if clusterOperatorName != "" {
		statusReporter, err = status.NewReporter(cfg, mgr, namespace, clusterOperatorName, os.Getenv("RELEASE_VERSION"), stopCh)
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
	}

	// Populate the global default OperatorSources definition and config
	err = defaults.PopulateGlobals()
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, options.ControllerOptions{}); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	// Serve a health check.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	// Wait until this instance becomes the leader.
	logrus.Info("Waiting to become leader.")
	err = leader.Become(context.TODO(), leaderElectionConfigMapName)
	if err != nil {
		logrus.Error(err, "Failed to retry for leader lock")
		os.Exit(1)
	}
	logrus.Info("Elected leader.")

	logrus.Info("Starting the Cmd.")

	// migrate away from Marketplace API
	clientGo, err := client.New(cfg, client.Options{Scheme: mgr.GetScheme()})
	if err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Error(err, "Failed to instantiate the client for migrator")
		os.Exit(1)
	}
	migrator := migrator.New(clientGo)
	err = migrator.Migrate()
	if err != nil {
		logrus.Error(err, "[migration] Error while migrating Marketplace away from OperatorSource API")
	}

	err = cleanUpOldOpsrcResources(clientGo)
	if err != nil {
		logrus.Error(err, "OperatorSource child resource cleanup failed")
	}

	// Handle the defaults
	err = ensureDefaults(cfg, mgr.GetScheme())
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	err = defaults.RemoveObsoleteOpsrc(clientGo)
	if err != nil {
		logrus.Error(err, "[defaults] Could not remove the obsolete default OperatorSource(s)")
	}
	// statusReportingDoneCh will be closed after the operator has successfully stopped reporting ClusterOperator status.
	statusReportingDoneCh := statusReporter.StartReporting()

	// Start the Cmd
	err = mgr.Start(stopCh)

	// Wait for ClusterOperator status reporting routine to close the statusReportingDoneCh channel.
	<-statusReportingDoneCh

	exit(err)
}

// TODO(tflannag): Why aren't we passing a logrus.FieldLogger here?
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

// cleanUpOldOpsrcResources cleans up old deployments and services associated with OperatorSources
func cleanUpOldOpsrcResources(kubeClient client.Client) error {
	ctx := context.TODO()

	deploy := &appsv1.DeploymentList{}
	svc := &corev1.ServiceList{}
	o := []client.ListOption{
		client.MatchingLabels{builders.OpsrcOwnerNameLabel: builders.OpsrcOwnerNamespaceLabel},
	}

	var allErrors []error
	if err := kubeClient.List(ctx, deploy, o...); err == nil {
		for _, d := range deploy.Items {
			if err := kubeClient.Delete(ctx, &d); err != nil {
				allErrors = append(allErrors, err)
			}
		}
	} else {
		allErrors = append(allErrors, err)
	}
	if err := kubeClient.List(ctx, svc, o...); err == nil {
		for _, s := range svc.Items {
			if err := kubeClient.Delete(ctx, &s); err != nil {
				allErrors = append(allErrors, err)
			}
		}
	} else {
		allErrors = append(allErrors, err)
	}
	return utilerrors.NewAggregate(allErrors)
}
