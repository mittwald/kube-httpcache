package broadcaster

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
)

func (b *Broadcaster) Run() error {
	server := &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: b,
	}

	for i := 0; i < b.WorkersCount; i++ {
		go b.ProcessCastQueue()
	}

	return server.ListenAndServe()
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.errors <- err
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	glog.V(5).Infof("received a broadcast request: %+v", r)

	b.mutex.RLock()
	for _, endpoint := range b.endpoints.Endpoints {
		url := fmt.Sprintf("%s://%s:%s%s", b.EndpointScheme, endpoint.Host, endpoint.Port, r.RequestURI)
		request, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			b.errors <- err
		}
		request.Header = r.Header
		request.Host = r.Host
		request.Header.Set("X-Forwarded-For", r.RemoteAddr)
		b.castQueue <- Cast{request, 0}
	}
	b.mutex.RUnlock()

	fmt.Fprintf(w, "Request is being broadcasted.")
}

func (b *Broadcaster) ProcessCastQueue() {
	client := &http.Client{}

	for cast := range b.castQueue {
		response, err := client.Do(cast.Request)
		if err != nil {
			glog.Errorf("broadcasting error: %v", err.Error())
			glog.Infof("retring in %v", b.RetryBackoff)
			b.Retry(cast)
		} else if response.StatusCode >= 400 && response.StatusCode <= 599 {
			glog.Warningf("broadcasting error: %v", response.Status)
			glog.Infof("retring in %v", b.RetryBackoff)
			b.Retry(cast)
		} else {
			glog.V(5).Infof("recieved an endpoint response: %+v", response)
		}
	}
}

func (b *Broadcaster) Retry(cast Cast) {
	cast.Attempt++
	if cast.Attempt < b.MaxRetries {
		go func() {
			time.Sleep(b.RetryBackoff)
			b.castQueue <- cast
		}()
	}
}
