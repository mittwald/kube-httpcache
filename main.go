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

	backendWatcher := watcher.NewBackendWatcher(
		client,
		opts.Backend.Namespace,
		opts.Backend.Service,
		opts.Backend.Port,
		opts.Kubernetes.RetryBackoff,
	)

	templateWatcher := watcher.MustNewTemplateWatcher(opts.Varnish.VCLTemplate, opts.Varnish.VCLTemplatePoll)

	backendUpdates, backendErrors := backendWatcher.Run()
	templateUpdates, templateErrors := templateWatcher.Run()

	go func() {
		for {
			select {
			case err := <- backendErrors:
				glog.Errorf("error while watching backends: %s", err.Error())
			case err := <- templateErrors:
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
