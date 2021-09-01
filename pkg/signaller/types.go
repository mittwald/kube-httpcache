package signaller

import (
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
)

type Signal struct {
	Request *http.Request
	Attempt int
}

type Signaller struct {
	Address        string
	Port           int
	WorkersCount   int
	MaxRetries     int
	RetryBackoff   time.Duration
	EndpointScheme string
	endpoints      *watcher.EndpointConfig
	signalQueue    chan Signal
	errors         chan error
	mutex          sync.RWMutex
}

func NewSignaller(
	address string,
	port int,
	workersCount int,
	maxRetries int,
	retryBackoff time.Duration,
	queueLength int,
) *Signaller {
	if queueLength < 0 {
		queueLength = 0
		glog.Warning("signaller processing queue cannot have a negative length, falling back to default value: %d", queueLength)
	}

	return &Signaller{
		Address:        address,
		Port:           port,
		WorkersCount:   workersCount,
		MaxRetries:     maxRetries,
		RetryBackoff:   retryBackoff,
		EndpointScheme: "http",
		endpoints:      watcher.NewEndpointConfig(),
		signalQueue:    make(chan Signal, queueLength),
		errors:         make(chan error),
	}
}

func (b *Signaller) GetErrors() chan error {
	return b.errors
}

func (b *Signaller) SetEndpoints(e *watcher.EndpointConfig) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.endpoints = e
}
