package controller

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"text/template"

	"github.com/golang/glog"
	varnishclient "github.com/martin-helmich/go-varnish-client"
)

func (v *VarnishController) watchConfigUpdates(c *exec.Cmd, errors chan<- error) {
	i := 0

	for {
		i++

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

		case newConfig := <-v.frontendUpdates:
			glog.Infof("received new frontend configuration: %+v", newConfig)

			v.frontend = newConfig

			if v.varnishSignaller != nil {
				v.varnishSignaller.SetEndpoints(v.frontend)
			}

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
	ctx := context.Background()

	err := v.renderVCL(buf, v.frontend.Endpoints, v.frontend.Primary, v.backend.Endpoints, v.backend.Primary)
	if err != nil {
		return err
	}

	vcl := buf.Bytes()
	glog.V(8).Infof("new VCL: %s", string(vcl))

	client, err := varnishclient.DialTCP(ctx, fmt.Sprintf("127.0.0.1:%d", v.AdminPort))
	if err != nil {
		return err
	}

	err = client.Authenticate(ctx, v.secret)
	if err != nil {
		return err
	}

	configname := fmt.Sprintf("k8s-upstreamcfg-%d", i)

	err = client.DefineInlineVCL(ctx, configname, vcl, "auto")
	if err != nil {
		return err
	}

	err = client.UseVCL(ctx, configname)
	if err != nil {
		return err
	}

	if v.currentVCLName == "" {
		v.currentVCLName = "boot"
	}

	if err := client.SetVCLState(ctx, v.currentVCLName, varnishclient.VCLStateCold); err != nil {
		glog.V(1).Infof("error while changing state of VCL %s: %s", v.currentVCLName, err)
	}

	v.currentVCLName = configname

	return nil
}
