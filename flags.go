package main

import (
	"flag"
	"time"
)

type KubeHTTPProxyFlags struct {
	Kubernetes struct {
		Config             string
		RetryBackoffString string
		RetryBackoff       time.Duration
	}
	Frontend struct {
		Address string
		Port    int
	}
	Backend struct {
		Namespace string
		Service   string
		Port      string
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
	flag.IntVar(&f.Frontend.Port, "frontend-port", 80, "TCP address to listen on")

	flag.StringVar(&f.Backend.Namespace, "backend-namespace", "", "name of Kubernetes backend namespace")
	flag.StringVar(&f.Backend.Service, "backend-service", "", "name of Kubernetes backend service")
	flag.StringVar(&f.Backend.Port, "backend-port", "http", "name of backend port")
	flag.StringVar(&f.Admin.Address, "admin-addr", "127.0.0.1", "TCP address for the admin port")
	flag.IntVar(&f.Admin.Port, "admin-port", 6082, "TCP address for the admin port")

	flag.StringVar(&f.Varnish.SecretFile, "varnish-secret-file", "/etc/varnish/secret", "Varnish secret file")
	flag.StringVar(&f.Varnish.Storage, "varnish-storage", "file,/tmp/varnish-data,1G", "varnish storage config")
	flag.StringVar(&f.Varnish.VCLTemplate, "varnish-vcl-template", "/etc/varnish/default.vcl.tmpl", "VCL template file")
	flag.BoolVar(&f.Varnish.VCLTemplatePoll, "varnish-vcl-template-poll", false, "poll for file changes instead of using inotify (useful on some network filesystems)")

	flag.Parse()

	f.Kubernetes.RetryBackoff, err = time.ParseDuration(f.Kubernetes.RetryBackoffString)
	if err != nil {
		return err
	}

	return nil
}
