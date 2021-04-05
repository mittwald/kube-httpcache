package signaller

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
)

func (b *Signaller) Run() error {
	server := &http.Server{
		Addr:    b.Address + ":" + strconv.Itoa(b.Port),
		Handler: b,
	}

	glog.Infof("running signaller on %v", server.Addr)

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
	glog.V(5).Infof("endpoints=%#v", b.endpoints)


	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for _, endpoint := range b.endpoints.Endpoints {
		url := fmt.Sprintf("%s://%s:%s%s", b.EndpointScheme, endpoint.Host, endpoint.Port, r.RequestURI)
		glog.Infof("sending signal url=%v", url)
		request, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			b.errors <- err
		}
		request.Header = r.Header
		request.Host = r.Host
		request.Header.Set("X-Forwarded-For", r.RemoteAddr)
		b.signalQueue <- Signal{request, 0}
	}

	fmt.Fprintf(w, "Signal request is being broadcasted.")
}

func (b *Signaller) ProcessSignalQueue() {
	client := &http.Client{}

	glog.Infof("Signaller.ProcessSingnalQueue()")

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
