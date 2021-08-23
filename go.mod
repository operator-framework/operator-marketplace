module github.com/operator-framework/operator-marketplace

go 1.16

require (
	cloud.google.com/go v0.81.0 // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-openapi/spec v0.19.5
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/openshift/api v0.0.0-20210331193751-3acddb19d360
	github.com/openshift/client-go v0.0.0-20210331195552-cf6c2669e01f
	github.com/openshift/library-go v0.0.0-00010101000000-000000000000
	github.com/operator-framework/api v0.10.5
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/prometheus/client_golang v1.11.0
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.6.0 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78 // indirect
	golang.org/x/term v0.0.0-20210406210042-72f3dc4e9b72 // indirect
	k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver v0.21.4 // indirect
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
	sigs.k8s.io/controller-runtime v0.9.6
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200331152225-585af27e34fd // release-4.5
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200326155132-2a6cd50aedd0 // release-4.5
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20210204141222-0e7715cd7725 // release-4.6
	golang.org/x/text => golang.org/x/text v0.3.3

	k8s.io/client-go => k8s.io/client-go v0.21.4
)
