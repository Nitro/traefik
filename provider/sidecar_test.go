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
					Port:        8000,
					ServicePort: 8000,
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

func TestSidecarMakeBackends(t *testing.T) {
	setup()
	defer teardown()
	Convey("Verify creating backends", t, func() {
		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
			MaxConns: 10,
		}

		dummyState.AddServiceEntry(
			service.Service{
				ID:       "008",
				Name:     "api",
				Hostname: "another-aws-host",
				Status:   1,
			},
		)

		states, err := prov.fetchState()
		So(err, ShouldBeNil)

		sidecarStates := states.ByService()
		backs := prov.makeBackends(sidecarStates)

		So(backs["web"].LoadBalancer.Method, ShouldEqual, "wrr")
		So(backs["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:8000")
		So(backs["api"].Servers["another-aws-host"], ShouldBeZeroValue)

		So(backs["web"].MaxConn.Amount, ShouldEqual, 10)
		So(backs["web"].MaxConn.ExtractorFunc, ShouldEqual, "request.host")
		So(backs["api"].MaxConn.Amount, ShouldEqual, 10)
		So(backs["api"].MaxConn.ExtractorFunc, ShouldEqual, "request.host")

	})
}

func TestSidecarMakeFrontend(t *testing.T) {
	Convey("Verify Sidecar Frontend config loading", t, func() {
		prov := Sidecar{
			BaseProvider: BaseProvider{
				Watch: false,
			},
			Endpoint: "http://some.dummy.service",
			Frontend: "testdata/sidecar_testdata.toml",
		}

		conf, err := prov.makeFrontend()
		So(err, ShouldEqual, nil)
		So(conf["web"].PassHostHeader, ShouldEqual, true)
		So(conf["web"].EntryPoints, ShouldResemble, []string{"http", "https"})
		So(conf["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")

		prov.Frontend = "testdata/dummyfile.toml"
		_, err = prov.makeFrontend()
		So(err, ShouldNotBeNil)
	})
}

func TestSidecarProvider(t *testing.T) {
	setup()
	defer teardown()

	Convey("Verify Sidecar Provider", t, func() {
		prov := Sidecar{
			BaseProvider: BaseProvider{
				Watch: false,
			},
			Endpoint: "http://some.dummy.service",
			Frontend: "testdata/sidecar_testdata.toml",
		}

		configurationChan := make(chan types.ConfigMessage, 1)
		err := prov.Provide(configurationChan, nil, nil)
		configMsg, _ := <-configurationChan

		So(err, ShouldBeNil)
		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		So(configMsg.Configuration.Backends["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:8000")
	})
}

func TestSidecarWatcher(t *testing.T) {
	setup()
	defer teardown()

	Convey("Verify Sidecar Watcher", t, func(c C) {
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
				Watch: true,
			},
			Endpoint: "http://some.dummy.service",
			Frontend: "testdata/sidecar_testdata.toml",
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

		// Catch the loadSidecarConfig from the end of Provide()
		configMsg := <-configMsgChan

		// Unblock the first call to recycleConn()
		releaseWatch <- true
		configMsg = <-configMsgChan

		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		So(configMsg.Configuration.Backends["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:8000")

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

		// Unblock the second call to recycleConn() to receive the updated config
		releaseWatch <- true
		configMsg = <-configMsgChan

		So(configMsg.Configuration.Backends, ShouldContainKey, "api")
		So(configMsg.Configuration.Backends["api"].Servers["another-aws-host"].URL, ShouldEqual, "http://another-aws-host:9000")
	})
}
