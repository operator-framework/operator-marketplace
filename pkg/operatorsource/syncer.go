package operatorsource

import (
	"time"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRegistrySyncer returns a new instance of RegistrySyncer interface.
func NewRegistrySyncer(client client.Client, initialWait time.Duration, resyncWait time.Duration) RegistrySyncer {
	return &registrySyncer{
		initialWait: initialWait,
		resyncWait:  resyncWait,
		poller:      NewPoller(client),
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
	initialWait time.Duration
	resyncWait  time.Duration
	poller      Poller
}

func (s *registrySyncer) Sync(stop <-chan struct{}) {
	log.Infof("[sync] Operator source sync loop will start after %d minutes", s.initialWait)

	// Immediately after the operator process starts, it will spend time in
	// reconciling existing OperatorSource CR(s). Let's give the process a
	// grace period to reconcile and rebuild the local cache from existing CR(s).
	<-time.After(s.initialWait * time.Minute)

	log.Info("[sync] Operator source sync loop has started")
	for {
		select {
		case <-time.After(s.resyncWait * time.Minute):
			log.Debug("[sync] Checking for operator source update(s) in remote registry")
			s.poller.Poll()

		case <-stop:
			log.Info("[sync] Ending operator source watch loop")
			return
		}
	}
}
