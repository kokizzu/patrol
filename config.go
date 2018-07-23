package patrol

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	SECRET_MAX_LENGTH = 128
)

var (
	ERR_CONFIG_NIL              = fmt.Errorf("Config was NIL")
	ERR_PATROL_EMPTY            = fmt.Errorf("Patrol Apps and Servers were both empty")
	ERR_APPS_KEY_EMPTY          = fmt.Errorf("App Key was empty")
	ERR_APPS_KEY_INVALID        = fmt.Errorf("App Key was invalid")
	ERR_APPS_APP_NIL            = fmt.Errorf("App was nil")
	ERR_SERVICES_KEY_EMPTY      = fmt.Errorf("Service Key was empty")
	ERR_SERVICES_KEY_INVALID    = fmt.Errorf("Service Key was invalid")
	ERR_SERVICES_SERVICE_NIL    = fmt.Errorf("Service was nil")
	ERR_APP_LABEL_DUPLICATE     = fmt.Errorf("Duplicate App Label")
	ERR_SERVICE_LABEL_DUPLICATE = fmt.Errorf("Duplicate Service Label")
	ERR_LISTEN_HTTP_EMPTY       = fmt.Errorf("HTTP Listeners were empty, we required one to exist!")
	ERR_LISTEN_UDP_EMPTY        = fmt.Errorf("UDP Listeners were empty, we required one to exist!")
	ERR_SECRET_TOOLONG          = fmt.Errorf("Secret Longer than %d bytes", SECRET_MAX_LENGTH)
)

