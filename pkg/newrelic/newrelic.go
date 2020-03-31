package newrelic

import (
	"os"
	"sync"
	"time"

	newrelic "github.com/newrelic/go-agent"
	"github.com/pkg/errors"
)

var NRApp newrelic.Application
var nrCreationMutex sync.Mutex

func GetNewRelicApp() (newrelic.Application, error) {
	if NRApp == nil {
		// if there is no key provided don't use newrelic
		newRelicKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
		if newRelicKey == "" {
			return nil, nil
		}

		// only lock if we might have to create the app
		nrCreationMutex.Lock()
		defer nrCreationMutex.Unlock()

		// check to see if some other thread created the app while waiting for the mutex
		// if not, create the app
		if NRApp == nil {
			cfg := newrelic.NewConfig("kots-lint", newRelicKey)
			cfg.Logger = newrelic.NewLogger(os.Stdout)
			cfg.Labels["env"] = "production"
			cfg.TransactionEvents.Enabled = true
			cfg.DatastoreTracer.QueryParameters.Enabled = true
			cfg.DatastoreTracer.InstanceReporting.Enabled = true
			cfg.DatastoreTracer.SlowQuery.Enabled = true
			cfg.DatastoreTracer.SlowQuery.Threshold = time.Millisecond * 10
			cfg.TransactionTracer.Enabled = true
			app, err := newrelic.NewApplication(cfg)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create newrelic app")
			}
			NRApp = app
		}
	}
	return NRApp, nil
}
