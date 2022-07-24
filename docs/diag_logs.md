## diag logs

View logs from multiple containers

```
diag logs [flags]
```

### Examples

```

	The use case for this command is when you want to see the impact of an action over the container logs.
	As such, this command tails and follows the logs while a command is executed.
	for example, get all the logs from the "istio-proxy" container in the bookinfo namespace:
	while executing a curl command:

	kdiag logs -n bookinfo --all -c istio-proxy -- curl http://foo.bar.com

	You can also use the following syntax to get the logs from a specific container.
	
	This examples gets the logs from the "istio-proxy" container from all the pods with the app=productpage label

	kdiag logs -n bookinfo -l app=productpage:istio-proxy -- curl http://foo.bar.com

```

### Options

```
  -a, --all                       select all pods in the namespace
  -c, --container string          default container name to use for logs. defaults to first container in the pod
  -d, --drain-duration duration   duration to wait for logs after command exits (default 500ms)
  -h, --help                      help for logs
  -l, --labels stringArray        select a pods to watch logs by label. you can use k=v:containername to specify container name
      --no-color                  Disable color output
      --pod stringArray           podname to view logs of. you can use podname:containername to specify container name
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

