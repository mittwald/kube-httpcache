package broadcaster

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

func (b *Broadcaster) Run() error {
	server := &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: b,
	}

	go b.ProcessCastQueue()

	return server.ListenAndServe()
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.errors <- err
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b.mutex.RLock()
	for _, endpoint := range b.endpoints.Endpoints {
		url := fmt.Sprintf("%s://%s:%s%s", b.EndpointScheme, endpoint.Host, endpoint.Port, r.RequestURI)
		request, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			b.errors <- err
		}
		request.Header = r.Header
		b.castQueue <- Cast{request, 0}
	}
	b.mutex.RUnlock()

	fmt.Fprintf(w, "Request is being broadcasted.")
}

func (b *Broadcaster) ProcessCastQueue() {
	client := &http.Client{}

	for cast := range b.castQueue {
		_, err := client.Do(cast.Request)
		if err != nil {
			cast.Attempt++
			if cast.Attempt < b.Retries {
				go func() {
					time.Sleep(b.RetryBackoff)
					b.castQueue <- cast
				}()
			}
			b.errors <- err
		}
	}
}
