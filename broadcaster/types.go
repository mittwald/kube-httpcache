package broadcaster

import (
	"net/http"
	"sync"
	"time"

	"github.com/mittwald/kube-httpcache/watcher"
)

type Cast struct {
	Request *http.Request
	Attempt int
}

type Broadcaster struct {
	Address        string
	Port           int
	Retries        int
	RetryBackoff   time.Duration
	EndpointScheme string
	server         *http.Server
	endpoints      *watcher.EndpointConfig
	castQueue      chan Cast
	errors         chan error
	mutex          sync.RWMutex
}

func NewBroadcaster(
	address string,
	port int,
	retries int,
	retryBackoff time.Duration,
) *Broadcaster {
	return &Broadcaster{
		Address:        address,
		Port:           port,
		Retries:        retries,
		RetryBackoff:   retryBackoff,
		EndpointScheme: "http",
		endpoints:      watcher.NewEndpointConfig(),
		castQueue:      make(chan Cast),
	}
}
