package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/service"
	"github.com/containous/flaeg"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	fsnotify "gopkg.in/fsnotify.v1"
)

const (
	method                      = "wrr"
	weight                      = 0
	sticky                      = false
	defaultMaxConnAmount        = 2000
	defaultMaxConnExtractorFunc = "request.host"
)

var (
	// Disable all timeouts for watcher requests
	watcherHTTPTransport = &http.Transport{ResponseHeaderTimeout: 0}
	watcherHTTPClient    = &http.Client{
		Timeout:   0,
		Transport: watcherHTTPTransport,
	}

	sidecarHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
)

var _ Provider = (*Sidecar)(nil)

// Sidecar holds configurations of the Sidecar provider
type Sidecar struct {
	BaseProvider      `mapstructure:",squash"`
	Endpoint          string `description:"Sidecar URL"`
	configurationChan chan<- types.ConfigMessage
	RefreshConn       flaeg.Duration `description:"How often to refresh the connection to Sidecar backend"`
	connTimer         *time.Timer
}

type sidecarFrontend struct {
	*types.Frontend
	MaxConn *types.MaxConn `json:"maxConn,omitempty"`
}

type sidecarConfig struct {
	Frontends map[string]*sidecarFrontend `json:"frontends,omitempty"`
}

type callback func(map[string][]*service.Service, error)

// Provide allows the provider to provide configurations to traefik
// using the given configuration channel.
func (provider *Sidecar) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool, constraints types.Constraints) error {
	provider.configurationChan = configurationChan

	if provider.Watch {
		pool.Go(
			func(stop chan bool) {
				provider.sidecarWatcher(stop, pool)
			},
		)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Errorln("Error creating file watcher", err)
			return err
		}

		file, err := os.Open(provider.Filename)
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
						states, errState := provider.fetchState()
						if errState != nil {
							log.Errorln("Error reloading Sidecar config", errState)
						}
						errLoadConfig := provider.loadSidecarConfig(states)
						if errLoadConfig != nil {
							log.Errorf("Error loading Sidecar config: %s", errLoadConfig)
						}
					}
				case errWatcher := <-watcher.Errors:
					log.Errorln("Watcher event error", errWatcher)
				}
			}
		})
		err = watcher.Add(filepath.Dir(file.Name()))
		if err != nil {
			log.Error("Error adding file watcher", err)
			return err
		}
	}
	states, err := provider.fetchState()
	if err != nil {
		log.Fatalln("Error reloading Sidecar config", err)
	}

	return provider.loadSidecarConfig(states)
}

func (provider *Sidecar) constructConfig(sidecarStates *catalog.ServicesState) (*types.Configuration, error) {
	log.Infoln("loading sidecar config")
	sidecarConfig := types.Configuration{
		Backends:  provider.makeBackends(sidecarStates),
		Frontends: map[string]*types.Frontend{},
	}

	frontends, err := provider.makeFrontends()
	if err != nil {
		return nil, err
	}

	for name, frontend := range frontends {
		sidecarConfig.Frontends[name] = frontend.Frontend

		if backend, ok := sidecarConfig.Backends[name]; ok {
			backend.MaxConn = frontend.MaxConn
			if backend.MaxConn == nil {
				backend.MaxConn = &types.MaxConn{
					Amount:        defaultMaxConnAmount,
					ExtractorFunc: defaultMaxConnExtractorFunc,
				}
			} else {
				if backend.MaxConn.Amount == 0 {
					backend.MaxConn.Amount = defaultMaxConnAmount
				}

				if backend.MaxConn.ExtractorFunc == "" {
					backend.MaxConn.ExtractorFunc = defaultMaxConnExtractorFunc
				}
			}

		}
	}

	return &sidecarConfig, nil
}

func (provider *Sidecar) loadSidecarConfig(sidecarStates *catalog.ServicesState) error {
	conf, err := provider.constructConfig(sidecarStates)
	if err != nil {
		return err
	}

	provider.configurationChan <- types.ConfigMessage{
		ProviderName:  "sidecar",
		Configuration: conf,
	}
	return nil
}

