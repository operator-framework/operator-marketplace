package controller

import (
	"github.com/operator-framework/operator-marketplace/pkg/controller/options"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, options.ControllerOptions) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, o options.ControllerOptions) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, o); err != nil {
			return err
		}
	}
	return nil
}
