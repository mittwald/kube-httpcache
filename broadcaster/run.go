package broadcaster

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/mittwald/kube-httpcache/watcher"
)

func (b *Broadcaster) Run() error {
	b.server = &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: b,
	}

	return b.server.ListenAndServe()
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mutex.RLock()
	fmt.Fprintf(w, "%+v", b.endpoints)
	b.mutex.RUnlock()
}

func (b *Broadcaster) UpdateEndpoints(e *watcher.EndpointConfig) {
	b.mutex.Lock()
	b.endpoints = e
	b.mutex.Unlock()
}
