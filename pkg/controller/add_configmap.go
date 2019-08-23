package controller

import (
	"github.com/operator-framework/operator-marketplace/pkg/controller/configmap"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, configmap.Add)
}
