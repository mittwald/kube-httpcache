package watcher

import (
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
)

func (v *EndpointWatcher) Run() (chan *EndpointConfig, chan error) {
	updates := make(chan *EndpointConfig)
	errors := make(chan error)

	go v.watch(updates, errors)

	return updates, errors
}

func (v *EndpointWatcher) watch(updates chan *EndpointConfig, errors chan error) {
	for {
		w, err := v.client.CoreV1().Endpoints(v.namespace).Watch(metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", v.serviceName).String(),
		})

		if err != nil {
			glog.Errorf("error while establishing watch: %s", err.Error())
			glog.Infof("retrying after %s", v.retryBackoff.String())

			time.Sleep(v.retryBackoff)
			continue
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

				v.endpointConfig = NewEndpointConfig()

				continue
			}

			if v.endpointConfig.Endpoints.EqualsEndpoints(endpoint.Subsets[0]) {
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

			if len(addresses) == 0 {
				glog.Warningf("service '%s' has no endpoint that is ready", v.serviceName)
				v.endpointConfig = NewEndpointConfig()
				continue
			}

			endpoint.Subsets[0].Addresses = addresses

			newConfig := NewEndpointConfig()

			newBackendList, err := EndpointListFromSubset(endpoint.Subsets[0], v.portName)
			if err != nil {
				glog.Errorf("error while building backend list: %s", err.Error())
				continue
			}

			if v.endpointConfig.Primary != nil && newBackendList.Contains(v.endpointConfig.Primary) {
				newConfig.Primary = v.endpointConfig.Primary
			} else {
				newConfig.Primary = &newBackendList[0]
			}

			newConfig.Endpoints = newBackendList

			v.endpointConfig = newConfig
			updates <- newConfig
		}

		glog.V(5).Info("watch has ended. starting new watch")
	}
}
