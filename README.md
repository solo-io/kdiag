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

# scaffold with
https://github.com/kubernetes/sample-cli-plugin/blob/master/pkg/cmd/ns.go