func (provider *Sidecar) sidecarWatcher(stop chan bool, pool *safe.Pool) {
	//set timeout to be just a bit more than connection refresh interval
	provider.connTimer = time.NewTimer(time.Duration(provider.RefreshConn))

	log.Debugf("Using %s Sidecar connection refresh interval", provider.RefreshConn)
	provider.recycleConn(stop, pool)
}

func (provider *Sidecar) recycleConn(stop chan bool, pool *safe.Pool) {
	var err error
	var resp *http.Response
	var req *http.Request
	for {
		select {
		case <-stop:
			return
		default:
			//use refresh interval to occasionally reconnect to Sidecar in case the stream connection is lost
			req, err = http.NewRequest("GET", provider.Endpoint+"/watch?by_service=false", nil)
			if err != nil {
				log.Errorf("Error creating http request to Sidecar: %s, Error: %s", provider.Endpoint, err)
				continue
			}
			resp, err = watcherHTTPClient.Do(req)
			if err != nil {
				log.Errorf("Error connecting to Sidecar: %s, Error: %s", provider.Endpoint, err)
				time.Sleep(5 * time.Second)
				continue
			}

			safe.Go(func() { decodeStream(resp.Body, provider.callbackLoader) })

			//wait on refresh connection timer.  If this expires we haven't seen an update in a
			//while and should cancel the request, reset the time, and reconnect just in case
			<-provider.connTimer.C
			provider.connTimer.Reset(time.Duration(provider.RefreshConn))

			//TODO: Deprecated method. Refactor this to use a context.
			watcherHTTPTransport.CancelRequest(req)
		}
	}
}

func (provider *Sidecar) callbackLoader(sidecarStates *catalog.ServicesState, err error) {
	//load config regardless
	errLoadConfig := provider.loadSidecarConfig(sidecarStates)
	if errLoadConfig != nil {
		log.Error("Error reloading config: %s", errLoadConfig)
	}

	if err != nil {
		return
	}

	//else reset connection timer
	if !provider.connTimer.Stop() {
		<-provider.connTimer.C
	}

	provider.connTimer.Reset(time.Duration(provider.RefreshConn))
}

func (provider *Sidecar) makeFrontends() (map[string]*sidecarFrontend, error) {
	configuration := new(sidecarConfig)
	if _, err := toml.DecodeFile(provider.Filename, configuration); err != nil {
		log.Errorf("Error reading file: %s", err)
		return nil, err
	}

	return configuration.Frontends, nil
}

func (provider *Sidecar) makeBackends(sidecarStates *catalog.ServicesState) map[string]*types.Backend {
	sidecarBacks := make(map[string]*types.Backend)

	sidecarStates.EachService(
		func(hostname *string, serviceId *string, svc *service.Service) {
			var backend *types.Backend
			var ok bool
			if backend, ok = sidecarBacks[svc.Name]; !ok {
				backend = &types.Backend{
					LoadBalancer: &types.LoadBalancer{Method: method, Sticky: sticky},
					Servers:      make(map[string]types.Server),
				}

				sidecarBacks[svc.Name] = backend
			}

			if svc.IsAlive() {
				for _, port := range svc.Ports {
					name := svc.Hostname
					if len(svc.Ports) > 1 {
						name = fmt.Sprintf("%s_%d", svc.Hostname, port.Port)
					}

					host := port.IP
					if host == "" {
						ipAddr, err := net.LookupIP(svc.Hostname)
						if err != nil {
							log.Errorf("Error resolving IP address for host '%s': %s", svc.Hostname, err)
							host = svc.Hostname

						} else {
							host = ipAddr[0].String()
						}
					}

					backend.Servers[name] = types.Server{
						URL: fmt.Sprintf("http://%s:%d", host, port.Port),
					}
				}
			}
		},
	)

	return sidecarBacks
}

func (provider *Sidecar) fetchState() (*catalog.ServicesState, error) {
	resp, err := sidecarHTTPClient.Get(provider.Endpoint + "/state.json")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

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

func decodeStream(input io.Reader, callback func(*catalog.ServicesState, error)) error {
	dec := json.NewDecoder(input)
	for dec.More() {
		var conf catalog.ServicesState
		err := dec.Decode(&conf)

		callback(&conf, err)

		if err != nil {
			log.Errorf("Error decoding stream: %s", err)
			return err
		}
	}
	return nil
}
