# Varnish on Kubernetes

[![Build Status](https://travis-ci.org/mittwald/kube-httpcache.svg?branch=master)](https://travis-ci.org/mittwald/kube-httpcache)
[![Docker Repository on Quay](https://quay.io/repository/spaces/kube-httpcache/status "Docker Repository on Quay")](https://quay.io/repository/spaces/kube-httpcache)

This repository contains a controller that allows you to operate a [Varnish cache](https://varnish-cache.org/) on Kubernetes.

## How it works

This controller is not intended to be a replacement of a regular [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress/). Instead, it is intended to be used between your regular Ingress controller and your application's service.

```
+---------+      +---------+      +-------------+
| Ingress |----->| Varnish |----->| Application |
+---------+      +---------+      +-------------+
```

It can run in high avalability mode using multiple Varnish and application pods.

```
             +---------+
             | Ingress |
             +---------+
                  |
             +---------+
             | Service |
             +---------+
               /     \
+---------------+  +---------------+
|   Varnish 1   |--|   Varnish 2   |
| Broadcaster 1 |--| Broadcaster 2 |
+---------------+  +---------------+
          |      \/      |
          |      /\      |
+---------------+  +---------------+
| Application 1 |  | Application 2 |
+---------------+  +---------------+
```

It supports broadcasting PURGE and BAN requests to all Varnish nodes.


The Varnish controller needs the following prerequisites to run:

- A [Go-template](https://golang.org/pkg/text/template/) that will be used to generate a [VCL](https://varnish-cache.org/docs/trunk/users-guide/vcl.html) configuration file
- A [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) that will be used to determine Varnish (frontend) and application (backend) pods
- If RBAC is enabled in your cluster, you'll need a ServiceAccount with a role that grants `WATCH` access to the `endpoints` resource in the respective namespace

After starting, the Varnish controller will watch the configured Varnish service's endpoints and application service's endpoints; on startup and whenever these change, it will use the supplied VCL template to generate a new Varnish configuration and load this configuration at runtime.

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

    {{ range .Frontends }}
    backend fe-{{ .Name }} {
        .host = "{{ .Host }}";
        .port = "{{ .Port }}";
    }
    {{- end }}

    backend fe-primary {
        .host = "{{ .PrimaryFrontend.Host }}";
        .port = "{{ .PrimaryFrontend.Port }}";
    }

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
        {{- range .Frontends }}
        "{{ .Host }}";
        {{- end }}
        {{- range .Backends }}
        "{{ .Host }}";
        {{- end }}
    }

    sub vcl_init {
        new cluster = directors.hash();

        {{ range .Frontends -}}
        cluster.add_backend({{ .Name }}, 1);
        {{ end }}

        new lb = directors.round_robin();

        {{ range .Backends -}}
        lb.add_backend(be-{{ .Name }});
        {{ end }}
    }

    sub vcl_recv
    {
        # Set backend hint for non cachable objects.
        set req.backend_hint = lb.backend();

        # ...

        # Routing logic. Pass a request to an appropriate Varnish node.
        # See https://info.varnish-software.com/blog/creating-self-routing-varnish-cluster for more info.
        unset req.http.x-cache;
        set req.backend_hint = cluster.backend(req.url);
        set req.http.x-shard = req.backend_hint;
        if (req.http.x-shard != server.identity) {
            return(pass);
        }
        set req.backend_hint = lb.backend();

        # ...

        return(hash);
    }

    # ...
```

### Create a Secret

Create a `Secret` object that contains the secret for the Varnish administration port:

```
$ kubectl create secret generic varnish-secret --from-literal=secret=$(head -c32 /dev/urandom  | base64)
```

### [Optional] Configure RBAC roles

If RBAC is enabled in your cluster, you will need to create a `ServiceAccount` with a respective `Role`.

```
$ kubectl create serviceaccount kube-httpcache
$ kubectl apply -f https://raw.githubusercontent.com/mittwald/kube-httpcache/master/deploy/rbac.yaml
$ kubectl create rolebinding kube-httpcache --role=kube-httpcache --serviceaccount=kube-httpcache
```

### Deploy Varnish

1. Create a `StatefulSet` for the Varnish controller:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cache-statefulset
  labels:
    app: cache
spec:
  serviceName: cache-service
  replicas: 2
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: cache
    spec:
      containers:
      - name: cache
        image: quay.io/spaces/kube-httpcache:stable
        imagePullPolicy: Always
        restartPolicy: Always
        args:
        - -admin-addr=0.0.0.0
        - -admin-port=6083
        - -broadcaster-enable=true
        - -broadcaster-port=8090
        - -frontend-watch=true
        - -frontend-namespace=$(NAMESPACE)
        - -frontend-service=frontend-service
        - -backend-watch=true
        - -backend-namespace=$(NAMESPACE)
        - -backend-service=backend-service
        - -varnish-secret-file=/etc/varnish/k8s-secret/secret
        - -varnish-vcl-template=/etc/varnish/tmpl/default.vcl.tmpl
        - -varnish-storage=malloc,128M
        volumeMounts:
        - name: template
          mountPath: /etc/varnish/tmpl
        - name: secret
          mountPath: /etc/varnish/k8s-secret
      serviceAccountName: kube-httpcache  # when using RBAC
      env:
      - name: NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
      volumes:
      - name: template
        configMap:
          name: vcl-template
      - name: secret
        secret:
          secretName: varnish-secret
```

2. Create a service for the Varnish controller:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: cache-service
  labels:
    app: cache
spec:
  ports:
  - name: "http"
    port: 80
    targetPort: 80
  - name: "broadcaster"
    port: 8090
    targetPort: 8090
  selector:
    app: cache
```

3. Create an ingress to forward requests to cache service. You may end up with two URLs: http://www.example.com, http://broadcaster.example.com. A url for broadcaster is optional, if you choose to have it, make sure to limit access to it.

## Using built in broadcaster

To broadcast a BAN request to all Varnish endpoints, run:

```
curl -H "X-Url: /path" -X BAN http://cache-service:8090
```

or

```
curl -H "X-Url: /path" -X BAN http://broadcaster.example.com
```

To broadcast a PURGE request to all Varnish endpoints, run:

```
curl -H "X-Host: www.example.com" -X PURGE http://cache-service:8090/path
```

or

```
curl -H "X-Host: www.example.com" -X PURGE http://broadcaster.example.com/path
```

Specific headers for PURGE/BAN requests depend on your Varnish configuration. E.g. X-Host header is set for convinience, because broadcaster is listening on other URL than Varnish. However, you need to suport such headers in your vcl.

```vcl
sub vcl_recv
{
  # ...

  # Purge logic
  if ( req.method == "PURGE" ) {
    if ( client.ip !~ privileged ) {
      return (synth(403, "Not allowed."));
    }
    if (req.http.X-Host) {
      set req.http.host = req.http.X-Host;
    }
    return (purge);
  }

  # Ban logic
  if ( req.method == "BAN" ) {
    if ( client.ip !~ privileged ) {
      return (synth(403, "Not allowed."));
    }
    if (req.http.Cache-Tags) {
      ban("obj.http.Cache-Tags ~ " + req.http.Cache-Tags);
      return (synth(200, "Ban added " + req.http.host));
    }
    if (req.http.X-Url) {
      ban("obj.http.X-Url == " + req.http.X-Url);
      return (synth(200, "Ban added " + req.http.host));
    }
    return (synth(403, "Cache-Tags or X-Url header missing."));
  }

  # ...
}
```
