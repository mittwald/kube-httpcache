# Varnish on Kubernetes

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/mittwald/kube-httpcache/Test)

This repository contains a controller that allows you to operate a [Varnish cache](https://varnish-cache.org/) on Kubernetes.

---
:warning: **COMPATIBILITY NOTICE**: As of version v0.3, the image tag name of this project was renamed from `quay.io/spaces/kube-httpcache` to `quay.io/mittwald/kube-httpcache`. The old image will remain available (for the time being), but only the new image name will receive any updates. **Please remember to adjust the image name when upgrading**.

---

## Table of Contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [How it works](#how-it-works)
- [High-Availability mode](#high-availability-mode)
- [Getting started](#getting-started)
  - [Create a VCL template](#create-a-vcl-template)
  - [Create a Secret](#create-a-secret)
  - [[Optional] Configure RBAC roles](#optional-configure-rbac-roles)
  - [Deploy Varnish](#deploy-varnish)
- [Detailed how-tos](#detailed-how-tos)
  - [Using built in signaller component](#using-built-in-signaller-component)
  - [Proxying to external services](#proxying-to-external-services)
- [Helm Chart installation](#helm-chart-installation)
- [Developer notes](#developer-notes)
  - [Build the Docker image locally](#build-the-docker-image-locally)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## How it works

This controller is not intended to be a replacement of a regular [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress/). Instead, it is intended to be used between your regular Ingress controller and your application's service.

```
┌─────────┐      ┌─────────┐      ┌─────────────┐
│ Ingress ├─────▶│ Varnish ├─────▶│ Application │
└─────────┘      └─────────┘      └─────────────┘
```

The Varnish controller needs the following prerequisites to run:

- A [Go-template](https://golang.org/pkg/text/template/) that will be used to generate a [VCL](https://varnish-cache.org/docs/trunk/users-guide/vcl.html) configuration file
- An application [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) that will be used as backend for the Varnish controller
- A Varnish [Kubernetes service](https://kubernetes.io/docs/concepts/services-networking/service/) that will be used as frontend for the Varnish controller
- If RBAC is enabled in your cluster, you'll need a ServiceAccount with a role that grants `WATCH` access to the `endpoints` resource in the respective namespace

After starting, the Varnish controller will watch the configured Varnish service's endpoints and application service's endpoints; on startup and whenever these change, it will use the supplied VCL template to generate a new Varnish configuration and load this configuration at runtime.

The controller does not ship with any preconfigured configuration; the upstream connection and advanced features like load balancing are possible, but need to be configured in the VCL template supplied by you.

## High-Availability mode

It can run in high avalability mode using multiple Varnish and application pods.

```
             ┌─────────┐
             │ Ingress │
             └────┬────┘
                  |
             ┌────┴────┐
             │ Service │
             └───┬┬────┘
             ┌───┘└───┐
┌────────────┴──┐  ┌──┴────────────┐
│   Varnish 1   ├──┤   Varnish 2   │
│  Signaller 1  ├──┤  Signaller 2  │
└─────────┬┬────┘  └────┬┬─────────┘
          │└─────┌──────┘│
          │┌─────┘└─────┐│
┌─────────┴┴────┐  ┌────┴┴─────────┐
│ Application 1 │  | Application 2 │
└───────────────┘  └───────────────┘
```

The Signaller component supports broadcasting PURGE and BAN requests to all Varnish nodes.

## Getting started

### Create a VCL template

<hr>

:warning: **NOTE**: The current implementation (supplying a VCL template as `ConfigMap`) may still be subject to change. Future implementations might for example use a Kubernetes Custom Resource for the entire configuration set.

<hr>

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

    // ".Frontends" is a slice that contains all known Varnish instances
    // (as selected by the service specified by -frontend-service).
    // The backend name needs to be the Pod name, since this value is compared
    // to the server identity ("server.identity" [1]) later.
    //
    //   [1]: https://varnish-cache.org/docs/6.4/reference/vcl.html#local-server-remote-and-client
    {{ range .Frontends }}
    backend {{ .Name }} {
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

Environment variables can be used from the template. `{{ .Env.ENVVAR }}` is replaced with the
environment variable value. This can be used to set for example the Host-header for the external
service.

### Create a Secret

Create a `Secret` object that contains the secret for the Varnish administration port:

```
$ kubectl create secret generic varnish-secret --from-literal=secret=$(head -c32 /dev/urandom  | base64)
```

### [Optional] Configure RBAC roles

If RBAC is enabled in your cluster, you will need to create a `ServiceAccount` with a respective `Role`.

```
$ kubectl create serviceaccount kube-httpcache
$ kubectl apply -f https://raw.githubusercontent.com/mittwald/kube-httpcache/master/deploy/kubernetes/rbac.yaml
$ kubectl create rolebinding kube-httpcache --clusterrole=kube-httpcache --serviceaccount=kube-httpcache
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
      selector:
        matchLabels:
          app: cache
      template:
        metadata:
          labels:
            app: cache
        spec:
          containers:
          - name: cache
            image: quay.io/mittwald/kube-httpcache:stable
            imagePullPolicy: Always
            args:
            - -admin-addr=0.0.0.0
            - -admin-port=6083
            - -signaller-enable
            - -signaller-port=8090
            - -frontend-watch
            - -frontend-namespace=$(NAMESPACE)
            - -frontend-service=frontend-service
            - -frontend-port=8080
            - -backend-watch
            - -backend-namespace=$(NAMESPACE)
            - -backend-service=backend-service
            - -varnish-secret-file=/etc/varnish/k8s-secret/secret
            - -varnish-vcl-template=/etc/varnish/tmpl/default.vcl.tmpl
            - -varnish-storage=malloc,128M
            env:
            - name: NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            volumeMounts:
            - name: template
              mountPath: /etc/varnish/tmpl
            - name: secret
              mountPath: /etc/varnish/k8s-secret
            ports:
            - containerPort: 8080
              name: http
            - containerPort: 8090
              name: signaller
          serviceAccountName: kube-httpcache  # when using RBAC
          restartPolicy: Always
          volumes:
          - name: template
            configMap:
              name: vcl-template
          - name: secret
            secret:
              secretName: varnish-secret
    ```

    **NOTE**: Using a `StatefulSet` is particularly important when using a stateful, self-routed Varnish cluster. Otherwise, you could also use a `Deployment` resource, instead.

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
        targetPort: http
      - name: "signaller"
        port: 8090
        targetPort: signaller
      selector:
        app: cache
    ```

3. Create an `Ingress` to forward requests to cache service. Typically, you should only need an Ingress for the Services `http` port, and not for the `signaller` port (if for some reason you do, make sure to implement proper access controls)

## Detailed how-tos

### Using built in signaller component

The signaller component is responsible for broadcasting HTTP requests to all nodes of a Varnish cluster. This is useful in HA cluster setups, when `BAN` or `PURGE` requests should be broadcast across the entire cluster.

To broadcast a `BAN` or `PURGE` request to all Varnish endpoints, run one of the following commands, respectively:

    $ curl -H "X-Url: /path" -X BAN http://cache-service:8090
    $ curl -H "X-Host: www.example.com" -X PURGE http://cache-service:8090/path

When running from outside the cluster, you can use `kubectl port-forward` to forward the signaller port to your local machine (and then send your requests to `http://localhost:8090`):

    $ kubectl port-forward service/cache-service 8090:8090

**NOTE:** Specific headers for `PURGE`/`BAN` requests depend on your Varnish configuration. E.g. `X-Host` header is set for convenience, because signaller is listening on other URL than Varnish. However, you need to support such headers in your VCL.

```vcl
sub vcl_recv {
  # ...

  # Purge logic
  if (req.method == "PURGE") {
    if (client.ip !~ purgers) {
      return (synth(403, "Not allowed."));
    }
    if (req.http.X-Host) {
      set req.http.host = req.http.X-Host;
    }
    return (purge);
  }

  # Ban logic
  if (req.method == "BAN") {
    if (client.ip !~ purgers) {
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

### Proxying to external services

<hr>

**NOTE**: Native support for `ExternalName` services is a requested feature. Have a look at [#39](https://github.com/mittwald/kube-httpcache/issues/39) if you're willing to help out.

<hr>

In some cases, you might want to cache content from a cluster-external resource. In this case, create a new Kubernetes service of type `ExternalName` for your backend:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
  namespace: default
spec:
  type: ExternalName
  externalName: external-service.example
```

In your VCL template, you can then simply use this service as static backend (since there are no dynamic endpoints, you do not need to iterate over `.Backends` in your VCL template):

```yaml
kind: ConfigMap
apiVersion: v1
metadata: # [...]
data:
  default.vcl.tmpl: |
    vcl 4.0;

    {{ range .Frontends }}
    backend {{ .Name }} {
        .host = "{{ .Host }}";
        .port = "{{ .Port }}";
    }
    {{- end }}

    backend backend {
        .host = "external-service.svc";
    }

    // ...
```

When starting kube-httpcache, remember to set the `--backend-watch=false` flag to disable watching the (non-existent) backend endpoints.

## Helm Chart installation

You can use the [Helm chart](chart/) to rollout an instance of kube-httpcache:

```
$ helm repo add mittwald https://helm.mittwald.de
$ helm install -f your-values.yaml kube-httpcache mittwald/kube-httpcache
```

For possible values, have a look at the comments in the provided [`values.yaml` file](./chart/values.yaml). Take special note that you'll most likely have to overwrite the `vclTemplate` value with your own VCL configuration file.

Ensure your defined backend services have a port named `http`:

```
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  type: ClusterIP
```

An ingress points to the kube-httpcache service which cached
your backend service:

```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
spec:
  rules:
  - host: www.example.com
    http:
      paths:
      - backend:
          service:
            name: kube-httpcache
            port:
              number: 80
        path: /
        pathType: Prefix
```

Look at the `vclTemplate` property in [chart/values.yaml](chart/values.yaml) to define
your own Varnish cluster rules or load with `extraVolume` an extra file
as initContainer if your ruleset is really big.

## Developer notes

### Build the Docker image locally

A Dockerfile for building the container image yourself is located in `build/package/docker`. Invoke `docker build` as follows:

```
$ docker build -t $IMAGE_NAME -f build/package/docker/Dockerfile .
```
