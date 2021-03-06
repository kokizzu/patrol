package patrol

type API_Status struct {
	// Instance ID - UUIDv4
	InstanceID string                   `json:"instance-id,omitempty"`
	Apps       map[string]*API_Response `json:"apps,omitempty"`
	Services   map[string]*API_Response `json:"service,omitempty"`
	// Timestamp Patrol started at
	Started *Timestamp `json:"started,omitempty"`
	// Is Patrol in a Shutdown state?
	Shutdown bool `json:"shutdown,omitempty"`
}

func (self *Patrol) GetStatus() *API_Status {
	self.mu.RLock()
	started := self.ticker_running
	shutdown := self.shutdown
	self.mu.RUnlock()
	result := &API_Status{
		InstanceID: self.instance_id,
		Apps:       make(map[string]*API_Response),
		Services:   make(map[string]*API_Response),
		Shutdown:   shutdown,
	}
	if !started.IsZero() {
		result.Started = &Timestamp{
			Time:            started,
			TimestampFormat: self.config.Timestamp,
		}
	}
	for id, app := range self.apps {
		app.o.RLock()
		result.Apps[id] = app.apiResponse(api_endpoint_status)
		app.o.RUnlock()
	}
	for id, service := range self.services {
		service.o.RLock()
		result.Services[id] = service.apiResponse(api_endpoint_status)
		service.o.RUnlock()
	}
	return result
}
