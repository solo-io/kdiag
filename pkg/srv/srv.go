package srv

func RedirectTraffic(podPort int, localPort int) error {
	// start listening socket on a random port

	// setup iptables redirect to that random port

	// every connection received, proxy to client via the grpc connection

	panic("not implemented") // TODO: Implement
}
