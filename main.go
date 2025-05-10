package main

import (
	"encoding/json"
	"flag"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Port     string   `json:"port"`
	Backends []string `json:"backends"`
}

type LoadBalancer struct {
	config         Config
	backends       []*url.URL
	proxy          *httputil.ReverseProxy
	currentBackend int
	mutex          sync.Mutex
	client        *http.Client
}

func NewLoadBalancer(config Config) *LoadBalancer {
	lb := &LoadBalancer{
		config:   config,
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	for _, backend := range config.Backends {
		backendURL, err := url.Parse(backend)
		if err != nil {
			log.Printf("Error parsing backend URL %s: %v", backend, err)
			continue
		}
		lb.backends = append(lb.backends, backendURL)
	}

	lb.healthCheck()

	return lb
}

func (lb *LoadBalancer) healthCheck() {
	var healthyBackends []*url.URL

	for _, backend := range lb.backends {
		resp, err := lb.client.Get(backend.String() + "/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			log.Printf("Backend %s is unavailable", backend.String())
			continue
		}
		healthyBackends = append(healthyBackends, backend)
		resp.Body.Close()
	}

	lb.mutex.Lock()
	lb.backends = healthyBackends
	if len(lb.backends) == 0 {
		log.Fatal("All backends are unavailable")
	}
	if lb.currentBackend >= len(lb.backends) {
		lb.currentBackend = 0
	}
	lb.mutex.Unlock()
}

func (lb *LoadBalancer) getNextBackend() *url.URL {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if len(lb.backends) == 0 {
		return nil
	}

	backend := lb.backends[lb.currentBackend]
	lb.currentBackend = (lb.currentBackend + 1) % len(lb.backends)
	return backend
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.getNextBackend()
	if backend == nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	log.Printf("Forwarding request to %s", backend.String())

	proxy := httputil.NewSingleHostReverseProxy(backend)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Error proxying to %s: %v", backend.String(), err)
		lb.healthCheck()
		http.Error(w, "Bad gateway", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
}

func main() {
	configFile := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	lb := NewLoadBalancer(config)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: lb,
	}

	go func() {
		log.Printf("Load balancer started on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}

	log.Println("Server stopped")
}