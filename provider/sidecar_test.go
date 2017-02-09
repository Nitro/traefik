package provider

import (
	"context"
	"net/http"
	"reflect"
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
		var baseTime = time.Now().UTC().Round(time.Second)
		var testPort = service.Port{Type: "tcp", Port: 8000, ServicePort: 8000}

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service/state.json",
			func(req *http.Request) (*http.Response, error) {

				service := service.Service{ID: "007", Name: "api", Hostname: "some-aws-host",
					Updated: baseTime, Status: 1, Ports: []service.Port{testPort}}
				returnState := catalog.NewServicesState()
				returnState.AddServiceEntry(service)
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
		testServices := testState.ByService()

		compareState := catalog.NewServicesState()
		service := &service.Service{ID: "007", Name: "api", Hostname: "some-aws-host",
			Updated: baseTime, Status: 1, Ports: []service.Port{testPort}}
		compareState.AddServiceEntry(*service)
		compareServices := compareState.ByService()

		So(err, ShouldBeNil)
		So(reflect.DeepEqual(testServices["api"][0].Ports, compareServices["api"][0].Ports), ShouldBeTrue)
		So(testServices["api"][0].Hostname, ShouldEqual, compareServices["api"][0].Hostname)

		compareServices["api"][0].Hostname = "wrong-host"
		So(testServices["api"][0].Hostname, ShouldNotEqual, compareServices["api"][0].Hostname)

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
		}

		httpmock.RegisterResponder("GET", "http://some.dummy.service/state.json",
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
		}
		prov.Watch = true
		prov.Frontend = "testdata/sidecar_testdata.toml"
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

func Test_SidecarProvider(t *testing.T) {
	Convey("Verify Sidecar Provider", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service/state.json",
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

func Test_SidecarWatcher(t *testing.T) {
	Convey("Verify Sidecar Provider", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", "http://some.dummy.service/state.json",
			func(req *http.Request) (*http.Response, error) {

				testPort := service.Port{Type: "tcp", Port: 9000, ServicePort: 9000}
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

		httpmock.RegisterResponder("GET", "http://some.dummy.service/watch",
			func(req *http.Request) (*http.Response, error) {

				testPort := service.Port{Type: "tcp", Port: 9000, ServicePort: 9000}
				returnState := catalog.NewServicesState()
				baseTime := time.Now().UTC().Round(time.Second)
				serv := service.Service{ID: "007", Name: "web", Hostname: "some-aws-host",
					Updated: baseTime.Add(5 * time.Second), Status: 0, Ports: []service.Port{testPort}}
				returnState.AddServiceEntry(serv)
				resp, err := httpmock.NewJsonResponse(200, returnState.ByService())
				if err != nil {
					return httpmock.NewStringResponse(500, ""), nil
				}
				return resp, nil
			},
		)
		prov := Sidecar{
			Endpoint: "http://some.dummy.service",
		}
		prov.Watch = true
		prov.Frontend = "testdata/sidecar_testdata.toml"
		configurationChan := make(chan types.ConfigMessage, 100)
		constraints := types.Constraints{}
		pool := safe.NewPool(context.Background())
		go prov.Provide(configurationChan, pool, constraints)
		configMsg, _ := <-configurationChan
		So(configMsg.ProviderName, ShouldEqual, "sidecar")
		So(configMsg.Configuration.Frontends["web"].Routes["test_1"].Rule, ShouldEqual, "Host: some-aws-host")
		So(configMsg.Configuration.Backends["web"].Servers["some-aws-host"].URL, ShouldEndWith, "http://some-aws-host:9000")
	})
}
