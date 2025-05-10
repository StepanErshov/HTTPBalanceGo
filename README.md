# Load Balancer with Rate Limiting

Simple HTTP load balancer with rate limiting implemented in Go.

## Features

- Round-robin load balancing
- Health checks for backend servers
- Token bucket rate limiting algorithm
- Graceful shutdown
- Configurable via JSON file

## Build and Run

### Prerequisites

- Go 1.21 (and only him if you want to use docker)
- Docker (optional)

### Running locally

1. Clone the repository

2. Build the application:

   ```bash
   go build -o loadbalancer .
   ```
3. Create a config.json file (see example above)
4. Run the application:

    ```bash
    ./loadbalancer --config config.json
    ```

### Running with Docker

1. Build and run with Docker Compose:
    ```bash
    docker-compose up --build
    ```
2. If you want to compile the file in .exe format then do the following
	```bash
	go build -o loadbalancer .
	```
### Testing

Run unit tests:
```bash
go test -v ./...
```

Run benchmark tests:
```bash
go test -bench=. -race
```

Test with Apache Bench:
```bash
ab -n 5000 -c 1000 http://localhost:8080/
```

### Configuration

Modify config.json to change the load balancer settings:

- port: Port to listen on

- backends: List of backend servers to balance between

### Intagration tests

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoadBalancer(t *testing.T) {

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend1"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend2"))
	}))
	defer backend2.Close()

	config := Config{
		Port:     "8080",
		Backends: []string{backend1.URL, backend2.URL},
	}

	lb := NewLoadBalancer(config)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	lb.ServeHTTP(w, req)
	if w.Body.String() != "backend1" {
		t.Errorf("Expected backend1, got %s", w.Body.String())
	}

	w = httptest.NewRecorder()
	lb.ServeHTTP(w, req)
	if w.Body.String() != "backend2" {
		t.Errorf("Expected backend2, got %s", w.Body.String())
	}

	backend1.Close()
	w = httptest.NewRecorder()
	lb.ServeHTTP(w, req)
	if w.Body.String() != "backend2" {
		t.Errorf("Expected backend2 after backend1 failed, got %s", w.Body.String())
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	clientID := "test-client"

	for i := 0; i < 10; i++ {
		if !rl.Allow(clientID, 10, 1) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	if rl.Allow(clientID, 10, 1) {
		t.Error("11th request should be denied")
	}

	time.Sleep(1 * time.Second)

	if !rl.Allow(clientID, 10, 1) {
		t.Error("Request after refill should be allowed")
	}
}

func BenchmarkLoadBalancer(b *testing.B) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	config := Config{
		Port:     "8080",
		Backends: []string{backend.URL},
	}

	lb := NewLoadBalancer(config)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lb.ServeHTTP(w, req)
	}
}
```

### Additional improvements

1. Support for multiple balancing algorithms:
```go
type BalancingAlgorithm int

const (
	RoundRobin BalancingAlgorithm = iota
	LeastConnections
	Random
)

type LoadBalancer struct {
	// ...
	algorithm BalancingAlgorithm
}

func (lb *LoadBalancer) getNextBackend() *url.URL {
	switch lb.algorithm {
	case RoundRobin:
		return lb.getNextRoundRobin()
	case LeastConnections:
		return lb.getLeastConnections()
	case Random:
		return lb.getRandomBackend()
	default:
		return lb.getNextRoundRobin()
	}
}
```

2. Backend Health Checks:

```go
func (lb *LoadBalancer) startHealthChecks() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				lb.healthCheck()
			}
		}
	}()
}
```

3. API for managing rate limiting clients:

```go
func (rl *RateLimiter) AddClient(clientID string, capacity, rate int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	rl.buckets[clientID] = NewTokenBucket(capacity, rate)
}

func (rl *RateLimiter) RemoveClient(clientID string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	delete(rl.buckets, clientID)
}

func (rl *RateLimiter) UpdateClient(clientID string, capacity, rate int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	if bucket, exists := rl.buckets[clientID]; exists {
		bucket.capacity = capacity
		bucket.rate = rate
	}
}
```