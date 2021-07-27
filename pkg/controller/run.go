package controller

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
	"os"
	"os/exec"
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

	return nil  // never gonna happen
}

func (v *VarnishController) startVarnish() (*exec.Cmd, <-chan error) {
	args := []string{
		"-F",
		"-f", v.configFile,
		"-S", v.SecretFile,
		"-s", v.Storage,
		"-a", fmt.Sprintf("%s:%d", v.FrontendAddr, v.FrontendPort),
		"-T", fmt.Sprintf("%s:%d", v.AdminAddr, v.AdminPort),
	}

	if v.name != "" {
		args = append(args, "-n", v.name)
	}

	for _, a := range v.addresses {
		args = append(args, "-a", a)
	}

	for _, p := range v.parameters {
		args = append(args, "-p", p)
	}

	c := exec.Command(
		v.Executable,
		args...,
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
