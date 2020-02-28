package kubeapi

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Proxy struct {
	addr  string
	err   chan error
	stop  chan struct{}
	proxy *portforward.PortForwarder
}

func (p *Proxy) Close() error     { close(p.stop); return <-p.err }
func (p Proxy) LocalAddr() string { return p.addr }

// PodPortForward proxies TCP connections to a random local port to a port on a
// pod.
func (k *KubeAPI) PodPortForward(namespace, name, port string) (*Proxy, error) {
	// portforward.PortForwarder tries to listen on both IPv4 and IPv6 when
	// address is "localhost". That'd be great, but it doesn't indicate which
	// random port was assigned to which network. Use IPv4 (tcp4) loopback address
	// to avoid that ambiguity.
	const address = "127.0.0.1"

	request := k.Client.CoreV1().RESTClient().Post().
		Resource("pods").SubResource("portforward").
		Namespace(namespace).Name(name)

	tripper, upgrader, err := spdy.RoundTripperFor(k.Config)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: tripper}, "POST", request.URL())
	ready := make(chan struct{})

	p := Proxy{
		err:  make(chan error),
		stop: make(chan struct{}),
	}

	if p.proxy, err = portforward.NewOnAddresses(
		dialer, []string{address}, []string{":" + port},
		p.stop, ready, ioutil.Discard, ioutil.Discard,
	); err != nil {
		return nil, err
	}

	go func() { p.err <- p.proxy.ForwardPorts() }()

	select {
	case err = <-p.err:
		return nil, err
	case <-ready:
	}

	ports, _ := p.proxy.GetPorts()
	p.addr = net.JoinHostPort(address, fmt.Sprintf("%d", ports[0].Local))

	return &p, nil
}
