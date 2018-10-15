package controller

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/martin-helmich/go-varnish-client"
	"os/exec"
)

func (v *VarnishController) watchConfigUpdates(c *exec.Cmd, errors chan<- error) {
	i := 0
	for newConfig := range v.updates {
		i ++
		glog.Infof("received new backend configuration: %+v", newConfig)

		buf := new(bytes.Buffer)

		err := v.renderVCL(buf, newConfig.Backends, newConfig.Primary)
		if err != nil {
			errors <- err
			continue
		}

		vcl := buf.Bytes()
		glog.V(8).Infof("new VCL: %s", string(vcl))

		client, err := varnishclient.DialTCP(fmt.Sprintf("127.0.0.1:%d", v.AdminPort))
		if err != nil {
			errors <- err
			continue
		}

		err = client.Authenticate(v.secret)
		if err != nil {
			errors <- err
			continue
		}

		configname := fmt.Sprintf("k8s-upstreamcfg-%d", i)

		err = client.DefineInlineVCL(configname, vcl, "warm")
		if err != nil {
			errors <- err
			continue
		}

		err = client.UseVCL(configname)
		if err != nil {
			errors <- err
			continue
		}
	}
}
