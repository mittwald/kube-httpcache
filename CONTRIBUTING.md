# Contribution guide

## Deployment on a local cluster

This guide explains how to build the kube-httpcache Docker image locally and test it in a local KinD[^1] cluster.

1. Build image and load into kind:

    ```
    $ docker build -t quay.io/mittwald/kube-httpcache:dev -f build/packages/docker/Dockerfile .
    $ kind load docker-image quay.io/mittwald/kube-httpcache:dev
    ```

2. Deploy an example backend workload:

    ```
    $ kubectl apply -f examples/test.yaml
    ```

3. Deploy Helm chart with example configuration:

    ```
    $ helm upgrade --install -f ./test/test-values.yaml kube-httpcache ./chart
    ```

4. Port-forward to the cache:

    ```
    $ kubectl port-forward svc/kube-httpcache 8080:80
    ```

[^1]: https://kind.sigs.k8s.io
