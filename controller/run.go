package controller

import (
	"fmt"
	"github.com/golang/glog"
	"os"
	"os/exec"
)

func (v *VarnishController) Run() error {
	glog.Infof("waiting for initial configuration before starting Varnish")

	initialBackends := <- v.updates
	target, err := os.Create(v.configFile)
	if err != nil {
		return err
	}

	glog.Infof("creating initial VCL config")
	err = v.renderVCL(target, initialBackends.Backends, initialBackends.Primary)
	if err != nil {
		return err
	}

	cmd, errChan := v.startVarnish()

	v.waitForAdminPort()

	watchErrors := make(chan error)
	go v.watchConfigUpdates(cmd, watchErrors)

	go func() {
		for err := range watchErrors {
			glog.Warningf("error while watching for updates: %s", err.Error())
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
