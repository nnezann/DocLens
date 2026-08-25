package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	total   atomic.Uint64
	errors  atomic.Uint64
	mu      sync.Mutex
	latency map[string][]time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{latency: make(map[string][]time.Duration)}
}

func (m *Metrics) Observe(route string, status int, duration time.Duration) {
	m.total.Add(1)
	if status >= 500 {
		m.errors.Add(1)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.latency[route]) < 1000 {
		m.latency[route] = append(m.latency[route], duration)
	}
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "doclens_gateway_requests_total %d\n", m.total.Load())
	fmt.Fprintf(w, "doclens_gateway_errors_total %d\n", m.errors.Load())

	m.mu.Lock()
	defer m.mu.Unlock()
	routes := make([]string, 0, len(m.latency))
	for route := range m.latency {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	for _, route := range routes {
		samples := m.latency[route]
		if len(samples) == 0 {
			continue
		}
		var sum time.Duration
		for _, sample := range samples {
			sum += sample
		}
		route = strings.ReplaceAll(route, `"`, "")
		fmt.Fprintf(w, "doclens_gateway_request_latency_seconds_avg{route=%q} %.6f\n", route, sum.Seconds()/float64(len(samples)))
	}
}
