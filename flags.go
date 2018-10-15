package main

import "flag"

type KubeHTTPProxyFlags struct {
	Kubernetes struct {
		Config string
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
		SecretFile  string
		Storage     string
		VCLTemplate string
	}
}

func (f *KubeHTTPProxyFlags) Parse() {
	flag.StringVar(&f.Kubernetes.Config, "kubeconfig", "", "kubeconfig file")
	flag.StringVar(&f.Frontend.Address, "frontend-addr", "0.0.0.0", "TCP address to listen on")
	flag.IntVar(&f.Frontend.Port, "frontend-port", 80, "TCP address to listen on")

	flag.StringVar(&f.Backend.Service, "backend-namespace", "", "name of Kubernetes backend namespace")
	flag.StringVar(&f.Backend.Service, "backend-service", "", "name of Kubernetes backend service")
	flag.StringVar(&f.Backend.Port, "backend-port", "http", "name of backend port")
	flag.StringVar(&f.Admin.Address, "admin-addr", "127.0.0.1", "TCP address for the admin port")
	flag.IntVar(&f.Admin.Port, "admin-port", 6082, "TCP address for the admin port")

	flag.StringVar(&f.Varnish.SecretFile, "varnish-secret-file", "/etc/varnish/secret", "Varnish secret file")
	flag.StringVar(&f.Varnish.Storage, "varnish-storage", "file,/tmp/varnish-data,1G", "varnish storage config")
	flag.StringVar(&f.Varnish.VCLTemplate, "varnish-vcl-template", "/etc/varnish/default.vcl.tmpl", "VCL template file")

	flag.Parse()
}
