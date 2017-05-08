package provider

import (
	"context"
	"fmt"
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
	defaultMethod = "wrr"
	defaultSticky = false
)

var (
	// Disable all timeouts for watcher requests
	watcherHTTPClient = &http.Client{
		Timeout:   0,
		Transport: &http.Transport{ResponseHeaderTimeout: 0},
	}

	sidecarHTTPClient = &http.Client{
		Timeout: 15 * time.Second,
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

type callback func(map[string][]*service.Service, error)

// Provide allows the provider to provide configurations to traefik
// using the given configuration channel.
func (provider *Sidecar) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool, _ types.Constraints) error {
	provider.configurationChan = configurationChan

	// Exit with a bang if we can't reach the Sidecar endpoint
	// TODO: On second thought, maybe it's best to continuously log errors in a loop
	// since no other provider uses log.Fatal if something goes wrong...
	_, err := watcherHTTPClient.Get(provider.Endpoint)
	if err != nil {
		log.Fatal("Failed to connect to the Sidecar endpoint: ", err)
	}

	if provider.Watch {
		safe.Go(func() {
			provider.runSidecarWatcher()
		})

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Error("Error creating file watcher: ", err)
			return err
		}

		file, err := os.Open(provider.Filename)
		if err != nil {
			log.Error("Error opening file: ", err)
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
						log.Debug("Sidecar config file event: ", event)
						err = provider.reloadConfig()
						if err != nil {
							log.Error(err)
						}
					}
				case errWatcher := <-watcher.Errors:
					log.Error("Watcher event error: ", errWatcher)
				}
			}
		})
		err = watcher.Add(filepath.Dir(file.Name()))
		if err != nil {
			log.Error("Error adding file watcher: ", err)
			return err
		}
	} else {
		err := provider.reloadConfig()
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (provider *Sidecar) reloadConfig() error {
	states, err := provider.fetchState()
	if err != nil {
		return fmt.Errorf("Error fetching Sidecar state: %s", err)
	}

	err = provider.loadConfig(states.ByService())
	if err != nil {
		return fmt.Errorf("Error loading Sidecar config: %s", err)
	}

	return nil
}

func (provider *Sidecar) loadConfig(sidecarStates map[string][]*service.Service) error {
	log.Info("Loading sidecar config...")

	config := &types.Configuration{
		Backends: make(map[string]*types.Backend),
	}

	if _, err := toml.DecodeFile(provider.Filename, config); err != nil {
		return err
	}

	// Create backends from Sidecar state data
	for serviceName, services := range sidecarStates {
		backend, ok := config.Backends[serviceName]
		if !ok {
			backend = &types.Backend{}
			config.Backends[serviceName] = backend
		}

		if backend.LoadBalancer == nil {
			backend.LoadBalancer = &types.LoadBalancer{Method: defaultMethod, Sticky: defaultSticky}
		}
		if backend.Servers == nil {
			backend.Servers = make(map[string]types.Server)
		}

		for _, serv := range services {
			if serv.IsAlive() {
				ipAddr, err := net.LookupIP(serv.Hostname)

				for i := 0; i < len(serv.Ports); i++ {
					// TODO: is there any point to add unreachable hosts?
					var hostname string
					if err != nil {
						log.Warn("Failed to resolve IP address: ", err)
						hostname = serv.Hostname
					} else {
						hostname = ipAddr[0].String()
					}

					backend.Servers[serv.Hostname] = types.Server{
						URL: fmt.Sprintf("http://%s:%d", hostname, serv.Ports[i].Port),
					}
				}
			}
		}
	}

	provider.configurationChan <- types.ConfigMessage{
		ProviderName:  "sidecar",
		Configuration: config,
	}

	log.Info("Finished loading sidecar config")

	return nil
}

func (provider *Sidecar) runSidecarWatcher() {
	// Set timeout to be just a bit more than connection refresh interval
	provider.connTimer = time.NewTimer(time.Duration(provider.RefreshConn))

	log.Debugf("Using %s Sidecar connection refresh interval", provider.RefreshConn)
	for {
		// Call a separate function because the defer statement has function scope
		provider.sidecarWatcher()
	}
}

func (provider *Sidecar) sidecarWatcher() {
	// Use refresh interval to occasionally reconnect to Sidecar in case the stream connection is lost
	req, err := http.NewRequest(http.MethodGet, provider.Endpoint+"/watch", nil)
	if err != nil {
		log.Errorf("Error creating http request to Sidecar instance '%s': %s", provider.Endpoint, err)
		time.Sleep(5 * time.Second)
		return
	}

	cx, cancel := context.WithCancel(context.Background())
	// Cancel the infinite timeout request automatically after we reset connTimer
	defer cancel()

	req = req.WithContext(cx)

	resp, err := watcherHTTPClient.Do(req)
	if err != nil {
		log.Errorf("Error connecting to Sidecar instance '%s': %s", provider.Endpoint, err)
		time.Sleep(5 * time.Second)
		return
	}
	defer resp.Body.Close()

	safe.Go(func() { catalog.DecodeStream(resp.Body, provider.callbackLoader) })

	// Wait on refresh connection timer. If this expires we haven't seen an update in a
	// while and should cancel the request, reset the time, and reconnect just in case
	<-provider.connTimer.C
	provider.connTimer.Reset(time.Duration(provider.RefreshConn))
}

func (provider *Sidecar) callbackLoader(sidecarStates map[string][]*service.Service, err error) {
	//load config regardless
	configErr := provider.loadConfig(sidecarStates)
	if configErr != nil {
		log.Error("Error loading sidecar config: ", err)
	}

	if err != nil {
		return
	}

	// Else reset connection timer
	if !provider.connTimer.Stop() {
		<-provider.connTimer.C
	}

	provider.connTimer.Reset(time.Duration(provider.RefreshConn))
}

func (provider *Sidecar) fetchState() (*catalog.ServicesState, error) {
	resp, err := watcherHTTPClient.Get(provider.Endpoint + "/state.json")
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
