package controller

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/martin-helmich/go-varnish-client"
	"os/exec"
	"text/template"
)

func (v *VarnishController) watchConfigUpdates(c *exec.Cmd, errors chan<- error) {
	i := 0

	for {
		i ++

		select {
		case tmplContents := <-v.vclTemplateUpdates:
			glog.Infof("VCL template was updated")

			tmpl, err := template.New("vcl").Parse(string(tmplContents))
			if err != nil {
				errors <- err
				continue
			}

			v.vclTemplate = tmpl

			errors <- v.rebuildConfig(i)

		case newConfig := <-v.backendUpdates:
			glog.Infof("received new backend configuration: %+v", newConfig)

			v.backend = newConfig

			errors <- v.rebuildConfig(i)
		}
	}
}

func (v *VarnishController) rebuildConfig(i int) error {
	buf := new(bytes.Buffer)

	err := v.renderVCL(buf, v.backend.Backends, v.backend.Primary)
	if err != nil {
		return err
	}

	vcl := buf.Bytes()
	glog.V(8).Infof("new VCL: %s", string(vcl))

	client, err := varnishclient.DialTCP(fmt.Sprintf("127.0.0.1:%d", v.AdminPort))
	if err != nil {
		return err
	}

	err = client.Authenticate(v.secret)
	if err != nil {
		return err
	}

	configname := fmt.Sprintf("k8s-upstreamcfg-%d", i)

	err = client.DefineInlineVCL(configname, vcl, "auto")
	if err != nil {
		return err
	}

	err = client.UseVCL(configname)
	if err != nil {
		return err
	}

	if v.currentVCLName == "" {
		v.currentVCLName = "boot"
	}

	if err := client.SetVCLState(v.currentVCLName, varnishclient.VCLStateCold); err != nil {
		glog.V(1).Infof("error while changing state of VCL %s: %s", v.currentVCLName, err)
	}

	v.currentVCLName = configname

	return nil
}
