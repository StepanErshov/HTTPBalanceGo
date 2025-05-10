package loadbalancer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer unhealthyServer.Close()

	lb := NewLoadBalancer(Config{
		Backends: []string{healthyServer.URL, unhealthyServer.URL},
	})

	if len(lb.backends) != 1 {
		t.Errorf("Expected 1 healthy backend, got %d", len(lb.backends))
	}
}