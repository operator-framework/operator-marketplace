package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	apiconfigv1 "github.com/openshift/api/config/v1"
	olm "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/signals"
	"github.com/operator-framework/operator-marketplace/pkg/apis"
	configv1 "github.com/operator-framework/operator-marketplace/pkg/apis/config/v1"
	"github.com/operator-framework/operator-marketplace/pkg/catalogsourceconfig"
	"github.com/operator-framework/operator-marketplace/pkg/controller"
	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/operator-framework/operator-marketplace/pkg/migrator"
	"github.com/operator-framework/operator-marketplace/pkg/operatorhub"
	"github.com/operator-framework/operator-marketplace/pkg/operatorsource"
	"github.com/operator-framework/operator-marketplace/pkg/registry"
	"github.com/operator-framework/operator-marketplace/pkg/status"
	sourceCommit "github.com/operator-framework/operator-marketplace/pkg/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	log "github.com/sirupsen/logrus"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	// TODO: resyncInterval is hardcoded to 1 hour now, it would have to be
	// configurable on a per OperatorSource level.
	resyncInterval = time.Duration(60) * time.Minute

	initialWait                = time.Duration(1) * time.Minute
	updateNotificationSendWait = time.Duration(10) * time.Minute
)

var version = flag.Bool("version", false, "displays marketplace source commit info.")

func printVersion() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	// Parse the command line arguments
	flag.StringVar(&registry.ServerImage, "registryServerImage",
		registry.DefaultServerImage, "the image to use for creating the operator registry pod")
	flag.StringVar(&defaults.Dir, "defaultsDir",
		"", "the directory where the default OperatorSources are stored")
	flag.Parse()

	// Check if version flag was set
	if *version {
		fmt.Print(sourceCommit.String())
		// Exit immediately
		os.Exit(0)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("failed to get watch namespace: %v", err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Set OpenShift config API availability
	err = configv1.SetConfigAPIAvailability(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	// Even though we are asking to watch all namespaces, we only handle events
	// from the operator's namespace. The reason for watching all namespaces is
	// watch for CatalogSources in targetNamespaces being deleted and recreate
	// them.
	mgr, err := manager.New(cfg, manager.Options{Namespace: ""})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Registering Components.")

	catalogsourceconfig.InitializeStaticSyncer(mgr.GetClient(), initialWait)
	registrySyncer := operatorsource.NewRegistrySyncer(mgr.GetClient(), initialWait, resyncInterval, updateNotificationSendWait, catalogsourceconfig.Syncer, catalogsourceconfig.Syncer)

	// Setup Scheme for all defined resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		exit(err)
	}

	// Add external resource to scheme
	if err := olm.AddToScheme(mgr.GetScheme()); err != nil {
		exit(err)
	}

	// If the config API is available add the config resources to the scheme
	if configv1.IsAPIAvailable() {
		if err := apiconfigv1.AddToScheme(mgr.GetScheme()); err != nil {
			exit(err)
		}
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		exit(err)
	}

	// Serve a health check.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go http.ListenAndServe(":8080", nil)

	// Wait until this instance becomes the leader.
	log.Info("Waiting to become leader.")
	err = leader.Become(context.TODO(), "marketplace-operator-lock")
	if err != nil {
		log.Error(err, "Failed to retry for leader lock")
		os.Exit(1)
	}
	log.Info("Elected leader.")

	log.Print("Starting the Cmd.")

	// Populate the global default OperatorSources definition and config
	err = defaults.PopulateGlobals()
	if err != nil {
		exit(err)
	}

	// Handle the defaults
	err = ensureDefaults(cfg, mgr.GetScheme())
	if err != nil {
		exit(err)
	}

	stopCh := signals.SetupSignalHandler()

	// set ClusterOperator status to report Migration
	status.ReportMigration(cfg, mgr, namespace, os.Getenv("RELEASE_VERSION"), stopCh)

	client, err := client.New(cfg, client.Options{})
	if err != nil {
		exit(err)
	}

	// Perform migration logic to upgrade cluster from 4.1.z to 4.2.z
	migrator := migrator.NewMigrator(client)
	err = migrator.Migrate(namespace)
	if err != nil {
		exit(err)
	}

	// statusReportingDoneCh will be closed after the operator has successfully stopped reporting ClusterOperator status.
	statusReportingDoneCh := status.StartReporting(cfg, mgr, namespace, os.Getenv("RELEASE_VERSION"), stopCh)

	go registrySyncer.Sync(stopCh)
	go catalogsourceconfig.Syncer.Sync(stopCh)

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
		log.Fatalf("The operator encountered an error, exit code 1: %v", err)
	}

	// No error, graceful termination
	log.Info("The operator exited gracefully, exit code 0")
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
		log.Errorf("Error initializing client for handling defaults - %v", err)
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
