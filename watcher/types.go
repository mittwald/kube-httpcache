package watcher

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
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
	updates     chan *BackendConfig

	backendConfig *BackendConfig
}

func NewBackendWatcher(client kubernetes.Interface, namespace, serviceName, portName string) *BackendWatcher {
	return &BackendWatcher{
		client:        client,
		namespace:     namespace,
		serviceName:   serviceName,
		portName:      portName,
		updates:       make(chan *BackendConfig),
		backendConfig: NewBackendConfig(),
	}
}

func (v *BackendWatcher) Run() chan *BackendConfig {
	go v.watch()

	return v.updates
}

func (v *BackendWatcher) watch() {
	for {
		w, err := v.client.CoreV1().Endpoints(v.namespace).Watch(metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", v.serviceName).String(),
		})

		if err != nil {
			panic(err)
		}

		c := w.ResultChan()
		for ev := range c {
			if ev.Type == watch.Error {
				glog.Warningf("error while watching: %+v", ev.Object)
				continue
			}

			if ev.Type != watch.Added && ev.Type != watch.Modified {
				continue
			}

			endpoint := ev.Object.(*v1.Endpoints)

			if len(endpoint.Subsets) == 0 || len(endpoint.Subsets[0].Addresses) == 0 {
				glog.Warningf("service '%s' has no endpoints", v.serviceName)

				v.backendConfig = NewBackendConfig()

				continue
			}

			if v.backendConfig.Backends.EqualsEndpoints(endpoint.Subsets[0]) {
				glog.Infof("endpoints did not change")
				continue
			}

			var addresses []v1.EndpointAddress
			for _, a := range endpoint.Subsets[0].Addresses {
				puid := string(a.TargetRef.UID)

				po, err := v.client.CoreV1().Pods(v.namespace).Get(a.TargetRef.Name, metav1.GetOptions{})

				if err != nil {
					glog.Errorf("error while locating endpoint : %s", err.Error())
					continue
				}

				if len(po.Status.Conditions) > 0 && po.Status.Conditions[0].Status != v1.ConditionTrue {
					glog.Infof("skipping endpoint (not healthy): %s", puid)
					continue
				}

				addresses = append(addresses, a)
			}
			endpoint.Subsets[0].Addresses = addresses

			newConfig := NewBackendConfig()

			newBackendList, err := BackendListFromSubset(endpoint.Subsets[0], v.portName)
			if err != nil {
				glog.Errorf("error while building backend list: %s", err.Error())
				continue
			}

			if v.backendConfig.Primary != nil && newBackendList.Contains(v.backendConfig.Primary) {
				newConfig.Primary = v.backendConfig.Primary
			} else {
				newConfig.Primary = &newBackendList[0]
			}

			newConfig.Backends = newBackendList

			v.backendConfig = newConfig
			v.updates <- newConfig
		}

		glog.V(5).Info("watch has ended. starting new watch")
	}
}
