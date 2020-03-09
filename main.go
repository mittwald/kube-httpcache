package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/mittwald/kube-httpcache/controller"
	"github.com/mittwald/kube-httpcache/watcher"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var opts KubeHTTPProxyFlags

func init() {
	flag.Set("logtostderr", "true")
}

func main() {
	opts.Parse()
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

	go func() {
		for {
			select {
			case err := <-frontendErrors:
				glog.Errorf("error while watching frontends: %s", err.Error())
			case err := <-backendErrors:
				glog.Errorf("error while watching backends: %s", err.Error())
			case err := <-templateErrors:
				glog.Errorf("error while watching template changes: %s", err.Error())
			}
		}
	}()

	varnishController, err := controller.NewVarnishController(
		opts.Varnish.SecretFile,
		opts.Varnish.Storage,
		opts.Frontend.Address,
		opts.Frontend.Port,
		opts.Admin.Address,
		opts.Admin.Port,
		frontendUpdates,
		backendUpdates,
		templateUpdates,
		opts.Varnish.VCLTemplate,
	)
	if err != nil {
		panic(err)
	}

	err = varnishController.Run()
	if err != nil {
		panic(err)
	}
}
