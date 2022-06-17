package controller

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	varnishclient "github.com/martin-helmich/go-varnish-client"
)

func (v *VarnishController) watchConfigUpdates(ctx context.Context, c *exec.Cmd, errors chan<- error) {
	for {
		select {
		case tmplContents := <-v.vclTemplateUpdates:
			glog.Infof("VCL template has been updated")

			tmpl, err := template.New("vcl").Parse(string(tmplContents))
			if err != nil {
				errors <- err
				continue
			}

			v.vclTemplate = tmpl

			errors <- v.rebuildConfig(ctx)

		case newConfig := <-v.frontendUpdates:
			glog.Infof("received new frontend configuration: %+v", newConfig)

			v.frontend = newConfig

			if v.varnishSignaller != nil {
				v.varnishSignaller.SetEndpoints(v.frontend)
			}

			errors <- v.rebuildConfig(ctx)

		case newConfig := <-v.backendUpdates:
			glog.Infof("received new backend configuration: %+v", newConfig)

			v.backend = newConfig

			errors <- v.rebuildConfig(ctx)

		case <-ctx.Done():
			errors <- ctx.Err()
			return
		}
	}
}

func (v *VarnishController) rebuildConfig(ctx context.Context) error {
	buf := new(bytes.Buffer)

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

	secret, err := ioutil.ReadFile(v.SecretFile)
	if err != nil {
		return err
	}

	err = client.Authenticate(ctx, secret)
	if err != nil {
		return err
	}

	maxVclParam, err := client.GetParameter(ctx, "max_vcl")
	if err != nil {
		return err
	}

	maxVcl, err := strconv.Atoi(maxVclParam.Value)
	if err != nil {
		return err
	}

	loadedVcl, err := client.ListVCL(ctx)
	if err != nil {
		return err
	}

	availableVcl := make([]varnishclient.VCLConfig, 0)

	for i := range loadedVcl {
		if loadedVcl[i].Status == varnishclient.VCLAvailable {
			availableVcl = append(availableVcl, loadedVcl[i])
		}
	}

	if len(loadedVcl) >= maxVcl {
		// we're abusing the fact that "boot" < "reload"
		sort.Slice(availableVcl, func(i, j int) bool {
			return availableVcl[i].Name < availableVcl[j].Name
		})

		for i := 0; i < len(loadedVcl)-maxVcl+1; i++ {
			glog.V(8).Infof("discarding VCL: %s", availableVcl[i].Name)

			err = client.DiscardVCL(ctx, availableVcl[i].Name)
			if err != nil {
				return err
			}
		}
	}

	configname := strings.ReplaceAll(time.Now().Format("reload_20060102_150405.00000"), ".", "_")

	err = client.DefineInlineVCL(ctx, configname, vcl, varnishclient.VCLStateAuto)
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
