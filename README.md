# Varnish on Kubernetes

[![Docker Repository on Quay](https://quay.io/repository/spaces/kube-httpcache/status "Docker Repository on Quay")](https://quay.io/repository/spaces/kube-httpcache)

This repository contains a controller that allows you to operate a [Varnish cache](https://varnish-cache.org/) on Kubernetes.

## How it works

This controller is not intended to be a replacement of a regular [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress/). Instead, it is intended to be used between your regular Ingress controller and your application's service.

```
+---------+      +---------+      +-------------+
| Ingress |----->| Varnish |----->| Application |
+---------+      +---------+      +-------------+
```

The Varnish controller needs the following prerequisites to run:

- A [Go-template](https://golang.org/pkg/text/template/) that will be used to generate a [VCL](https://varnish-cache.org/docs/trunk/users-guide/vcl.html) configuration file
- A [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) that will be used as backend for the Varnish controller
- If RBAC is enabled in your cluster, you'll need a ServiceAccount with a role that grants `WATCH` access to the `endpoints` resource in the respective namespace

After starting, the Varnish controller will watch the configured backend service's endpoints; on startup and whenever these change, it will use the supplied VCL template to generate a new Varnish configuration and load this configuration at runtime.

The controller does not ship with any preconfigured configuration; the upstream connection and advanced features like load balancing are possible, but need to be configured in the VCL template supplied by you.

## Getting started

### Create a VCL template

**SUBJECT TO CHANGE**

Start by creating a `ConfigMap` that contains a VCL template:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vcl-template
data:
  default.vcl.tmpl: |
    vcl 4.0;

    import std;
    import directors;

    {{ range .Backends }}
    backend be-{{ .Name }} {
        .host = "{{ .Host }}";
        .port = "{{ .Port }}";
    }
    {{- end }}

    backend be-primary {
        .host = "{{ .PrimaryBackend.Host }}";
        .port = "{{ .PrimaryBackend.Port }}";
    }

    acl purgers {
        "127.0.0.1";
        "localhost";
        "::1";
    {{- range .Backends }}
        "{{ .Host }}";
    {{- end }}
    }

    sub vcl_init {
        new lb = directors.round_robin();

        {{ range .Backends -}}
        lb.add_backend(be-{{ .Name }});
        {{ end }}
    }

    # ...
```

### Create a Secret

Create a `Secret` object that contains the secret for the Varnish administration port:

```
$ kubectl create secret generic varnish-secret --from-literal=secret=$(head -c32 /dev/urandom  | base64)
```

### Deploy Varnish

Create a `Deployment` for the Varnish controller:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cache
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: cache
        image: quay.io/spaces/kube-httpcache:stable
        imagePullPolicy: Always
        args:
        - -admin-addr=0.0.0.0
        - -admin-port=6083
        - -varnish-secret-file=/etc/varnish/secret/secret
        - -varnish-vcl-template=/etc/varnish/tmpl/default.vcl.tmpl
        - -varnish-storage=malloc,128M
        volumeMounts:
        - name: template
          mountPath: /etc/varnish/tmpl
        - name: secret
          mountPath: /etc/varnish/secret
      volumes:
      - name: template
        configMap:
          name: vcl-template
      - name: secret
        secret:
          secretName: varnish-secret
```
