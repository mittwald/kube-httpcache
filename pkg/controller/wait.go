package controller

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/martin-helmich/go-varnish-client"
	"time"
)

func (v *VarnishController) waitForAdminPort(ctx context.Context) {
	glog.Infof("probing admin port until it is available")
	addr := fmt.Sprintf("127.0.0.1:%d", v.AdminPort)

	for {
		_, err := varnishclient.DialTCP(ctx, addr)
		if err == nil {
			glog.Infof("admin port is available")
			return
		}

		glog.V(6).Infof("admin port is not available yet. waiting")
		time.Sleep(time.Second)
	}
}
