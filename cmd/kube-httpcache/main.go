package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/cmd/kube-httpcache/internal"
	"github.com/mittwald/kube-httpcache/pkg/controller"
	"github.com/mittwald/kube-httpcache/pkg/signaller"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var opts internal.KubeHTTPProxyFlags

func init() {
	flag.Set("logtostderr", "true")
}

func main() {
	if err := opts.Parse(); err != nil {
		panic(err)
	}

	glog.Infof("running kube-httpcache with following options: %+v", opts)

	var config *rest.Config
	var err error
	var client kubernetes.Interface

	if opts.Kubernetes.Config == "" {
		glog.Infof("using in-cluster configuration")
		config, err = rest.InClusterConfig()
	} else {
		glog.Infof("using configuration from '%s'", opts.Kubernetes.Config)
		config, err = clientcmd.BuildConfigFromFlags("", opts.Kubernetes.Config)
	}

	if err != nil {
		panic(err)
	}

	client = kubernetes.NewForConfigOrDie(config)

	var frontendUpdates chan *watcher.EndpointConfig
	var frontendErrors chan error
	if opts.Frontend.Watch {
		frontendWatcher := watcher.NewEndpointWatcher(
			client,
			opts.Frontend.Namespace,
			opts.Frontend.Service,
			opts.Frontend.PortName,
			opts.Kubernetes.RetryBackoff,
		)
		frontendUpdates, frontendErrors = frontendWatcher.Run()
	}

	var backendUpdates chan *watcher.EndpointConfig
	var backendErrors chan error
	if opts.Backend.Watch {
		backendWatcher := watcher.NewEndpointWatcher(
			client,
			opts.Backend.Namespace,
			opts.Backend.Service,
			opts.Backend.PortName,
			opts.Kubernetes.RetryBackoff,
		)
		backendUpdates, backendErrors = backendWatcher.Run()
	}

	templateWatcher := watcher.MustNewTemplateWatcher(opts.Varnish.VCLTemplate, opts.Varnish.VCLTemplatePoll)
	templateUpdates, templateErrors := templateWatcher.Run()

	var varnishSignaller *signaller.Signaller
	var varnishSignallerErrors chan error
	if opts.Signaller.Enable {
		varnishSignaller = signaller.NewSignaller(
			opts.Signaller.Address,
			opts.Signaller.Port,
			opts.Signaller.WorkersCount,
			opts.Signaller.MaxRetries,
			opts.Signaller.RetryBackoff,
		)
		varnishSignallerErrors = varnishSignaller.GetErrors()

		go func() {
			err = varnishSignaller.Run()
			if err != nil {
				panic(err)
			}
		}()
	}

	go func() {
		for {
			select {
			case err := <-frontendErrors:
				glog.Errorf("error while watching frontends: %s", err.Error())
			case err := <-backendErrors:
				glog.Errorf("error while watching backends: %s", err.Error())
			case err := <-templateErrors:
				glog.Errorf("error while watching template changes: %s", err.Error())
			case err := <-varnishSignallerErrors:
				glog.Errorf("error while running varnish signaller: %s", err.Error())
			}
		}
	}()

	varnishController, err := controller.NewVarnishController(
		opts.Varnish.SecretFile,
		opts.Varnish.Storage,
		opts.Varnish.AdditionalParameters,
		opts.Varnish.WorkingDir,
		opts.Frontend.Address,
		opts.Frontend.Port,
		opts.Admin.Address,
		opts.Admin.Port,
		frontendUpdates,
		backendUpdates,
		templateUpdates,
		varnishSignaller,
		opts.Varnish.VCLTemplate,
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		s := <-signals

		glog.Infof("received signal %s", s)
		cancel()
	}()

	err = varnishController.Run(ctx)
	if err != nil {
		panic(err)
	}
}
