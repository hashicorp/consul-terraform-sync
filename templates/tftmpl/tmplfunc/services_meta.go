package tmplfunc

import (
	"fmt"
)

// ServicesMeta holds the user-defined metadata for the services of a task.
// From task configuration, metadata is provided to a task's services in one
// of two ways:
// 1. map of metadata: a map of service-name to metadata so that each service
//    has its own unique metadata
// 2. metadata: a set of metadata is shared across all services that a task is
//    configured with
type ServicesMeta struct {
	// Deprecated in 0.5
	// When this happens: when task.services field and associated
	// services.cts_user_defined_meta field are configured
	metaMap map[string]map[string]string

	// Introduced in 0.5. Replaces metaMap
	// When this happens: when task.condition "services" or
	// task.module_input "services" cts_user_defined_meta field is configured
	meta map[string]string
}

// SetMetaMap sets the task's services' meta with a map of service-name to meta.
// This will error if meta (non-map form) is already set.
func (m *ServicesMeta) SetMetaMap(metaMap map[string]map[string]string) error {
	if m.meta != nil {
		return fmt.Errorf("cannot set meta-map. already set with meta %+v",
			m.meta)
	}

	m.metaMap = metaMap
	return nil
}

// SetMeta sets the task's services' meta with a map of service-name to meta.
// This will error if meta (map form) is already set.
func (m *ServicesMeta) SetMeta(meta map[string]string) error {
	if m.metaMap != nil {
		return fmt.Errorf("cannot set meta. already set with meta-map %+v", m.metaMap)
	}

	m.meta = meta
	return nil
}

// Get returns the metadata for a given service name. Returns an empty map if
// no meta is associated with the service.
func (m *ServicesMeta) Get(serviceName string) map[string]string {
	if m.metaMap != nil {
		return m.metaMap[serviceName]
	}

	if m.meta != nil {
		// all services share the same meta
		return m.meta
	}

	return make(map[string]string)
}
