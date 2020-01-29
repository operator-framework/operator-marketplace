package options

import (
	"github.com/operator-framework/operator-marketplace/pkg/status"
)

type ControllerOptions struct {
	SyncSender status.SyncSender
}
