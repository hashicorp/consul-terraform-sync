package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/mitchellh/mapstructure"
)

const taskPath = "tasks"

// taskHandler handles the tasks endpoint
type taskHandler struct {
	store   *event.Store
	drivers map[string]driver.Driver
	version string
}

// newTaskHandler returns a new taskHandler
func newTaskHandler(store *event.Store, drivers map[string]driver.Driver,
	version string) *taskHandler {

	return &taskHandler{
		store:   store,
		drivers: drivers,
		version: version,
	}
}

// ServeHTTP serves the tasks endpoint
func (h *taskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[TRACE] (api.task) requesting tasks '%s'", r.URL.Path)

	switch r.Method {
	case http.MethodPatch:
		h.updateTask(w, r)
	default:
		err := fmt.Errorf("'%s' in an unsupported method. The task API "+
			"currently supports the method(s): '%s'", r.Method, http.MethodPatch)
		log.Printf("[TRACE] (api.task) unsupported method: %s", err)
		jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{
			"error": err.Error(),
		})
	}
}

// UpdateTaskConfig contains the fields available for patch updating a task.
// Not all task configuration is available for update
type UpdateTaskConfig struct {
	Enabled *bool `mapstructure:"enabled"`
}

// updateTask does a patch update to an existing task
func (h *taskHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	taskName, err := getTaskName(r.URL.Path, taskPath, h.version)
	if err != nil {
		log.Printf("[TRACE] (api.task) bad request: %s", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})

		return
	}

	if taskName == "" {
		err := fmt.Errorf("No taskname was included in the api request. " +
			"Updating a task requires the taskname: '/v1/tasks/:task_name'")
		log.Printf("[TRACE] (api.task) bad request: %s", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	d, ok := h.drivers[taskName]
	if !ok {
		err := fmt.Errorf("A task with the name '%s' does not exist or has not "+
			"been initialized yet", taskName)
		log.Printf("[TRACE] (api.task) task not found: %s", err)
		jsonResponse(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("[TRACE] (api.task) unable to read request body: %s", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	conf, err := decodeBody(body)
	if err != nil {
		log.Printf("[TRACE] (api.task) problem decoding body: %s", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	patch := driver.PatchTask{}
	if conf.Enabled != nil {
		log.Printf("[INFO] (api.task) Updating task to be enabled=%t",
			config.BoolVal(conf.Enabled))
		patch.Enabled = config.BoolVal(conf.Enabled)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = d.UpdateTask(ctx, patch) // TODO: consume plan in next commit
	if err != nil {
		log.Printf("[TRACE] (api.task) error while updating task: %s", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, "")
}

func decodeBody(body []byte) (UpdateTaskConfig, error) {
	var raw map[string]interface{}

	err := json.Unmarshal(body, &raw)
	if err != nil {
		return UpdateTaskConfig{}, err
	}

	var config UpdateTaskConfig
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &config,
	})
	if err != nil {
		return UpdateTaskConfig{}, err
	}

	if err = decoder.Decode(raw); err != nil {
		return UpdateTaskConfig{}, err
	}

	if len(md.Unused) > 0 {
		sort.Strings(md.Unused)
		err := fmt.Errorf("request body's JSON contains unsupported keys: %s",
			strings.Join(md.Unused, ", "))
		return UpdateTaskConfig{}, err
	}

	return config, nil
}
