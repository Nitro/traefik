package provider

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Nitro/sidecar/catalog"
	"github.com/Nitro/sidecar/service"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

func Test_FetchState(t *testing.T) {
	Convey("Verify Fetching State handler", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service",
			func(req *http.Request) (*http.Response, error) {

				returnState := catalog.NewServicesState()
				resp, err := httpmock.NewJsonResponse(200, returnState)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)

		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
		}

		testState, err := prov.fetchState()
		compareState := catalog.NewServicesState()

		So(err, ShouldBeNil)
		So(testState, ShouldHaveSameTypeAs, compareState)

		prov.Endpoint = "http://yetanother.dummy.service"
		_, err = prov.fetchState()

		So(err, ShouldNotBeNil)
	})
}

func Test_FetchBackend(t *testing.T) {
	Convey("Verify Fetching Backend", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
			Interval: 5,
		}

		httpmock.RegisterResponder("GET", "http://some.dummy.service",
			func(req *http.Request) (*http.Response, error) {

				testPort := service.Port{Type: "tcp", Port: 8000, ServicePort: 8000}
				returnState := catalog.NewServicesState()
				baseTime := time.Now().UTC().Round(time.Second)
				serviceA := service.Service{ID: "007", Name: "web", Hostname: "some-aws-host",
					Updated: baseTime.Add(5 * time.Second), Status: 0, Ports: []service.Port{testPort}}
				serviceB := service.Service{ID: "008", Name: "api", Hostname: "another-aws-host",
					Updated: baseTime, Status: 1}
				returnState.AddServiceEntry(serviceA)
				returnState.AddServiceEntry(serviceB)
				resp, err := httpmock.NewJsonResponse(200, returnState)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		states, err := prov.fetchState()
		sidecarStates := states.ByService()
		backs := prov.makeBackends(sidecarStates)

		So(err, ShouldBeNil)
		So(backs["web"].LoadBalancer.Method, ShouldEqual, "drr")
		So(backs["web"].MaxConn.Amount, ShouldEqual, 300)
		So(backs["web"].Servers["some-aws-host"].URL, ShouldEqual, "http://some-aws-host:8000")
		So(backs["api"].MaxConn.ExtractorFunc, ShouldEqual, "client.ip")
		So(backs["api"].Servers["another-aws-host"], ShouldBeZeroValue)
	})
}

func Test_MakeFrontEnd(t *testing.T) {
	Convey("Verify Sidecar Frontend Config Loader", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service",
			func(req *http.Request) (*http.Response, error) {

				returnState := catalog.NewServicesState()
				resp, err := httpmock.NewJsonResponse(200, returnState)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
			Interval: 1,
		}
		prov.Watch = true
		prov.Frontend = "testdata/sidecar_testdata.toml"
		conf, err := prov.loadSidecarConfig()
		So(err, ShouldEqual, nil)
		So(conf.Frontends["web"].PassHostHeader, ShouldEqual, true)
		So(conf.Frontends["web"].EntryPoints, ShouldResemble, []string{"http", "https"})
		So(conf.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		prov.Frontend = "testdata/dummyfile.toml"
		_, err = prov.loadSidecarConfig()
		So(err, ShouldNotBeNil)
	})
}

func Test_SidecarProvider(t *testing.T) {
	Convey("Verify Sidecar Provider", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service",
			func(req *http.Request) (*http.Response, error) {

				testPort := service.Port{Type: "tcp", Port: 8000, ServicePort: 8000}
				returnState := catalog.NewServicesState()
				baseTime := time.Now().UTC().Round(time.Second)
				serv := service.Service{ID: "007", Name: "web", Hostname: "some-aws-host",
					Updated: baseTime.Add(5 * time.Second), Status: 0, Ports: []service.Port{testPort}}
				returnState.AddServiceEntry(serv)
				resp, err := httpmock.NewJsonResponse(200, returnState)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
		}
		prov.Watch = false
		prov.Interval = 3
		prov.Frontend = "testdata/sidecar_testdata.toml"

		configurationChan := make(chan types.ConfigMessage, 100)
		constraints := types.Constraints{}
		pool := safe.NewPool(context.Background())
		err := prov.Provide(configurationChan, pool, constraints)
		configMsg, _ := <-configurationChan
		So(err, ShouldBeNil)
		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		So(configMsg.Configuration.Backends["web"].Servers["some-aws-host"].URL, ShouldEndWith, "http://some-aws-host:8000")
	})
}

func Test_Watcher(t *testing.T) {
	Convey("Verify Sidecar Watcher", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service",
			func(req *http.Request) (*http.Response, error) {

				testPort := service.Port{Type: "tcp", Port: 8000, ServicePort: 8000}
				returnState := catalog.NewServicesState()
				baseTime := time.Now().UTC().Round(time.Second)
				serv := service.Service{ID: "007", Name: "web", Hostname: "some-aws-host",
					Updated: baseTime.Add(5 * time.Second), Status: 0, Ports: []service.Port{testPort}}
				returnState.AddServiceEntry(serv)
				resp, err := httpmock.NewJsonResponse(200, returnState)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)

		httpmock.RegisterResponder("GET", "http://another.dummy.service",
			func(req *http.Request) (*http.Response, error) {

				testPort := service.Port{Type: "tcp", Port: 9000, ServicePort: 9000}
				returnState := catalog.NewServicesState()
				baseTime := time.Now().UTC().Round(time.Second)
				serv := service.Service{ID: "007", Name: "web", Hostname: "another-aws-host",
					Updated: baseTime.Add(5 * time.Second), Status: 0, Ports: []service.Port{testPort}}
				returnState.AddServiceEntry(serv)
				resp, err := httpmock.NewJsonResponse(200, returnState)
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)

		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
			Frontend: "testdata/sidecar_testdata.toml",
			Interval: 1,
		}

		configurationChan := make(chan types.ConfigMessage, 1)
		go prov.sidecarWatcher(configurationChan)
		time.Sleep(3 * 1000 * time.Millisecond)
		prov.Endpoint = "http://another.dummy.service"
		configMsg, _ := <-configurationChan

		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Backends["web"].Servers["another-aws-host"].URL, ShouldEqual, "http://another-aws-host:9000")
	})
}
