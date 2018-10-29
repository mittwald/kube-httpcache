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

type TemplateWatcher struct {
	filename string
	watcher  *fsnotify.Watcher
}

func MustNewTemplateWatcher(filename string) *TemplateWatcher {
	w, err := NewTemplateWatcher(filename)
	if err != nil {
		panic(err)
	}

	return w
}

func NewTemplateWatcher(filename string) (*TemplateWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Add(filename)
	if err != nil {
		return nil, err
	}

	return &TemplateWatcher{
		filename: filename,
		watcher:  watcher,
	}, nil
}