func LoadConfig(
	path string,
) (
	*Config,
	error,
) {
	file, err := os.Open(path)
	if err != nil {
		// couldn't open file
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := &Config{}
	if err := decoder.Decode(config); err != nil {
		// couldn't decode file as json
		return nil, err
	}
	if config == nil {
		// decoded config was nil
		// while this would rarely occur in practice, it is technically possible for config to be nil after decode/unmarshal
		// a json file with the value of "null" would cause this to occur
		// the only time this really plays out in reality is if someone were to POST null to a JSON API
		// this something that is sometimes overlooked
		return nil, ERR_CONFIG_NIL
	}
	// we will NOT validate our config here!!!
	// we can validate manually or on create
	return config, nil
}

type Config struct {
	// Apps/Services must contain a unique non empty key: ( 0-9 A-Z a-z - )
	// ID MUST be usable as a valid hostname label, ie: len <= 63 AND no starting/ending -
	// Keys are NOT our binary name
	// Keys are only used as unique identifiers for our API and Keep Alive
	Apps     map[string]*ConfigApp     `json:"apps,omitempty"`
	Services map[string]*ConfigService `json:"services,omitempty"`
	// TickEvery is an integer value in seconds of how often we will check the state of our Apps and Services
	// Value of 0 Defaults to 15 seconds
	TickEvery int `json:"tick-every,omitempty"`
	// History is the maximum amount of instance history we should hold
	// Value of 0 Defaults to 100
	History int `json:"history,omitempty"`
	// Timestamp Layout is used by the JSON API and HTTP GUI templates
	//
	// Timestamp Layout can be found here:
	// https://golang.org/pkg/time/#pkg-constants
	// https://golang.org/pkg/time/#example_Time_Format
	//
	// The recommended value is RFC1123Z: "Mon, 02 Jan 2006 15:04:05 -0700"
	//
	// An empty value will default to time.String()
	// https://golang.org/pkg/time/#Time.String
	// This default is: "2006-01-02 15:04:05.999999999 -0700 MST"
	// This default will also include our monotonic clock as a suffix: "m=±<value>"
	Timestamp string `json:"json-timestamp,omitempty"`
	// PingTimeout is an integer value in seconds of how often we require a Ping to be sent
	// This only applies to App KeepAlives: APP_KEEPALIVE_HTTP and APP_KEEPALIVE_UDP
	PingTimeout int `json:"ping-timeout,omitempty"`
	// ListenHTTP/ListenUDP is our list of listeners
	// These values are passed as Environment Variables to our executed Apps as JSON Arrays
	//
	// Example Environment Variables:
	// PATROL_HTTP=["127.0.0.1:8421"]
	// PATROL_UDP=["127.0.0.1:1248"]
	//
	// When using APP_KEEPALIVE_HTTP and APP_KEEPALIVE_UDP, these are the addresses we MUST ping
	ListenHTTP []string `json:"listen-http,omitempty"`
	ListenUDP  []string `json:"listen-udp,omitempty"`
	// HTTP/UDP currently only support the attribute `listen`
	// This will allow us to overwrite our default listeners for HTTP and UDP
	// In the future this will include additional options.
	HTTP *ConfigHTTP `json:"http,omitempty"`
	UDP  *ConfigUDP  `json:"udp,omitempty"`
	// Triggers are only available when you extend Patrol as a library
	// These values will NOT be able to be set from `config.json` - They must be set manually
	//
	// TriggerStart is called on CreatePatrol
	// This will only be called ONCE
	// If an error is returned a Patrol object will NOT be returned!
	TriggerStart func(
		patrol *Patrol,
	) error `json:"-"`
	// TriggerShutdown is called when we call Patrol.Shutdown()
	// This will only be called ONCE
	// Once Patrol.Shutdown() is called our Patrol object will no longer be usable
	TriggerShutdown func(
		patrol *Patrol,
	) `json:"-"`
	// TriggerStarted is called every time we call Patrol.Start()
	TriggerStarted func(
		patrol *Patrol,
	) `json:"-"`
	// TriggerTick is called every time we Patrol.tick() and BEFORE we check our App and Service States
	TriggerTick func(
		patrol *Patrol,
	) `json:"-"`
	// TriggerStopped is called every time we call Patrol.Stop()
	TriggerStopped func(
		patrol *Patrol,
	) `json:"-"`
	// this is only used internally and checked once on creation
	unittesting bool
}

func (self *Config) IsValid() bool {
	if self == nil {
		return false
	}
	return true
}
func (self *Config) Clone() *Config {
	if self == nil {
		return nil
	}
	config := &Config{
		Apps:            make(map[string]*ConfigApp),
		Services:        make(map[string]*ConfigService),
		TickEvery:       self.TickEvery,
		History:         self.History,
		Timestamp:       self.Timestamp,
		PingTimeout:     self.PingTimeout,
		ListenHTTP:      make([]string, 0, len(self.ListenHTTP)),
		ListenUDP:       make([]string, 0, len(self.ListenUDP)),
		HTTP:            self.HTTP.Clone(),
		UDP:             self.UDP.Clone(),
		TriggerStart:    self.TriggerStart,
		TriggerShutdown: self.TriggerShutdown,
		TriggerStarted:  self.TriggerStarted,
		TriggerTick:     self.TriggerTick,
		TriggerStopped:  self.TriggerStopped,
	}
	for k, v := range self.Apps {
		config.Apps[k] = v.Clone()
	}
	for k, v := range self.Services {
		config.Services[k] = v.Clone()
	}
	for _, l := range self.ListenHTTP {
		config.ListenHTTP = append(config.ListenHTTP, l)
	}
	for _, l := range self.ListenUDP {
		config.ListenUDP = append(config.ListenUDP, l)
	}
	return config
}
func (self *Config) Validate() error {
	if len(self.Apps) == 0 &&
		len(self.Services) == 0 {
		// no apps or services found
		return ERR_PATROL_EMPTY
	}
	// we need to check for one exception, JSON keys are case sensitive
	// we won't allow any duplicate case insensitive keys to exist as our ID MAY be used as a hostname label
	// we're actually going to create a secondary dereferenced map with our keys set to lowercase
	// this way we can rely in the future on IDs being lowercase
	apps := make(map[string]*ConfigApp)
	// check apps
	http := false
	udp := false
	for id, app := range self.Apps {
		if id == "" {
			return ERR_APPS_KEY_EMPTY
		}
		if !IsAppServiceID(id) {
			return ERR_APPS_KEY_INVALID
		}
		// dereference
		app = app.Clone()
		if !app.IsValid() {
			return ERR_APPS_APP_NIL
		}
		if err := app.Validate(); err != nil {
			return err
		}
		// create lowercase ID
		id = strings.ToLower(id)
		if _, ok := apps[id]; ok {
			// ID already exists!!
			return ERR_APP_LABEL_DUPLICATE
		}
		apps[id] = app
		if app.KeepAlive == APP_KEEPALIVE_HTTP {
			http = true
		} else if app.KeepAlive == APP_KEEPALIVE_UDP {
			udp = true
		}
	}
	if http && len(self.ListenHTTP) == 0 {
		// no http servers
		return ERR_LISTEN_HTTP_EMPTY
	}
	if udp && len(self.ListenUDP) == 0 {
		// no udp servers
		return ERR_LISTEN_UDP_EMPTY
	}
	// overwrite apps
	self.Apps = apps
	// dereference and lowercase services
	services := make(map[string]*ConfigService)
	// check services
	for id, service := range self.Services {
		if id == "" {
			return ERR_SERVICES_KEY_EMPTY
		}
		if !IsAppServiceID(id) {
			return ERR_SERVICES_KEY_INVALID
		}
		// dereference
		service = service.Clone()
		if !service.IsValid() {
			return ERR_SERVICES_SERVICE_NIL
		}
		if err := service.Validate(); err != nil {
			return err
		}
		// create lowercase ID
		id = strings.ToLower(id)
		if _, ok := services[id]; ok {
			// ID already exists!!
			return ERR_SERVICE_LABEL_DUPLICATE
		}
		services[id] = service
	}
	// overwrite services
	self.Services = services
	// config
	if self.TickEvery == 0 {
		self.TickEvery = TICKEVERY_DEFAULT
	} else if self.TickEvery < TICKEVERY_MIN {
		self.TickEvery = TICKEVERY_MIN
	} else if self.TickEvery > TICKEVERY_MAX {
		self.TickEvery = TICKEVERY_MAX
	}
	if self.History == 0 {
		self.History = HISTORY_DEFAULT
	} else if self.History < HISTORY_MIN {
		self.History = HISTORY_MIN
	} else if self.History > HISTORY_MAX {
		self.History = HISTORY_MAX
	}
	if self.PingTimeout == 0 {
		self.PingTimeout = APP_PING_TIMEOUT_DEFAULT
	} else if self.PingTimeout < HISTORY_MIN {
		self.PingTimeout = APP_PING_TIMEOUT_MIN
	} else if self.PingTimeout > HISTORY_MAX {
		self.PingTimeout = APP_PING_TIMEOUT_MAX
	}
	return nil
}
