package actions

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo/worker"
	"github.com/gomods/athens/pkg/eventlog"
	"github.com/gomods/athens/pkg/payloads"
)

// GetCacheMissReporterJob porcesses queue of cache misses and reports them to Olympus
func GetCacheMissReporterJob(w worker.Worker) worker.Handler {
	return func(args worker.Args) (err error) {
		module, version, err := parseArgs(args)
		if err != nil {
			return err
		}

		if err := reportCacheMiss(module, version); err != nil {
			return err
		}

		return queueCacheMissFetch(module, version, w)
	}
}

func reportCacheMiss(module, version string) error {
	cm := payloads.Module{Name: module, Version: version}
	content, err := json.Marshal(cm)
	if err != nil {
		return err
	}

	olympusEndpoint := getCurrentOlympus()
	if olympusEndpoint == "" {
		return nil
	}

	req, err := http.NewRequest("POST", olympusEndpoint+"/cachemiss", bytes.NewBuffer(content))
	if err != nil {
		return err
	}

	client := http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			updateCurrentOlympus(req.URL.String())
			return &eventlog.ErrUseNewOlympus{Endpoint: req.URL.String()}
		},
	}

	_, err = client.Do(req)
	return err
}

func queueCacheMissFetch(module, version string, w worker.Worker) error {
	return w.Perform(worker.Job{
		Queue:   workerQueue,
		Handler: FetcherWorkerName,
		Args: worker.Args{
			workerModuleKey:   module,
			workerVersionKey:  version,
			workerTryCountKey: maxTryCount,
		},
	})
}
