Various Diagnostics and Debug Tooling.

Set of tools to make it easier to develop service in kubernetes. Especially servers / control planes.

Note: most of the tools here (except for logs) require kubernetes 1.23+.

To install, add kubectl-diag to your PATH; krew install coming soon.

Examples:

Reverse port forward - redirect traffic pod's port 8080 to local port 8089

```sh
kubectl diag redir --pod mypod 8080:8089
```

Reverse port forward - redirect all the ports the pod listens on, to localhost.

```sh
kubectl diag redir --pod mypod
```

Reverse port forward - redirect outgoing traffic from the pod locally.

```sh
kubectl diag redir --pod mypod 8080:8080 --outgoing
```

Start a shell with debug tools:

```sh
kubectl diag shell --pod mypod
```
Note that the shell shares the pid namespace with the first container in the pod (can be changed using `-t` flag). This means that you can do `cd /proc/1/root` to access the other container's file system.

# Examples

## Local Istio Debug

To redirect a sidecar to your istio running on your laptop, start your local pilot discovery, and then:

```sh
kubectl diag -l app=productpage -n bookinfo redirect --outgoing 15010 15012 15014
```
# How it works?

To help set-up reverse redirects, we inject an EphemeralContainer to the pod. The container has a process (called manager) that exposes a grpc api.
This allows it to communicate with the command line.

When doing a reverse port forward, the follow happens:
- command line sends a request to the manager in the container.
- manager starts up a listener on a random port
- manager sets up iptable rules to capture the traffic to the listener it just opened.
- the manager starts another listener on another random port.
- when a connection arrive in the first listener, the manager sends a message to the commandline with the port of the second listener.
- the command line then starts a port forward to that second listener's port, and connects to the local port. and bridges these two connections
- the manager accepts the connection on the second listener from the command line, and bridges the two connections it has (this one, and the one from the first listener).
- that's it!


# Dev/Debug:

## Iterating locally with kind
```sh
make create-test-env
# or, after the first time:
make reload-test-env

go run . shell -l app=curl
```

## Iterating with a remote cluster

This shows how to query the debug container with grpcurl

```sh
# clean slate
kubectl delete pod -n istio-system -l app=istiod

# push updated image to a repo that you control, and the cluster can access
IMG=myrepo.example.com/kdiag:dev
make docker-build IMG=${IMG}
docker push ${IMG}

# run manage command, that just starts the ephemeral container
CONTAINER=$(go run . -l app=istiod -n istio-system --dbg-image ${IMG} --pull-policy=Always manage|cut -d' ' -f1)
# get the manager port form the logs
PORT=$(kubectl logs -n istio-system deploy/istiod -c ${CONTAINER}|head -1|rev|cut -d: -f1|rev)
# portforward to that port
kubectl port-forward -n istio-system deploy/istiod 8087:${PORT} &
# query it with grpc curl
grpcurl -plaintext localhost:8087 kdiag.solo.io.Manager.Ps
```
