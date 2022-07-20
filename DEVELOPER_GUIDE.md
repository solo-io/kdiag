
# Dev/Debug

This project has 2 pieces:
- a so-called manager that runs in an ephemeral container in your desired pod
- a kubectl plugin that communicates with it. the plugin creates the ephemeral container in the pod if not already there.

The role of the manager is to do setup inside the pod. For example when setting up traffic redirects
the manager sets up the iptables rules and forwards traffic to the cli-plugin.

Communication between the cli plugin and the manager is via gRPC (see `api/kdiag` folder). To avoid
collisions with pod ports, the manager starts up on a random port, and prints it to logs. the cli
grabs the port from the pod's logs.


If you change the gRPC API, run `make generate`.
Most of the code is under the `pkg` folder. e2e test using kind are in `test/e2e`. to run the e2e tests:

```sh
make create-test-env
# or, after the first time:
make reload-test-env
go test ./...
```

For development, a container is built and loaded to `kind` using `Dockerfile`. For release `Dockerfile.release` is used.
There's a CI check to make sure these stay in sync (i.e. that the section after the "Install dependencies" comment is the same).


## How does redirection works?

To help set-up reverse redirects, we inject an EphemeralContainer to the pod. The container has a process (called manager) that exposes a grpc api.
This allows it to communicate with the command line.

When doing a reverse port forward, the follow happens:
- command line sends a request to the manager in the container.
- manager starts up a listener on a random port
- manager sets up iptable rules to capture the traffic to the listener it just opened.
- the manager starts another listener on another random port.
- when a connection arrive in the first listener, the manager sends a message to the command-line with the port of the second listener.
- the command line then starts a port forward to that second listener's port, and connects to the local port. and bridges these two connections
- the manager accepts the connection on the second listener from the command line, and bridges the two connections it has (this one, and the one from the first listener).
- that's it!

## How does the shell command work?

We have a prebuilt busybox standalone `ash` shell. standalone means that it executes commands internally
without needing the commands to be on the path.

We use a small [nsenter inspired utility](scratch-shell/enter.c) to inject the `ash` shell to your pods namespaces.
Due to the syscalls we use, this requires linux kernel version 5.3+.

# Iterating locally with kind
```sh
make create-test-env
# or, after the first time:
make reload-test-env

# start a debug shell for example
go run . shell -l app=curl

# log command for example:
go run . logs -l app=nginx curl "http://$(kubectl get node kind-control-plane -o jsonpath='{.status.addresses[0].address}'):$(kubectl get service nginx -o jsonpath='{.spec.ports[0].nodePort}')"
```

# Iterating with a remote cluster

This shows how to query the debug container with grpcurl

```sh
# clean slate
kubectl delete pod -n istio-system -l app=istiod

# push updated image to a repo that you control, and the cluster can access
IMG=myrepo.example.com/kdiag:dev
make docker-build IMG=${IMG}
docker push ${IMG}

# run manage command, that just starts the ephemeral container
# note the pull policy set to Always to make sure we use the latest pushed image
# here we run the `manage` command. it's pretty useless outside of development hence it is hidden.
# It prints the name of the created ephemeral container.
CONTAINER=$(go run . -l app=istiod -n istio-system --dbg-image ${IMG} --pull-policy=Always manage|cut -d' ' -f1)
# get the manager port form the logs
PORT=$(kubectl logs -n istio-system deploy/istiod -c ${CONTAINER}|head -1|rev|cut -d: -f1|rev)
# portforward to that port
kubectl port-forward -n istio-system deploy/istiod 8087:${PORT} &
# query it with grpc curl
grpcurl -plaintext localhost:8087 kdiag.solo.io.Manager.Ps
```

# Test krew bot

docker run --rm -v ${PWD}/.krew.yaml:/tmp/template-file.yaml rajatjindal/krew-release-bot:v0.0.43 krew-release-bot template --tag v0.0.5 --template-file /tmp/template-file.yaml
