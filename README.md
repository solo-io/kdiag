# Various Diagnostics and Debug Tooling.

This plugin contains a set of tools to make it easier to develop service in kubernetes. Especially servers / control planes.

Note:
- Most of the tools here (except for logs) require kubernetes 1.23+.
- This software is beta quality. It seems to work, but there are definitely some bugs lurking around.

To install, add kubectl-diag to your PATH; krew install coming soon.

# Examples

Reverse port forward - redirect traffic pod's port 8080 to local port 8089

```sh
kubectl diag redir --pod mypod 8080:8089
```

Reverse port forward - redirect all the ports the pod listens on, to localhost.

```sh
kubectl diag redir --pod mypod
```

Reverse port forward - redirect outgoing traffic from the port 8080 on the pod pod to local port 8080 (in outgoing mode, ports must be specified).

```sh
kubectl diag redir --pod mypod 8080 --outgoing
```

Start a shell with debug tools:

```sh
kubectl diag shell --pod mypod
```
Note that the shell shares the pid namespace with the first container in the pod (can be changed using `-t` flag). This means that you can do `cd /proc/1/root` to access the other container's file system.

# Recipes

## Local Istio Debug

To redirect a sidecar to your istio running on your laptop, start your local pilot discovery, and then:

```sh
kubectl diag -l app=productpage -n bookinfo redirect --outgoing 15010 15012 15014
```

## Get root on a non root container

For example, get a root shell in the istio-proxy container:

```sh
kubectl diag shell -l app=productpage -t istio-proxy
nsenter -t 1 -a /bin/bash
```

# How it works?

See the [dev guide](DEVELOPER_GUIDE.md).