package provider

import (
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"time"

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
	BaseProvider `mapstructure:",squash"`
	Endpoint     string        `description:"Sidecar URL"`
	Interval     time.Duration `description:"How often to poll Sidecar URL for backend changes and file for frontend changes"`
	Frontend     string        `description:"Configuration file for frontend"`
}

// Provide allows the provider to provide configurations to traefik
// using the given configuration channel.
func (provider *Sidecar) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool, constraints types.Constraints) error {
	if provider.Watch {
		provider.sidecarWatcher(configurationChan)
	}
	configuration, err := provider.loadSidecarConfig()
	if err != nil {
		return err
	}
	configurationChan <- types.ConfigMessage{
		ProviderName:  "sidecar",
		Configuration: configuration,
	}
	return nil
}

func (provider *Sidecar) loadSidecarConfig() (*types.Configuration, error) {
	states, err := provider.fetchState()
	if err != nil {
		log.Errorln("Error fetching state from Sidecar, ", err)
		return nil, err
	}
	sidecarConfig := types.Configuration{}
	sidecarStates := states.ByService()
	log.Infoln("loading sidecar config")
	sidecarConfig.Backends = provider.makeBackends(sidecarStates)
	log.Infoln("loading frontend config from file: ", provider.Frontend)
	sidecarConfig.Frontends, err = provider.makeFrontend()
	if err != nil {
		return nil, err
	}
	return &sidecarConfig, nil
}

func (provider *Sidecar) sidecarWatcher(configurationChan chan<- types.ConfigMessage) error {
	conf, err := provider.loadSidecarConfig()
	if err != nil {
		return err
	}
	for {
		time.Sleep(provider.Interval * 1000 * time.Millisecond)
		checkConf, err := provider.loadSidecarConfig()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(conf, checkConf) {
			log.Infoln("Sidecar update event")
			*conf = *checkConf
			configurationChan <- types.ConfigMessage{
				ProviderName:  "sidecar",
				Configuration: checkConf,
			}
		}
	}
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
	resp, err := client.Get(provider.Endpoint)
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
