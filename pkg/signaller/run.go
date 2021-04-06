package signaller

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
)

func (b *Signaller) Run() error {
	service := os.Getenv("DD_SERVICE")
	server := &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: httptrace.WrapHandler(b, service, "signaller"),
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
	defer b.mutex.RUnlock()

	// Compute XFF once because it will be the same for all requests
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		xff += ", "
	}
	xff += r.RemoteAddr

	// Copy the incoming header so we don't mutate it. This is done outside
	// the loop, and we share the same header object for all requests.
	h := r.Header.Clone()
	h.Set("X-Forwarded-For", xff)

	for _, endpoint := range b.endpoints.Endpoints {
		url := fmt.Sprintf("%s://%s:%s%s", b.EndpointScheme, endpoint.Host, endpoint.Port, r.RequestURI)
		request, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			b.errors <- err
		}
		request.Header = h
		request.Host = r.Host
		b.signalQueue <- Signal{request, 0}
	}

	fmt.Fprintf(w, "Signal request is being broadcasted.")
}

func (b *Signaller) ProcessSignalQueue() {
	client := &http.Client{}

	for signal := range b.signalQueue {
		response, err := client.Do(signal.Request)
		if err != nil {
			glog.Errorf("signal broadcast error: %v", err.Error())
			glog.Infof("retrying in %v", b.RetryBackoff)
			b.Retry(signal)
		} else if response.StatusCode >= 400 && response.StatusCode <= 599 {
			glog.Warningf("signal broadcast error: unusual status code from %s: %v", response.Request.URL.Host, response.Status)
			glog.Infof("retrying in %v", b.RetryBackoff)
			b.Retry(signal)
		} else {
			glog.V(5).Infof("received a signal response from %s: %+v", response.Request.URL.Host, response)
		}
		if response != nil {
			_ = response.Body.Close()
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
