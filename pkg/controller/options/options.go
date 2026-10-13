package options

import "github.com/operator-framework/operator-marketplace/pkg/certificateauthority"

type ControllerOptions struct {
	ClientCAStore *certificateauthority.ClientCAStore
}
