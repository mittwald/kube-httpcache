package controller

import (
	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
	"os"
)

func (v *VarnishController) Run() error {
	glog.Infof("waiting for initial configuration before starting Varnish")

	v.frontend = watcher.NewEndpointConfig()
	if v.frontendUpdates != nil {
		v.frontend = <-v.frontendUpdates
		if v.varnishSignaller != nil {
			v.varnishSignaller.SetEndpoints(v.frontend)
		}
	}

	v.backend = watcher.NewEndpointConfig()
	if v.backendUpdates != nil {
		v.backend = <-v.backendUpdates
	}

	target, err := os.Create(v.configFile)
	if err != nil {
		return err
	}

	glog.Infof("creating initial VCL config")
	err = v.renderVCL(target, v.frontend.Endpoints, v.frontend.Primary, v.backend.Endpoints, v.backend.Primary)
	if err != nil {
		return err
	}

	v.waitForAdminPort()

	watchErrors := make(chan error)
	go v.watchConfigUpdates(watchErrors)

	for err := range watchErrors {
		if err != nil {
			glog.Warningf("error while watching for updates: %s", err.Error())
		}
	}

	return nil // never gonna happen
}
