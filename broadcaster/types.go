package broadcaster

import (
	"net/http"
	"time"

	"github.com/mittwald/kube-httpcache/watcher"
)

type Cast struct {
	Request *http.Request
	Attempt int
}

type Broadcaster struct {
	Address      string
	Port         int
	Retries      int
	RetryBackoff time.Duration
	server       *http.Server
	frontends    *watcher.EndpointConfig
	casts        chan Cast
}

func NewBroadcaster(
	address string,
	port int,
	retries int,
	retryBackoff time.Duration,
) *Broadcaster {
	return &Broadcaster{
		Address:      address,
		Port:         port,
		Retries:      retries,
		RetryBackoff: retryBackoff,
		frontends:    watcher.NewEndpointConfig(),
		casts:        make(chan Cast),
	}
}
