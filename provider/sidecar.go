package provider

import (
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	fsnotify "gopkg.in/fsnotify.v1"

	"github.com/BurntSushi/toml"
	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/service"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
)

const (
	method               = "drr"
	sticky               = false
	maxConnAmount        = 300
	maxConnExtractorFunc = "client.ip"
	circuitBreaker       = "NetworkErrorRatio() > 0.5"
	weight               = 0
)

var _ Provider = (*Sidecar)(nil)

// Sidecar holds configurations of the Sidecar provider
type Sidecar struct {
	BaseProvider      `mapstructure:",squash"`
	Endpoint          string        `description:"Sidecar URL"`
	Interval          time.Duration `description:"How often to poll Sidecar URL for backend changes and file for frontend changes"`
	Frontend          string        `description:"Configuration file for frontend"`
	configurationChan chan<- types.ConfigMessage
}

type callback func(map[string][]*service.Service, error)

// Provide allows the provider to provide configurations to traefik
// using the given configuration channel.
func (provider *Sidecar) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool, constraints types.Constraints) error {
	provider.configurationChan = configurationChan
	err := provider.loadSidecarConfig()
	if err != nil {
		return err
	}

	if provider.Watch {
		safe.Go(func() { provider.sidecarWatcher() })

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Errorln("Error creating file watcher", err)
			return err
		}

		file, err := os.Open(provider.Frontend)
		if err != nil {
			log.Errorln("Error opening file", err)
			return err
		}
		defer file.Close()
		pool.Go(func(stop chan bool) {
			defer watcher.Close()
			for {
				select {
				case <-stop:
					return
				case event := <-watcher.Events:
					if strings.Contains(event.Name, file.Name()) {
						log.Debug("Sidecar Frontend File event:", event)
						provider.loadSidecarConfig()
					}
				case error := <-watcher.Errors:
					log.Error("Watcher event error", error)
				}
			}
		})
		err = watcher.Add(filepath.Dir(file.Name()))
		if err != nil {
			log.Error("Error adding file watcher", err)
			return err
		}
	}
	return nil
}

func (provider *Sidecar) constructConfig(states *catalog.ServicesState) (*types.Configuration, error) {
	sidecarConfig := types.Configuration{}
	sidecarStates := states.ByService()
	log.Infoln("loading sidecar config")
	sidecarConfig.Backends = provider.makeBackends(sidecarStates)
	log.Infoln("loading frontend config from file: ", provider.Frontend)
	var err error
	sidecarConfig.Frontends, err = provider.makeFrontend()
	if err != nil {
		return nil, err
	}
	return &sidecarConfig, nil
}

func (provider *Sidecar) loadSidecarConfig() error {
	states, err := provider.fetchState()
	if err != nil {
		log.Errorln("Error fetching state from Sidecar, ", err)
		return err
	}
	conf, err := provider.constructConfig(states)
	if err != nil {
		return err
	}
	provider.configurationChan <- types.ConfigMessage{
		ProviderName:  "sidecar",
		Configuration: conf,
	}
	return nil
}

func (provider *Sidecar) sidecarWatcher() error {
	client := &http.Client{Timeout: 0}
	resp, err := client.Get(provider.Endpoint + "/watch")
	if err != nil {
		return err
	}
	err = catalog.DecodeStream(resp.Body, provider.callbackLoader)
	if err != nil {
		return err
	}
	for { //check for open http connection, try to re-init if it fails
		if resp.Close {
			resp, err = client.Get(provider.Endpoint + "/watch")
			if err != nil {
				return err
			}
			err = catalog.DecodeStream(resp.Body, provider.callbackLoader)
			if err != nil {
				return err
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func (provider *Sidecar) callbackLoader(sideBackend map[string][]*service.Service, err error) error {
	err = provider.loadSidecarConfig()
	if err != nil {
		return err
	}
	return nil
}

func (provider *Sidecar) makeFrontend() (map[string]*types.Frontend, error) {
	configuration := new(types.Configuration)
	if _, err := toml.DecodeFile(provider.Frontend, configuration); err != nil {
		log.Error("Error reading file:", err)
		return nil, err
	}
	return configuration.Frontends, nil
}

func (provider *Sidecar) makeBackends(sidecarStates map[string][]*service.Service) map[string]*types.Backend {
	sidecarBacks := make(map[string]*types.Backend)
	for serviceName, services := range sidecarStates {
		newServers := make(map[string]types.Server)
		newBackend := &types.Backend{LoadBalancer: &types.LoadBalancer{Method: method, Sticky: sticky},
			MaxConn:        &types.MaxConn{Amount: maxConnAmount, ExtractorFunc: maxConnExtractorFunc},
			CircuitBreaker: &types.CircuitBreaker{Expression: circuitBreaker},
			Servers:        newServers}
		for _, serv := range services {
			if serv.IsAlive() {
				for i := 0; i < len(serv.Ports); i++ {
					ipAddr, err := net.LookupIP(serv.Hostname)
					if err != nil {
						log.Errorln("Error resolving Ip address, ", err)
						newBackend.Servers[serv.Hostname] = types.Server{URL: "http://" + serv.Hostname + ":" + strconv.FormatInt(serv.Ports[i].Port, 10), Weight: weight}
					} else {
						newBackend.Servers[serv.Hostname] = types.Server{URL: "http://" + ipAddr[0].String() + ":" + strconv.FormatInt(serv.Ports[i].Port, 10), Weight: weight}
					}
				}
			}
		}
		sidecarBacks[serviceName] = newBackend
	}
	return sidecarBacks
}

func (provider *Sidecar) fetchState() (*catalog.ServicesState, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(provider.Endpoint + "/state.json")
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	state, err := catalog.Decode(bytes)
	if err != nil {
		return nil, err
	}
	return state, nil
}
