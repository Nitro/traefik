package provider

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/service"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	origSidecarHTTPClient = sidecarHTTPClient
	origWatcherHTTPClient = watcherHTTPClient
	dummyState            *catalog.ServicesState
)

func setup() {
	sidecarHTTPClient = http.DefaultClient
	watcherHTTPClient = http.DefaultClient

	dummyState = catalog.NewServicesState()
	dummyState.AddServiceEntry(
		service.Service{
			ID:       "007",
			Name:     "web",
			Hostname: "some-aws-host",
			Status:   0,
			Ports: []service.Port{
				service.Port{
					Type:        "tcp",
					Port:        9000,
					ServicePort: 9000,
				},
			},
		},
	)

	httpmock.Activate()

	httpmock.RegisterResponder(http.MethodGet, "http://some.dummy.service",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, ""), nil
		},
	)

	httpmock.RegisterResponder(http.MethodGet, "http://some.dummy.service/state.json",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, dummyState)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)
}

func teardown() {
	httpmock.DeactivateAndReset()

	sidecarHTTPClient = origSidecarHTTPClient
	watcherHTTPClient = origWatcherHTTPClient
}

func TestSidecarFetchState(t *testing.T) {
	setup()
	defer teardown()

	Convey("Verify fetching state", t, func() {
		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
		}

		receivedState, err := prov.fetchState()
		So(err, ShouldBeNil)

		receivedServices := receivedState.ByService()

		expectedServices := dummyState.ByService()

		So(reflect.DeepEqual(receivedServices["web"][0].Ports, expectedServices["web"][0].Ports), ShouldBeTrue)
		So(receivedServices["web"][0].Hostname, ShouldEqual, expectedServices["web"][0].Hostname)

		expectedServices["web"][0].Hostname = "wrong-host"
		So(receivedServices["web"][0].Hostname, ShouldNotEqual, expectedServices["web"][0].Hostname)

		prov.Endpoint = "http://yetanother.dummy.service"
		_, err = prov.fetchState()
		So(err, ShouldNotBeNil)
	})
}

func TestSidecarLoadConfig(t *testing.T) {
	setup()
	defer teardown()

	Convey("Verify load config", t, func() {
		configMsgChan := make(chan types.ConfigMessage)
		prov := Sidecar{
			BaseProvider: BaseProvider{
				Filename: "testdata/sidecar_config.toml",
			},
			Endpoint:          "http://some.dummy.service",
			configurationChan: configMsgChan,
		}

		dummyState.AddServiceEntry(
			service.Service{
				ID:       "008",
				Name:     "raster",
				Hostname: "another-aws-host",
				Status:   1,
			},
		)

		states, err := prov.fetchState()
		sidecarStates := states.ByService()

		configLoaded := make(chan bool)
		go func() {
			err = prov.loadConfig(sidecarStates)
			configLoaded <- true
		}()
		configMsg := <-configMsgChan
		<-configLoaded

		So(err, ShouldBeNil)

		So(configMsg.ProviderName, ShouldEqual, "sidecar")

		frontends := configMsg.Configuration.Frontends
		backends := configMsg.Configuration.Backends

		So(frontends["web"].PassHostHeader, ShouldEqual, true)
		So(frontends["web"].EntryPoints, ShouldResemble, []string{"http", "https"})
		So(frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")

		So(backends, ShouldContainKey, "web")
		So(backends, ShouldContainKey, "raster")
		So(backends["web"].LoadBalancer.Method, ShouldEqual, "wrr")
		So(backends["web"].LoadBalancer.Sticky, ShouldEqual, false)
		So(backends["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:9000")
		So(backends["raster"].Servers["another-aws-host"], ShouldBeZeroValue)

		So(backends["web"].MaxConn.Amount, ShouldEqual, 10)
		So(backends["web"].MaxConn.ExtractorFunc, ShouldEqual, "request.host")

		prov.Filename = "testdata/dummyfile.toml"
		err = prov.loadConfig(sidecarStates)
		So(err, ShouldNotBeNil)
	})
}

func TestSidecarProvider(t *testing.T) {
	setup()
	defer teardown()

	Convey("Verify provider", t, func(c C) {
		prov := Sidecar{
			BaseProvider: BaseProvider{
				Watch:    false,
				Filename: "testdata/sidecar_config.toml",
			},
			Endpoint: "http://some.dummy.service",
		}

		configMsgChan := make(chan types.ConfigMessage)
		provideFinished := make(chan bool)
		go func() {
			err := prov.Provide(configMsgChan, nil, nil)

			c.So(err, ShouldBeNil)

			provideFinished <- true
		}()
		configMsg := <-configMsgChan
		<-provideFinished

		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		So(configMsg.Configuration.Backends["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:9000")

		So(configMsg.Configuration.Backends["web"].MaxConn.Amount, ShouldEqual, 10)
		So(configMsg.Configuration.Backends["web"].MaxConn.ExtractorFunc, ShouldEqual, "request.host")
	})
}

func TestSidecarWatcher(t *testing.T) {
	setup()
	defer teardown()

	Convey("Verify watcher", t, func(c C) {
		// Return the watcher mock HTTP response on demand
		releaseWatch := make(chan bool)
		httpmock.RegisterResponder(http.MethodGet, "http://some.dummy.service/watch",
			func(req *http.Request) (*http.Response, error) {
				<-releaseWatch

				resp, err := httpmock.NewJsonResponse(200, dummyState.ByService())
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)

		prov := Sidecar{
			BaseProvider: BaseProvider{
				Watch:    true,
				Filename: "testdata/sidecar_config.toml",
			},
			Endpoint: "http://some.dummy.service",
		}

		// We have no way to shut down the provider when it's running in watch mode,
		// so just let the unit test close it at the end
		configMsgChan := make(chan types.ConfigMessage)
		go func() {
			err := prov.Provide(
				configMsgChan,
				safe.NewPool(context.Background()),
				nil,
			)

			c.So(err, ShouldBeNil)
		}()

		releaseWatch <- true
		configMsg := <-configMsgChan

		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		So(configMsg.Configuration.Backends["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:9000")
		So(configMsg.Configuration.Backends["web"].MaxConn.Amount, ShouldEqual, 10)
		So(configMsg.Configuration.Backends["web"].MaxConn.ExtractorFunc, ShouldEqual, "request.host")

		dummyState.AddServiceEntry(
			service.Service{
				ID:       "009",
				Name:     "api",
				Hostname: "another-aws-host",
				Status:   0,
				Ports: []service.Port{
					service.Port{
						Type:        "tcp",
						Port:        9000,
						ServicePort: 9000,
					},
				},
			},
		)

		releaseWatch <- true
		configMsg = <-configMsgChan

		So(configMsg.Configuration.Backends["api"].Servers["another-aws-host"].URL, ShouldEqual, "http://another-aws-host:9000")
	})
}
