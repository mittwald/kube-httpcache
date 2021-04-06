package signaller

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
)

func (b *Signaller) Run() error {
	server := &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: b,
	}

	for i := 0; i < b.WorkersCount; i++ {
		go b.ProcessSignalQueue()
	}

	return server.ListenAndServe()
}

func (b *Signaller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.errors <- err
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	glog.V(5).Infof("received a signal request: %+v", r)

	b.mutex.RLock()
	endpoints := make([]watcher.Endpoint, len(b.endpoints.Endpoints))
	copy(endpoints, b.endpoints.Endpoints)
	b.mutex.RUnlock()

	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s://%s:%s%s", b.EndpointScheme, endpoint.Host, endpoint.Port, r.RequestURI)
		request, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			b.errors <- err
		}
		request.Header = r.Header.Clone()
		request.Host = r.Host
		request.Header.Set("X-Forwarded-For", r.RemoteAddr)
		b.signalQueue <- Signal{request, 0}
	}

	fmt.Fprintf(w, "Signal request is being broadcasted.")
}

func (b *Signaller) ProcessSignalQueue() {
	client := &http.Client{}

	for signal := range b.signalQueue {
		response, err := client.Do(signal.Request)
		if err != nil {
			glog.Errorf("singal broadcast error: %v", err.Error())
			glog.Infof("retring in %v", b.RetryBackoff)
			b.Retry(signal)
		} else if response.StatusCode >= 400 && response.StatusCode <= 599 {
			glog.Warningf("singal broadcast error: unusual status code from %s: %v", response.Request.URL.Host, response.Status)
			glog.Infof("retring in %v", b.RetryBackoff)
			b.Retry(signal)
		} else {
			glog.V(5).Infof("recieved a signal response from %s: %+v", response.Request.URL.Host, response)
		}

		if response != nil {
			if err := response.Body.Close(); err != nil {
				glog.Error("error on closing response body:", err)
			}
		}
	}
}

func (b *Signaller) Retry(signal Signal) {
	signal.Attempt++
	if signal.Attempt < b.MaxRetries {
		go func() {
			time.Sleep(b.RetryBackoff)
			b.signalQueue <- signal
		}()
	}
}
