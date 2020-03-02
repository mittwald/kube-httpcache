package watcher

import (
	"time"

	"github.com/fsnotify/fsnotify"

	"k8s.io/client-go/kubernetes"
)

type EndpointConfig struct {
	Endpoints EndpointList
	Primary   *Endpoint
}

func NewEndpointConfig() *EndpointConfig {
	return &EndpointConfig{
		Endpoints: []Endpoint{},
		Primary:   nil,
	}
}

type EndpointWatcher struct {
	client      kubernetes.Interface
	namespace   string
	serviceName string
	portName    string

	endpointConfig *EndpointConfig
	retryBackoff   time.Duration
}

func NewEndpointWatcher(client kubernetes.Interface, namespace, serviceName, portName string, retryBackoff time.Duration) *EndpointWatcher {
	return &EndpointWatcher{
		client:         client,
		namespace:      namespace,
		serviceName:    serviceName,
		portName:       portName,
		endpointConfig: NewEndpointConfig(),
		retryBackoff:   retryBackoff,
	}
}

type fsnotifyTemplateWatcher struct {
	filename string
	watcher  *fsnotify.Watcher
}

type pollingTemplateWatcher struct {
	filename              string
	lastObservedTimestamp time.Time
}

type TemplateWatcher interface {
	Run() (chan []byte, chan error)
}

func MustNewTemplateWatcher(filename string, polling bool) TemplateWatcher {
	w, err := NewTemplateWatcher(filename, polling)
	if err != nil {
		panic(err)
	}

	return w
}

func NewTemplateWatcher(filename string, polling bool) (TemplateWatcher, error) {
	if polling {
		return &pollingTemplateWatcher{
			filename: filename,
		}, nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Add(filename)
	if err != nil {
		return nil, err
	}

	return &fsnotifyTemplateWatcher{
		filename: filename,
		watcher:  watcher,
	}, nil
}
