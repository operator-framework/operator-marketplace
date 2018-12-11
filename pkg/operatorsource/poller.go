package operatorsource

import (
	"fmt"

	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	"github.com/operator-framework/operator-marketplace/pkg/phase"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewPoller returns a new instance of Poller interface.
func NewPoller(client client.Client) Poller {
	poller := &poller{
		datastore: datastore.Cache,
		updater: &updater{
			factory:      appregistry.NewClientFactory(),
			datastore:    datastore.Cache,
			client:       client,
			transitioner: phase.NewTransitioner(),
		},
	}

	return poller
}

// Poller is an interface that wraps the Poll method.
//
// Poll iterates through all available operator source(s) that are in the
// underlying datastore and performs the following action(s):
//   a) It polls the remote registry namespace to check if there are any
//      update(s) available.
//
//   b) If there is an update available then it triggers a purge andrebuild
//      operation for the specified OperatorSource object.
//
// On ay error during each iteration it logs the error encountered and moves
// on ti the next OperatorSource object.
type Poller interface {
	Poll()
}

// poller implements the Poller interface.
type poller struct {
	updater   Updater
	datastore datastore.Writer
}

func (p *poller) Poll() {
	sources := p.datastore.GetAllOperatorSources()

	for _, source := range sources {
		if err := p.poll(source); err != nil {
			log.Errorf("%v", err)
		}
	}
}

func (p *poller) poll(source *datastore.OperatorSourceKey) error {
	updated, err := p.updater.Check(source)
	if err != nil {
		return fmt.Errorf("[sync] error checking for updates [%s] - %v", source.Name, err)
	}

	if !updated {
		return nil
	}

	log.Infof("[sync] remote registry has update(s) - purging OperatorSource [%s]", source.Name)
	deleted, err := p.updater.Trigger(source)
	if err != nil {
		return fmt.Errorf("[sync] error updating object [%s] - %v", source.Name, err)
	}

	if deleted {
		log.Infof("[sync] object deleted [%s] - no action taken", source.Name)
	}

	return nil
}
