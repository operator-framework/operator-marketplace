//go:build tools
// +build tools

package tools

import (
	_ "github.com/mikefarah/yq/v3"
	_ "github.com/openshift/build-machinery-go"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
)
