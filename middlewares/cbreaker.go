package middlewares

import (
	"net/http"
	"sync"

	"github.com/vulcand/oxy/cbreaker"
	"github.com/vulcand/oxy/memmetrics"
)

// CircuitBreaker holds the oxy circuit breaker.
type CircuitBreaker struct {
	circuitBreaker *cbreaker.CircuitBreaker
	mutex          *sync.RWMutex
	conns          map[string]int64
}

type CBStats struct {
	CBState string
	Metrics *memmetrics.RTMetrics
	Conns   map[string]int64
}

// NewCircuitBreaker returns a new CircuitBreaker.
func NewCircuitBreaker(next http.Handler, expression string, options ...cbreaker.CircuitBreakerOption) (*CircuitBreaker, error) {
	circuitBreaker, err := cbreaker.New(next, expression, options...)
	if err != nil {
		return nil, err
	}
	return &CircuitBreaker{circuitBreaker, &sync.RWMutex{}, make(map[string]int64)}, nil
}

func (cb *CircuitBreaker) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	cb.acquire(r.Host)
	defer cb.release(r.Host)

	cb.circuitBreaker.ServeHTTP(rw, r)
}

func (cb *CircuitBreaker) acquire(url string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.conns[url] += int64(1)
}

func (cb *CircuitBreaker) release(url string) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.conns[url] -= int64(1)
}

func (cb *CircuitBreaker) GetStats() *CBStats {
	connsMap := make(map[string]int64)
	currState := &CBStats{cb.circuitBreaker.String(), cb.circuitBreaker.Metrics, connsMap}

	//don't bother with a mutex for copying the map.  We don't want to block traffic for this
	for k, v := range cb.conns {
		currState.Conns[k] = v
	}
	return currState
}
