package watcher

import (
	"github.com/fsnotify/fsnotify"
	"time"

	"k8s.io/client-go/kubernetes"
)

type BackendConfig struct {
	Backends BackendList
	Primary  *Backend
}

func NewBackendConfig() *BackendConfig {
	return &BackendConfig{
		Backends: []Backend{},
		Primary:  nil,
	}
}

type BackendWatcher struct {
	client      kubernetes.Interface
	namespace   string
	serviceName string
	portName    string

	backendConfig *BackendConfig
	retryBackoff  time.Duration
}

func NewBackendWatcher(client kubernetes.Interface, namespace, serviceName, portName string, retryBackoff time.Duration) *BackendWatcher {
	return &BackendWatcher{
		client:        client,
		namespace:     namespace,
		serviceName:   serviceName,
		portName:      portName,
		backendConfig: NewBackendConfig(),
		retryBackoff:  retryBackoff,
	}
}

type fsnotifyTemplateWatcher struct {
	filename string
	watcher  *fsnotify.Watcher
}

type pollingTemplateWatcher struct {
	filename string
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
