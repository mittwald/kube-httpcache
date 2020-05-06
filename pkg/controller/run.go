package controller

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
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

	cmd, errChan := v.startVarnish()

	v.waitForAdminPort()
	v.startPrometheusVarnishExporter()

	watchErrors := make(chan error)
	go v.watchConfigUpdates(cmd, watchErrors)

	go func() {
		for err := range watchErrors {
			if err != nil {
				glog.Warningf("error while watching for updates: %s", err.Error())
			}
		}
	}()

	return <-errChan
}

func (v *VarnishController) startVarnish() (*exec.Cmd, <-chan error) {
	c := exec.Command(
		"varnishd",
		"-F",
		"-f", v.configFile,
		"-S", v.SecretFile,
		"-s", v.Storage,
		"-a", fmt.Sprintf("%s:%d", v.FrontendAddr, v.FrontendPort),
		"-T", fmt.Sprintf("%s:%d", v.AdminAddr, v.AdminPort),
		"-a", ":6085",
		"-a", ":6086,PROXY",
	)

	c.Dir = "/"
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	r := make(chan error)

	go func() {
		err := c.Run()
		r <- err
	}()

	return c, r
}

func (v *VarnishController) startPrometheusVarnishExporter() {
	c := exec.Command(
		"prometheus_varnish_exporter",
	)
	c.Dir = "/"
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	go func() {
		err := c.Run()
		glog.Errorf("prometheus_varnish_exporter exited: %v", err)
	}()
}