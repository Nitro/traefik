package provider

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	method               = "wrr"
	weight               = 0
	sticky               = false
	maxConnExtractorFunc = "request.host"
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
	Frontend          string `description:"Configuration file for frontend"`
	configurationChan chan<- types.ConfigMessage
	RefreshConn       flaeg.Duration `description:"How often to refresh the connection to Sidecar backend"`
	connTimer         *time.Timer
	MaxConns          int64 `description:"Maximum number of connections allowed for each backend"`
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
	sidecarConfig := types.Configuration{Backends: provider.makeBackends(sidecarStates)}
	var err error
	sidecarConfig.Frontends, err = provider.makeFrontends()
	if err != nil {
		return nil, err
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

			pool.Go(
				func(stop chan bool) {
					decodeStream(resp.Body, provider.callbackLoader, stop)
				},
			)

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

func (provider *Sidecar) makeFrontends() (map[string]*types.Frontend, error) {
	configuration := new(types.Configuration)
	if _, err := toml.DecodeFile(provider.Frontend, configuration); err != nil {
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
					MaxConn: &types.MaxConn{
						Amount:        provider.MaxConns,
						ExtractorFunc: maxConnExtractorFunc,
					},
				}

				sidecarBacks[svc.Name] = backend
			}

			if svc.IsAlive() {
				for i := 0; i < len(svc.Ports); i++ {
					ipAddr, err := net.LookupIP(svc.Hostname)
					if err != nil {
						log.Errorln("Error resolving Ip address, ", err)
						backend.Servers[svc.Hostname] = types.Server{
							URL: "http://" + svc.Hostname + ":" + strconv.FormatInt(svc.Ports[i].Port, 10),
						}
					} else {
						backend.Servers[svc.Hostname] = types.Server{
							URL: "http://" + ipAddr[0].String() + ":" + strconv.FormatInt(svc.Ports[i].Port, 10),
						}
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

func decodeStream(input io.Reader, callback func(*catalog.ServicesState, error), stop chan bool) error {
	dec := json.NewDecoder(input)
	for dec.More() {
		select {
		case <-stop:
			return nil
		default:
			var conf catalog.ServicesState
			err := dec.Decode(&conf)

			callback(&conf, err)

			if err != nil {
				log.Errorf("Error decoding stream: %s", err)
				return err
			}
		}
	}
	return nil
}
