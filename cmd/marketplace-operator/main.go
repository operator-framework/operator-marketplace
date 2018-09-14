package main

import (
	"context"
	"runtime"
	"time"

	stub "github.com/operator-framework/operator-marketplace/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	sdk.ExposeMetricsPort()

	resource := "marketplace.redhat.com/v1alpha1"
	catalogSourceConfigKind := "CatalogSourceConfig"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}

	// No reason to resync until we implement updating of CatalogSourceConfig CRs
	resyncPeriod := time.Duration(0) * time.Second

	logrus.Infof("Watching %s, %s, %s", resource, catalogSourceConfigKind, namespace)
	sdk.Watch(resource, catalogSourceConfigKind, namespace, resyncPeriod)

	operatorSourceKind := "OperatorSource"
	logrus.Infof("Watching %s, %s, %s, %d", resource, operatorSourceKind, namespace, resyncPeriod)
	sdk.Watch(resource, operatorSourceKind, namespace, resyncPeriod)

	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
