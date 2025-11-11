package configmap_test

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	"github.com/operator-framework/operator-marketplace/pkg/certificateauthority"
	"github.com/operator-framework/operator-marketplace/pkg/controller/configmap"
	"github.com/operator-framework/operator-marketplace/pkg/metrics"
)

type certKeyPair struct {
	serverName string
	key *rsa.PrivateKey
	keyPEM []byte
	cert *x509.Certificate
	certPEM []byte
}

func (i *certKeyPair) generateTestCert(parentCert *x509.Certificate, parentKey *rsa.PrivateKey) error {
	randSource := rand.New(rand.NewSource(0))
	if i.key == nil {
		// generate key once, if it does not exist.
		key, err := rsa.GenerateKey(randSource, 1024)
		if err != nil {
			return err
		}
		i.key = key
		keyPEMBytes := bytes.Buffer{}
		if err := pem.Encode(&keyPEMBytes, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(i.key)}); err != nil {
			return err
		}
		i.keyPEM = keyPEMBytes.Bytes()
	}

	templateCert := &x509.Certificate{NotAfter: metav1.Now().Add(5*time.Minute)}
	if len(i.serverName) != 0 {
		templateCert.DNSNames = []string{i.serverName}
	}
	if parentCert == nil && parentKey == nil {
		// root CA
		templateCert.MaxPathLen = 0
		templateCert.MaxPathLenZero = true
		templateCert.BasicConstraintsValid = true
		templateCert.IsCA = true
		templateCert.KeyUsage = x509.KeyUsageCertSign
		parentKey = i.key
		parentCert = templateCert
	} else if parentKey == nil || parentCert == nil {
		return fmt.Errorf("Both parentCert and parentKey must be non-nil for non-rootCA certificates")
	}
	certBytes, err := x509.CreateCertificate(randSource, templateCert, parentCert, i.key.Public(), parentKey)
	if err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}
	i.cert = cert

	certPEMBytes := bytes.Buffer{}
	if err := pem.Encode(&certPEMBytes, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return err
	}
	i.certPEM = certPEMBytes.Bytes()

	return nil
}

func updateTestCAConfigMap(client crclient.WithWatch, reconciler *configmap.ReconcileConfigMap, caPEM []byte) error {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: configmap.ClientCAConfigMapName, Namespace: configmap.ClientCANamespace}, Data:map[string]string{configmap.ClientCAKey: string(caPEM)}}
	if err := client.Create(context.TODO(), cm); err != nil {
		if errors.IsAlreadyExists(err) {
			err = client.Update(context.TODO(), cm)
		}
		if err != nil {
			return err
		}
	}
	if _, err := reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}}); err != nil {
		return err
	}
	return nil
}

func makeRequest(t *testing.T, certPool *x509.CertPool, clientCert *tls.Certificate, serverName string, expectedCond func (*http.Response, error) bool) {
	httpClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
			ServerName: serverName,
		},
	}}

	if clientCert != nil {
		httpClient.Transport.(*http.Transport).TLSClientConfig.Certificates =[]tls.Certificate{*clientCert}
	}

	require.Eventually(t, func() bool{
		response, err := httpClient.Get("https://:8081/metrics")
		return expectedCond(response, err)
	}, 5*time.Second, 1*time.Second)
}

func TestConfigMapClientCAAuth(t *testing.T) {
	caStore := certificateauthority.NewClientCAStore(x509.NewCertPool())
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, openshiftconfigv1.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: configmap.ClientCANamespace}}).Build()
	mgr, err := manager.New(&rest.Config{}, manager.Options{NewClient:  func(config *rest.Config, options crclient.Options) (crclient.Client, error){return client, nil}})
	require.NoError(t, err)
	reconciler := configmap.NewReconciler(mgr, caStore)

	// client CA and key
	testClientName := "test.client"
	clientCAKeyCertPair := certKeyPair{serverName: testClientName}
	require.NoError(t, clientCAKeyCertPair.generateTestCert(nil, nil))

	// Update configmap with current client rootCA
	require.NoError(t, updateTestCAConfigMap(client, reconciler, clientCAKeyCertPair.certPEM))

	// client cert signed by CA
	clientKeyCertPair := certKeyPair{}
	require.NoError(t, clientKeyCertPair.generateTestCert(clientCAKeyCertPair.cert, clientCAKeyCertPair.key))
	tlsClientCert, err := tls.X509KeyPair(clientKeyCertPair.certPEM, clientKeyCertPair.keyPEM)
	require.NoError(t, err)

	// self-signed server cert-key pair
	testServerName := "test.server"
	serverKeyCertPair := certKeyPair{serverName: testServerName}
	require.NoError(t, serverKeyCertPair.generateTestCert(nil, nil))

	// create cert and key files for metrics server
	testDir, err := os.MkdirTemp("", "client-auth-")
	defer os.RemoveAll(testDir)
	require.NoError(t, err)
	certFile := filepath.Join(testDir, "server.crt")
	keyFile := filepath.Join(testDir, "server.key")
	require.NoError(t, os.WriteFile(keyFile, serverKeyCertPair.keyPEM, fs.ModePerm|os.FileMode(os.O_CREATE|os.O_RDWR)))
	require.NoError(t, os.WriteFile(certFile, serverKeyCertPair.certPEM, fs.ModePerm|os.FileMode(os.O_CREATE|os.O_RDWR)))

	// start metrics HTTPS server
	go func() {
		if err := metrics.ServePrometheus(certFile, keyFile, caStore); err != nil {
			panic(err)
		}
	}()

	// certpool used to validate server certs, with server cert added as a valid CA
	serverCertPool := x509.NewCertPool()
	serverCertPool.AddCert(serverKeyCertPair.cert)

	// Fail unauthenticated client request
	makeRequest(t, serverCertPool, nil, testServerName, func(_ *http.Response, err error) bool {
		return err != nil && strings.Contains(err.Error(), "tls: certificate required")
	})

	// Succeed when providing client cert
	makeRequest(t, serverCertPool, &tlsClientCert, testServerName, func(response *http.Response, err error) bool {
		return response != nil && response.StatusCode == 200
	})

	// Fail when client uses new CA before update
	clientCAKeyCertPairNew := certKeyPair{serverName: testClientName}
	require.NoError(t, clientCAKeyCertPairNew.generateTestCert(nil, nil))
	require.NoError(t, clientKeyCertPair.generateTestCert(clientCAKeyCertPairNew.cert, clientCAKeyCertPairNew.key))
	tlsClientCert, err = tls.X509KeyPair(clientKeyCertPair.certPEM, clientKeyCertPair.keyPEM)
	require.NoError(t, err)
	makeRequest(t, serverCertPool, &tlsClientCert, testServerName, func(response *http.Response, err error) bool {
		return err != nil && strings.Contains(err.Error(), "tls: unknown certificate authority")
	})

	// Succeed after reconciling the clientCA ConfigMap
	require.NoError(t, updateTestCAConfigMap(client, reconciler, clientCAKeyCertPairNew.certPEM))
	makeRequest(t, serverCertPool, &tlsClientCert, testServerName, func(response *http.Response, err error) bool {
		return response != nil && response.StatusCode == 200
	})
}
