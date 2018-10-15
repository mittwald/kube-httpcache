package watcher

import (
	"fmt"
	"k8s.io/api/core/v1"
	"strconv"
)

type BackendProbe struct {
	URL       string
	Interval  int
	Timeout   int
	Window    int
	Threshold int
}

type Backend struct {
	Name  string
	Host  string
	Port  string
	Probe *BackendProbe
}

type BackendList []Backend

func (l BackendList) EqualsEndpoints(ep v1.EndpointSubset) bool {
	if len(l) != len(ep.Addresses) {
		return false
	}

	matchingAddresses := map[string]bool{}
	for i := range l {
		matchingAddresses[l[i].Host] = true
	}

	for i := range ep.Addresses {
		h := ep.Addresses[i].IP
		_, ok := matchingAddresses[h]
		if !ok {
			return false
		}
	}

	return true
}

func (l BackendList) Contains(b *Backend) bool {
	if b == nil {
		return false
	}

	for i := range l {
		if l[i].Host == b.Host && l[i].Port == b.Port {
			return true
		}
	}

	return false
}

func BackendListFromSubset(ep v1.EndpointSubset, portName string) (BackendList, error) {
	var port int32

	l := make(BackendList, len(ep.Addresses))

	for i := range ep.Ports {
		if ep.Ports[i].Name == portName {
			port = ep.Ports[i].Port
		}
	}

	if port == 0 {
		return nil, fmt.Errorf("port '%s' not found in endpoint list", portName)
	}

	for i := range ep.Addresses {
		a := &ep.Addresses[i]

		if a.TargetRef != nil {
			l[i].Name = a.TargetRef.Name
		}

		l[i].Host = a.IP
		l[i].Port = strconv.Itoa(int(port))
	}

	return l, nil
}
