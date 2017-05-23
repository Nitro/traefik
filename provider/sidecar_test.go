package provider

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/service"
	"github.com/containous/flaeg"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSidecar(t *testing.T) {
	Convey("Sidecar should", t, func() {
		// httpmock only works with the default HTTP client
		origSidecarHTTPClient := sidecarHTTPClient
		origWatcherHTTPClient := watcherHTTPClient
		sidecarHTTPClient = http.DefaultClient
		watcherHTTPClient = http.DefaultClient

		dummyState := catalog.NewServicesState()
		dummyState.AddServiceEntry(
			service.Service{
				ID:       "007",
				Name:     "web",
				Hostname: "some-aws-host",
				Status:   0,
				Ports: []service.Port{
					{
						Type:        "tcp",
						Port:        8000,
						ServicePort: 8000,
						IP:          "127.0.0.1",
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

		Convey("fetch state", func() {
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

		Convey("create backends", func() {
			prov := Sidecar{
				Endpoint: "http://some.dummy.service",
			}

			Convey("and add servers for services with a single port exposed", func() {
				states, err := prov.fetchState()
				So(err, ShouldBeNil)

				backs := prov.makeBackends(states)

				So(backs, ShouldContainKey, "web")

				So(backs["web"].LoadBalancer.Method, ShouldEqual, "wrr")
				So(backs["web"].Servers["some-aws-host_8000"].URL, ShouldEqual, "http://127.0.0.1:8000")
			})

			Convey("and add servers for services with multiple ports exposed", func() {
				dummyState.AddServiceEntry(
					service.Service{
						ID:       "008",
						Name:     "api",
						Hostname: "another-aws-host",
						Status:   0,
						Ports: []service.Port{
							{
								Type:        "tcp",
								Port:        8000,
								ServicePort: 8000,
								IP:          "127.0.0.1",
							},
							{
								Type:        "udp",
								Port:        9000,
								ServicePort: 9000,
								IP:          "127.0.0.1",
							},
						},
					},
				)

				states, err := prov.fetchState()
				So(err, ShouldBeNil)

				backs := prov.makeBackends(states)

				So(backs, ShouldContainKey, "api")

				So(backs["api"].Servers["another-aws-host_8000"].URL, ShouldEqual, "http://127.0.0.1:8000")
				So(backs["api"].Servers["another-aws-host_9000"].URL, ShouldEqual, "http://127.0.0.1:9000")
			})

			Convey("and not add servers for which the IP cannot be obtained", func() {
				dummyState.AddServiceEntry(
					service.Service{
						ID:       "008",
						Name:     "api",
						Hostname: "another-aws-host",
						Status:   0,
						Ports: []service.Port{
							{
								Type:        "tcp",
								Port:        8000,
								ServicePort: 8000,
							},
						},
					},
				)

				states, err := prov.fetchState()
				So(err, ShouldBeNil)

				backs := prov.makeBackends(states)

				So(backs, ShouldContainKey, "api")

				// Don't add the server if the service.Port does not contain the IP address
				// and the hostname IP address can't be resolved
				So(backs["api"].Servers, ShouldNotContainKey, "another-aws-host_8000")
			})

			Convey("and not add servers for services that are not alive", func() {
				dummyState.AddServiceEntry(
					service.Service{
						ID:       "009",
						Name:     "sso",
						Hostname: "yet-another-aws-host",
						Status:   1,
						Ports: []service.Port{
							{
								Type:        "tcp",
								Port:        8000,
								ServicePort: 8000,
								IP:          "127.0.0.1",
							},
						},
					},
				)

				states, err := prov.fetchState()
				So(err, ShouldBeNil)

				backs := prov.makeBackends(states)

				So(backs, ShouldContainKey, "sso")
				So(backs["sso"].Servers, ShouldBeEmpty)
			})
		})

		Convey("construct config", func() {
			prov := Sidecar{
				BaseProvider: BaseProvider{
					Filename: "testdata/sidecar_testdata.toml",
				},
				Endpoint: "http://some.dummy.service",
			}

			dummyState.AddServiceEntry(
				service.Service{
					ID:       "008",
					Name:     "sso",
					Hostname: "yet-another-aws-host",
					Status:   0,
				},
			)

			states, err := prov.fetchState()
			So(err, ShouldBeNil)

			config, err := prov.constructConfig(states)
			So(err, ShouldBeNil)

			So(config.Frontends, ShouldContainKey, "web")
			So(config.Frontends, ShouldNotContainKey, "sso")

			So(config.Backends, ShouldContainKey, "web")
			So(config.Backends, ShouldContainKey, "sso")

			Convey("and set maxconn values", func() {
				So(config.Backends["web"].MaxConn.Amount, ShouldEqual, 10)
				So(config.Backends["web"].MaxConn.ExtractorFunc, ShouldEqual, "client.ip")

				// Don't add any connection limits if a backend is not assigned to any frontend
				So(config.Backends["sso"].MaxConn, ShouldBeNil)
			})

			Convey("and set default maxconn values", func() {
				dummyState.AddServiceEntry(
					service.Service{
						ID:       "009",
						Name:     "api",
						Hostname: "aws-host",
						Status:   0,
					},
				)

				dummyState.AddServiceEntry(
					service.Service{
						ID:       "010",
						Name:     "maxconn_amount_only",
						Hostname: "aws-host",
						Status:   0,
					},
				)

				dummyState.AddServiceEntry(
					service.Service{
						ID:       "011",
						Name:     "maxconn_extractorfunc_only",
						Hostname: "aws-host",
						Status:   0,
					},
				)

				states, err := prov.fetchState()
				So(err, ShouldBeNil)
				config, err := prov.constructConfig(states)
				So(err, ShouldBeNil)

				So(config.Frontends, ShouldContainKey, "api")
				So(config.Frontends, ShouldContainKey, "maxconn_amount_only")
				So(config.Frontends, ShouldContainKey, "maxconn_extractorfunc_only")
				So(config.Backends, ShouldContainKey, "api")
				So(config.Backends, ShouldContainKey, "maxconn_amount_only")
				So(config.Backends, ShouldContainKey, "maxconn_extractorfunc_only")

				So(config.Backends["api"].MaxConn.Amount, ShouldEqual, defaultMaxConnAmount)
				So(config.Backends["api"].MaxConn.ExtractorFunc, ShouldEqual, defaultMaxConnExtractorFunc)

				So(config.Backends["maxconn_amount_only"].MaxConn.Amount, ShouldEqual, 42)
				So(config.Backends["maxconn_amount_only"].MaxConn.ExtractorFunc, ShouldEqual, defaultMaxConnExtractorFunc)

				So(config.Backends["maxconn_extractorfunc_only"].MaxConn.Amount, ShouldEqual, defaultMaxConnAmount)
				So(config.Backends["maxconn_extractorfunc_only"].MaxConn.ExtractorFunc, ShouldEqual, "client.ip")
			})
		})

		Convey("run Provide", func() {
			prov := Sidecar{
				BaseProvider: BaseProvider{
					Watch:    false,
					Filename: "testdata/sidecar_testdata.toml",
				},
				Endpoint: "http://some.dummy.service",
			}

			configurationChan := make(chan types.ConfigMessage, 1)
			err := prov.Provide(configurationChan, nil, nil)
			configMsg := <-configurationChan

			So(err, ShouldBeNil)
			So(configMsg.ProviderName, ShouldEqual, "sidecar")
			So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
			So(configMsg.Configuration.Backends["web"].Servers["some-aws-host_8000"].URL, ShouldEqual, "http://127.0.0.1:8000")
		})

		Convey("run Provide() in watcher mode", func(c C) {
			// Return the watcher mock HTTP response on demand
			releaseWatch := make(chan bool)
			httpmock.RegisterResponder(http.MethodGet, "http://some.dummy.service/watch",
				func(req *http.Request) (*http.Response, error) {
					<-releaseWatch

					resp, err := httpmock.NewJsonResponse(200, dummyState)
					if err != nil {
						return httpmock.NewStringResponse(500, ""), nil
					}
					return resp, nil
				},
			)

			prov := Sidecar{
				BaseProvider: BaseProvider{
					Watch:    true,
					Filename: "testdata/sidecar_testdata.toml",
				},
				Endpoint:    "http://some.dummy.service",
				RefreshConn: flaeg.Duration(100 * time.Millisecond),
			}

			configMsgChan := make(chan types.ConfigMessage)
			pool := safe.NewPool(context.Background())
			defer pool.Cleanup()
			go func() {
				err := prov.Provide(
					configMsgChan,
					pool,
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
			So(configMsg.Configuration.Backends["web"].Servers["some-aws-host_8000"].URL, ShouldEqual, "http://127.0.0.1:8000")

			dummyState.AddServiceEntry(
				service.Service{
					ID:       "009",
					Name:     "api",
					Hostname: "another-aws-host",
					Status:   0,
					Ports: []service.Port{
						{
							Type:        "tcp",
							Port:        9000,
							ServicePort: 9000,
							IP:          "169.254.1.1",
						},
					},
				},
			)

			// Unblock the rest of the calls to recycleConn() to receive the updated config
			close(releaseWatch)
			configMsg = <-configMsgChan

			So(configMsg.Configuration.Backends, ShouldContainKey, "api")
			So(configMsg.Configuration.Backends["api"].Servers["another-aws-host_9000"].URL, ShouldEqual, "http://169.254.1.1:9000")

		})

		Reset(func() {
			httpmock.DeactivateAndReset()

			sidecarHTTPClient = origSidecarHTTPClient
			watcherHTTPClient = origWatcherHTTPClient
		})
	})
}

func TestSidecarMakeFrontend(t *testing.T) {
	Convey("Verify Sidecar Frontend config loading", t, func() {
		prov := Sidecar{
			BaseProvider: BaseProvider{
				Watch:    false,
				Filename: "testdata/sidecar_testdata.toml",
			},
			Endpoint: "http://some.dummy.service",
		}

		frontends, err := prov.makeFrontends()
		So(err, ShouldEqual, nil)
		So(frontends, ShouldContainKey, "web")
		So(frontends, ShouldContainKey, "api")
		So(frontends["web"].PassHostHeader, ShouldEqual, true)
		So(frontends["web"].EntryPoints, ShouldResemble, []string{"http", "https"})
		So(frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")

		prov.Filename = "testdata/dummyfile.toml"
		_, err = prov.makeFrontends()
		So(err, ShouldNotBeNil)
	})
}
