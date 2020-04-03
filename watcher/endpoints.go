package watcher

import (
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
)

type EndpointProbe struct {
	URL       string
	Interval  int
	Timeout   int
	Window    int
	Threshold int
}

type Endpoint struct {
	Name  string
	Host  string
	Port  string
	Probe *EndpointProbe
}

type EndpointList []Endpoint

func (l EndpointList) EqualsEndpoints(ep v1.EndpointSubset) bool {
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

func (l EndpointList) Contains(b *Endpoint) bool {
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

func EndpointListFromSubset(ep v1.EndpointSubset, portName string) (EndpointList, error) {
	var port int32

	l := make(EndpointList, len(ep.Addresses))

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
