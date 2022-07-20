## diag shell

start a debug shell to the pod with an ephemeral container

```
diag shell [flags]
```

### Examples

```

	Start a shell to a pod. Unlike regular "kubectl exec" this command works even in distroless and
	scratch containers. It does so by using standalone busybox ash binary as the shell. This means
	that the shell is more limited, but works on any container as long as "/proc" is mounted.
	Note that this command needs linux kernel >= 5.3 to work. though this requirement may be relaxed
	in the future if needed.

	Optionally, you can start a shell to our ephemeral container (with --debug-shell flag). it has various debugging tools. 
	this is essentially a shortcut to "kubectl debug" with our image, and mainly useful for development purposes.

	For example:

	kdiag -l app=productpage -n bookinfo -t istio-proxy shell

	Start a shell targeting the istio-proxy container in the productpage pod.

	Note: a container is only created once, and may have been created from the previous commands. so specifying
	a different target the second time will have no effect.

```

### Options

```
  -h, --help                 help for shell
  -l, --labels string        select a pod by label. an arbitrary pod will be selected, with preference to newer pods
      --pod string           podname to diagnose
      --pull-policy string   image pull policy for the ephemeral container. defaults to IfNotPresent (default "IfNotPresent")
  -t, --target string        target container to diagnose, defaults to first container in pod
```

### Options inherited from parent commands

```
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "$HOME/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --dbg-image string               default dbg container image (default "ghcr.io/solo-io/kdiag:dev")
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

### SEE ALSO

* [diag](diag.md)	 - 

