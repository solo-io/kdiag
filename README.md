Debug and performance tooling!


Examples:

start a shell with debug tools:
-> kubectl diag shell mypod

the reverse of port forward - redirect traffic from the pod locally
-> kubectl diag port-redirect mypod 8080:80

collect flame graph!
-> kubectl diag pprof -c container mypod

display useful debug info: process, who listens on what port, what pods are connected to here
-> kubectl diag stats mypod

start dlv session:
-> kubectl diag debug-go mypod

copy files:
-> kubectl diag cp mypod:foo .

forward all ports from all pods in the namespace / pod
-> kubectl diag auto-port-forward -n foo

```
# mkdir /hostfs; (cd /proc/1/root; mount --bind . /hostfs)
ln -s /proc/1/root /hostfs
```

k diag log-exec

k diag log-exec pod pod2 -- curl localhost:8090
k diag log-snapshot pod pod2 -- k exec pod2 curl localhost:8090



# scaffold with
https://github.com/kubernetes/sample-cli-plugin/blob/master/pkg/cmd/ns.go


https://gperftools.github.io/gperftools/cpuprofile.html

https://docs.google.com/presentation/d/10JmeisHsug-7XCB5Ym1HPYoeKLU_r9MOSNNPGWAteKg/preview?pru=AAABf-EPQ3o*Jin1U1-U9oS0IXI9FRHpFg#slide=id.g5218f0a929_0_0



# dev/debug:

```shell
kubectl delete pod -n istio-system -l app=istiod

IMG=gcr.io/solo-test-236622/kdiag:dev
make docker-build IMG=${IMG}
docker push ${IMG}

CONTAINER=$(go run . -l app=istiod -n istio-system --dbg-image ${IMG} --pull-policy=Always manage|cut -d' ' -f1)
PORT=$(kubectl logs -n istio-system deploy/istiod -c ${CONTAINER}|head -1|rev|cut -d: -f1|rev)
kubectl port-forward -n istio-system deploy/istiod 8087:${PORT} &

grpcurl -plaintext localhost:8087 kdiag.solo.io.Manager.Ps

kubectl logs -n istio-system deploy/istiod -c ${CONTAINER}


```