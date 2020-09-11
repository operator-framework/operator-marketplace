package operatorsource

import (
	"time"

	wrapper "github.com/operator-framework/operator-marketplace/pkg/client"
	log "github.com/sirupsen/logrus"
)

// NewRegistrySyncer returns a new instance of RegistrySyncer interface.
func NewRegistrySyncer(client wrapper.Client, initialWait time.Duration, resyncInterval time.Duration) RegistrySyncer {
	return &registrySyncer{
		initialWait:    initialWait,
		resyncInterval: resyncInterval,
		poller:         NewPoller(client),
	}
}

// RegistrySyncer is an interface that wraps the Sync method.
//
// Sync kicks off the registry sync operation every N (resync wait time)
// minutes. Sync will stop running once the stop channel is closed.
type RegistrySyncer interface {
	Sync(stop <-chan struct{})
}

// registrySyncer implements RegistrySyncer interface.
type registrySyncer struct {
	initialWait    time.Duration
	resyncInterval time.Duration
	poller         Poller
}

func (s *registrySyncer) Sync(stop <-chan struct{}) {
	log.Infof("[sync] Operator source sync loop will start after %s", s.initialWait)

	// Immediately after the operator process starts, it will spend time in
	// reconciling existing OperatorSource CR(s). Let's give the process a
	// grace period to reconcile and rebuild the local cache from existing CR(s).
	<-time.After(s.initialWait)

	log.Info("[sync] Operator source sync loop has started")
	for {
		select {
		case <-time.After(s.resyncInterval):
			log.Info("[sync] Checking for operator source update(s) in remote registry")
			s.poller.Poll()

		case <-stop:
			log.Info("[sync] Ending operator source watch loop")
			return
		}
	}
}
