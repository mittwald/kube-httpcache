package controller

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ihr-radioedit/go-tracing"
	"os/exec"
	"strings"
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

func (v *VarnishController) rebuildConfig(i int) (err error) {
	txn, ctx := tracing.StartBackgroundTransaction(context.Background(), "varnish update")
	defer func() {
		txn.Finish(err)
	}()

	buf := new(bytes.Buffer)

	err = v.renderVCL(buf, v.frontend.Endpoints, v.frontend.Primary, v.backend.Endpoints, v.backend.Primary)
	if err != nil {
		return err
	}

	vcl := buf.Bytes()
	glog.V(8).Infof("new VCL: %s", string(vcl))

	var client varnishclient.Client
	client, err = varnishclient.DialTCP(fmt.Sprintf("127.0.0.1:%d", v.AdminPort))
	if err != nil {
		return err
	}

	err = client.Authenticate(v.secret)
	if err != nil {
		return err
	}

	configname := fmt.Sprintf("k8s-upstreamcfg-%d", i)

	err = client.DefineInlineVCL(configname, vcl, "warm")
	if err != nil {
		return err
	}

	err = client.UseVCL(configname)
	if err != nil {
		return err
	}

	return nil
}
