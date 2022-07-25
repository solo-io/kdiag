# Diagnostics and Debug Tooling.

This plugin contains a set of tools to make it easier to develop multi pod systems in kubernetes. Especially servers / control planes.

Note:
- Most of the tools here (except for logs) require kubernetes 1.23+. Shell command requires kernel 5.3+.
- This software is beta quality. It seems to work, but there are definitely some bugs lurking around.

To install, add kubectl-diag to your PATH.

# How is this useful?

It allows you to get the following:
- Shell access to scratch containers: As often deployments are hardened.
    This works similar to `kubectl exec` (giving you a shell inside a container in a running pod), but works any kind of container (scratch, distroless, ...)
- Log from multiple pods at the same time: When debugging request flows through a service mesh, with multiple pods and sidecars involved, it can be convenient to see logs
  from all containers at the same time.
- Reverse port forward - forward ports from the pod to your machine: Use this to test local changes, without redeploying. redirect incoming traffic to the pod to your laptop. This can be used to rapidly test changes to istiod for example. You can also redirect outgoing traffic (e.g. only point one sidecar to your local control plane).

# Examples

## Local Istio Debug

To redirect a sidecar to your istio running on your laptop, start your local pilot discovery, and then:

```sh
kubectl diag -l app=productpage -n bookinfo redirect --outgoing 15010 15012 15014
```

Redirect all ports that istiod listens on locally:

```sh
kubectl diag -l app=istiod -n bookinfo redirect
```

You can also specify specific ports:

```sh
kubectl diag -l app=istiod -n bookinfo redirect 15012:15012
```

## Get a root shell in a container

For example, get a root [`ash`](https://www.busybox.net/) shell in the istio-proxy container:
Note that this works even on scratch and distroless containers!

```sh
kubectl diag shell -l app=productpage -t istio-proxy
```

## Log multiple pods at once

When debugging a a request going through the cluster, it can be useful to see the logs of multiple pods as they request
flow through the cluster.

See your request traverses the mesh:
For example, this will show the logs of all the `istio-proxy` container in pods in the `bookinfo` namespace.
It will execute the curl command and then terminate.

```sh
kubectl diag logs -n bookinfo --all -c istio-proxy -- curl http://foo.bar.com
```


# How it works?

See the [dev guide](DEVELOPER_GUIDE.md).