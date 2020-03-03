package main

import (
	"flag"
	"time"

	"github.com/golang/glog"
)

type KubeHTTPProxyFlags struct {
	Kubernetes struct {
		Config             string
		RetryBackoffString string
		RetryBackoff       time.Duration
	}
	Frontend struct {
		Address   string
		Port      int
		Watch     bool
		Namespace string
		Service   string
		PortName  string
	}
	Backend struct {
		Watch     bool
		Namespace string
		Service   string
		Port      string
		PortName  string
	}
	Broadcaster struct {
		Enabled            bool
		Address            string
		Port               int
		Retries            int
		RetryBackoffString string
		RetryBackoff       time.Duration
	}
	Admin struct {
		Address string
		Port    int
	}
	Varnish struct {
		SecretFile      string
		Storage         string
		VCLTemplate     string
		VCLTemplatePoll bool
	}
}

func (f *KubeHTTPProxyFlags) Parse() error {
	var err error

	flag.StringVar(&f.Kubernetes.Config, "kubeconfig", "", "kubeconfig file")
	flag.StringVar(&f.Kubernetes.RetryBackoffString, "retry-backoff", "30s", "backoff for Kubernetes API reconnection attempts")

	flag.StringVar(&f.Frontend.Address, "frontend-addr", "0.0.0.0", "TCP address to listen on")
	flag.IntVar(&f.Frontend.Port, "frontend-port", 80, "TCP port to listen on")

	flag.BoolVar(&f.Frontend.Watch, "frontend-watch", false, "watch for Kubernetes frontend updates")
	flag.StringVar(&f.Frontend.Namespace, "frontend-namespace", "", "name of Kubernetes frontend namespace")
	flag.StringVar(&f.Frontend.Service, "frontend-service", "", "name of Kubernetes frontend service")
	flag.StringVar(&f.Frontend.PortName, "frontend-portname", "http", "name of frontend port")

	flag.BoolVar(&f.Backend.Watch, "backend-watch", true, "watch for Kubernetes backend updates")
	flag.StringVar(&f.Backend.Namespace, "backend-namespace", "", "name of Kubernetes backend namespace")
	flag.StringVar(&f.Backend.Service, "backend-service", "", "name of Kubernetes backend service")
	flag.StringVar(&f.Backend.Port, "backend-port", "", "deprecated: name of backend port")
	flag.StringVar(&f.Backend.PortName, "backend-portname", "http", "name of backend port")

	flag.BoolVar(&f.Broadcaster.Enabled, "broadcaster-enabled", false, "enable broadcaster functionality for PURGE and BAN requests")
	flag.StringVar(&f.Broadcaster.Address, "broadcaster-addr", "0.0.0.0", "TCP address for the boradcaster")
	flag.IntVar(&f.Broadcaster.Port, "broadcaster-port", 8090, "TCP port for the boradcaster")
	flag.IntVar(&f.Broadcaster.Retries, "broadcaster-retries", 5, "maximum number of attempts for request broadcasting")
	flag.StringVar(&f.Broadcaster.RetryBackoffString, "broadcaster-backoff", "30s", "backoff for request broadcasting attempts")

	flag.StringVar(&f.Admin.Address, "admin-addr", "127.0.0.1", "TCP address for the Varnish admin")
	flag.IntVar(&f.Admin.Port, "admin-port", 6082, "TCP port for the Varnish admin")

	flag.StringVar(&f.Varnish.SecretFile, "varnish-secret-file", "/etc/varnish/secret", "Varnish secret file")
	flag.StringVar(&f.Varnish.Storage, "varnish-storage", "file,/tmp/varnish-data,1G", "varnish storage config")
	flag.StringVar(&f.Varnish.VCLTemplate, "varnish-vcl-template", "/etc/varnish/default.vcl.tmpl", "VCL template file")
	flag.BoolVar(&f.Varnish.VCLTemplatePoll, "varnish-vcl-template-poll", false, "poll for file changes instead of using inotify (useful on some network filesystems)")

	flag.Parse()

	if len(f.Backend.Port) > 0 {
		f.Backend.PortName = f.Backend.Port
		glog.Warningf("-backend-port flag has been deprecated in favor of -backend-portname and will be removed in future versions")
	}

	f.Kubernetes.RetryBackoff, err = time.ParseDuration(f.Kubernetes.RetryBackoffString)
	if err != nil {
		return err
	}

	f.Broadcaster.RetryBackoff, err = time.ParseDuration(f.Broadcaster.RetryBackoffString)
	if err != nil {
		return err
	}

	return nil
}
