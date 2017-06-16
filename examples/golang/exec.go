package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	//"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"net/http"
	"net/url"
	//"k8s.io/client-go/pkg/api/errors"

	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"

	"k8s.io/client-go/pkg/api/unversioned"
	//"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	var namespace = "default"
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	if clientset != nil {
	}

	var podName = "bone-replica-284779624-9f27c"

	var u *url.URL

	u, err = url.Parse(config.Host)
	if err != nil {
		fmt.Println("error url.Parse " + err.Error())
		return
	}

	u.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, podName)
	u.RawQuery = getCommandString("ls")

	fmt.Println(u.String())

	var tprconfig *rest.Config
	tprconfig = config
	configureClient(tprconfig)

	tprclient, err := rest.RESTClientFor(tprconfig)
	if err != nil {
		panic(err)
	}
	if tprclient != nil {
	}

	var restclient *http.Client
	restclient = tprclient.Client
	if restclient != nil {
	}

	// Create a round tripper with all necessary kubernetes security details
	wrappedRoundTripper, err := roundTripperFromConfig(config)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Create a request out of config and the query parameters
	var container = "database"
	var command = "ls"
	req, err := requestFromConfig(config, podName, container, namespace, command)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Send the request and let the callback do its work
	_, err = wrappedRoundTripper.RoundTrip(req)

	if err != nil {
		fmt.Println(err.Error())
	}

}

func getCommandString(cmd string) string {
	return "command=" + cmd +
		"&container=database" +
		"&stderr=true&stdout=true"
}

func configureClient(config *rest.Config) {
	groupversion := unversioned.GroupVersion{
		Group:   "k8s.io",
		Version: "v1",
	}

	config.GroupVersion = &groupversion
	//config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			scheme.AddKnownTypes(
				groupversion,
				&api.ListOptions{},
				&api.DeleteOptions{},
			)
			return nil
		})
	schemeBuilder.AddToScheme(api.Scheme)
}

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer conn.Close()
	}
	return resp, d.Do(conn, resp, err)
}

func WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error) error {

	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			return fmt.Errorf("Can't connect to console (%d): %s\n", resp.StatusCode, buf.String())
		}
		return fmt.Errorf("Can't connect to console: %s\n", err.Error())
	}

	txt := ""
	for {
		_, body, err := ws.ReadMessage()
		if err != nil {
			fmt.Println(txt)
			if err == io.EOF {
				return nil
			}
			return err
		}
		txt = txt + string(body)
	}
}
func roundTripperFromConfig(config *rest.Config) (http.RoundTripper, error) {

	// Configure TLS
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}

	// Configure the websocket dialer
	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
	}

	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	rt := &WebsocketRoundTripper{
		Do:     WebsocketCallback,
		Dialer: dialer,
	}

	// Make sure we inherit all relevant security headers
	return rest.HTTPWrappersForConfig(config, rt)
}

func requestFromConfig(config *rest.Config, pod string, container string, namespace string, cmd string) (*http.Request, error) {

	u, err := url.Parse(config.Host)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, fmt.Errorf("Malformed URL %s", u.String())
	}

	u.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, pod)
	if container != "" {
		u.RawQuery = "command=" + cmd +
			"&container=" + container +
			"&stderr=true&stdout=true"
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}

	return req, nil
}
