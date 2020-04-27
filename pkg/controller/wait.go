package controller

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/martin-helmich/go-varnish-client"
	"time"
)

func (v *VarnishController) waitForAdminPort(ctx context.Context) error {
	glog.Infof("probing admin port until it is available")
	addr := fmt.Sprintf("127.0.0.1:%d", v.AdminPort)

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			_, err := varnishclient.DialTCP(ctx, addr)
			if err == nil {
				glog.Infof("admin port is available")
				return nil
			}

			glog.V(6).Infof("admin port is not available yet. waiting")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
