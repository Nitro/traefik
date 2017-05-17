package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/containous/traefik/integration/utils"
	"github.com/go-check/check"

	checker "github.com/vdemeester/shakers"
)

// ConnStats test suites (using libcompose)
type ConnStatsSuite struct {
	BaseSuite
	dummyConnStats ConnStats
	whoamiReq      *http.Request
}

type ConnStats struct {
	File struct {
		Backends struct {
			Whoami struct {
				MaxConn   int `json:"max_conn"`
				TotalConn int `json:"total_conn"`
			} `json:"whoami"`
		} `json:"backends"`
	} `json:"file"`
}

func (s *ConnStatsSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "conn_stats")
	s.composeProject.Start(c)

	err := json.Unmarshal(
		[]byte(`{"file":{"backends":{"whoami":{"max_conn":2,"total_conn":0}}}}`),
		&s.dummyConnStats,
	)
	c.Assert(err, checker.IsNil)

	s.whoamiReq, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:8000/whoami", nil)
	c.Assert(err, checker.IsNil)
}

func (s *ConnStatsSuite) TearDownSuite(c *check.C) {
	if s.composeProject != nil {
		s.composeProject.Stop(c)
	}
}

func (s *ConnStatsSuite) ValidateWhoamiResponse(c *check.C, expectedResponseStatusCode int) {
	whoamiResp, err := http.DefaultClient.Do(s.whoamiReq)
	c.Assert(err, checker.IsNil)
	c.Assert(whoamiResp.StatusCode, checker.Equals, expectedResponseStatusCode)
}

func (s *ConnStatsSuite) ValidateConnStats(c *check.C, totalConn int) {
	resp, err := http.Get("http://127.0.0.1:8080/api/conn_stats")
	c.Assert(err, checker.IsNil)

	connStatsPayload, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, checker.IsNil)

	var connStats ConnStats
	err = json.Unmarshal(
		connStatsPayload,
		&connStats,
	)
	c.Assert(err, checker.IsNil)

	s.dummyConnStats.File.Backends.Whoami.TotalConn = totalConn
	c.Assert(connStats, checker.DeepEquals, s.dummyConnStats)
}

func (s *ConnStatsSuite) TestEndToEndWorkflow(c *check.C) {
	whoamiHost := s.composeProject.Container(c, "whoami").NetworkSettings.IPAddress

	file := s.adaptFile(c, "fixtures/conn_stats/simple.toml", struct {
		Server string
	}{whoamiHost})
	defer os.Remove(file)
	cmd := exec.Command(traefikBinary, "--configFile="+file)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	// Wait for traefik to start
	err = utils.TryRequest(
		"http://127.0.0.1:8080/api/providers",
		60*time.Second,
		func(res *http.Response) error {
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			if !strings.Contains(string(body), "Path:/whoami") {
				return errors.New("Incorrect traefik config: " + string(body))
			}
			return nil
		},
	)
	c.Assert(err, checker.IsNil)

	// Make sure we can reach the whoami backend server
	s.ValidateWhoamiResponse(c, http.StatusOK)
	s.ValidateConnStats(c, 0)

	// Start two long running requests to the backend to reach the connection limit
	whoamiWaitReq, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8000/whoami?wait=2s", nil)
	c.Assert(err, checker.IsNil)
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			resp, err := http.DefaultClient.Do(whoamiWaitReq)
			c.Assert(err, checker.IsNil)
			c.Assert(resp.StatusCode, checker.Equals, http.StatusOK)
			wg.Done()
		}()
	}

	// Wait a bit for the backend to become saturated
	time.Sleep(time.Second * 1)

	s.ValidateWhoamiResponse(c, http.StatusTooManyRequests)
	s.ValidateConnStats(c, 2)

	// Wait for the backend to become available again
	wg.Wait()

	s.ValidateWhoamiResponse(c, http.StatusOK)
	s.ValidateConnStats(c, 0)
}
