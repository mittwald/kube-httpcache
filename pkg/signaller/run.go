package signaller

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/pkg/watcher"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	signallerRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kube_httpcache_signaller_requests_total",
		Help: "The total number of incoming requests to Signaller",
	})

	signallerErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kube_httpcache_signaller_errors_total",
		Help: "The total number of errors for incomming requests to Signaller",
	})

	signallerResponseTime = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "kube_httpcache_signaller_durations_seconds",
		Help:       "The Signaller response time",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})

	signallerUpstreamErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kube_httpcache_signaller_upstream_errors_total",
		Help: "The total number of errors for outgoing requests to upstreams",
	})

	signallerQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kube_httpcache_signaller_queue_length",
		Help: "The length of signaller queue",
	})

	signallerQueueLatency = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "kube_httpcache_signaller_queue_latency",
		Help:       "The Signaller queue latency",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})
)

func (b *Signaller) Run() error {
	server := &http.Server{
		Addr: b.Address + ":" + strconv.Itoa(b.Port),
	}

	for i := 0; i < b.WorkersCount; i++ {
		go b.ProcessSignalQueue()
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", b.Serve)

	return server.ListenAndServe()
}

func (b *Signaller) Serve(w http.ResponseWriter, r *http.Request) {
	t := time.Now()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		b.errors <- err
		http.Error(w, err.Error(), http.StatusInternalServerError)

		signallerErrorsTotal.Inc()
		return
	}

	glog.V(5).Infof("received a signal request: %+v", r)
	signallerRequestsTotal.Inc()

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
		signallerQueueLength.Set(float64(len(b.signalQueue)))
		tt := time.Now()

		b.signalQueue <- Signal{request, 0}

		signallerQueueLatency.Observe(time.Since(tt).Seconds())
	}

	fmt.Fprintf(w, "Signal request is being broadcasted.")
	signallerResponseTime.Observe(time.Since(t).Seconds())
}

func (b *Signaller) ProcessSignalQueue() {
	client := &http.Client{}
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if b.MaxConnsPerHost != -1 {
		transport.MaxConnsPerHost = b.MaxConnsPerHost
	}

	if b.MaxIdleConns != -1 {
		transport.MaxIdleConns = b.MaxIdleConns
	}

	if b.MaxIdleConnsPerHost != -1 {
		transport.MaxIdleConnsPerHost = b.MaxIdleConnsPerHost
	}

	client.Transport = transport

	if b.UpstreamRequestTimeout != 0 {
		client.Timeout = b.UpstreamRequestTimeout
	}

	for signal := range b.signalQueue {
		response, err := client.Do(signal.Request)
		if err != nil {
			glog.Errorf("signal broadcast error: %v", err.Error())
			signallerUpstreamErrorsTotal.Inc()
			b.Retry(signal)
		} else if response.StatusCode >= 400 && response.StatusCode <= 599 {
			glog.Warningf("signal broadcast error: unusual status code from %s: %v", response.Request.URL.Host, response.Status)
			signallerUpstreamErrorsTotal.Inc()
			b.Retry(signal)
		} else {
			glog.V(5).Infof("received a signal response from %s: %+v", response.Request.URL.Host, response)
		}

		if response != nil {
			if _, err := io.Copy(ioutil.Discard, response.Body); err != nil {
				glog.Error("error on discarding response body for connection reuse:", err)
			}

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
			glog.Infof("retrying in %v", b.RetryBackoff)
			time.Sleep(b.RetryBackoff)
			b.signalQueue <- signal
		}()
	}
}
