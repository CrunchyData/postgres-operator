/*
 Copyright 2017 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package util

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"io"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
)

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
}

//execs the cmd
func Exec(config *rest.Config, namespace, podname, containername string, cmd []string) error {

	wrappedRoundTripper, err := roundTripperFromConfig(config)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	req, err := requestFromConfig(config, podname, containername, namespace, cmd)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// Send the request and let the callback do its work
	_, err = wrappedRoundTripper.RoundTrip(req)

	if err != nil {
		log.Error(err.Error())
		return err
	}

	return err

}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer conn.Close()
	}
	return resp, d.Do(conn, resp, err)
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

func requestFromConfig(config *rest.Config, pod string, container string, namespace string, cmd []string) (*http.Request, error) {

	log.Info("config.Host is " + config.Host)
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
	params := url.Values{}
	for _, v := range cmd {
		params.Add("command", v)
	}
	params.Add("container", container)
	params.Add("stderr", "true")
	params.Add("stdout", "true")

	u.RawQuery = params.Encode()

	log.Info(u.String())

	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}

	return req, nil
}

func WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error) error {

	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			log.Error(err.Error())
			return err
		}
		log.Error(err.Error())
		return err
	}

	txt := ""
	for {
		_, body, err := ws.ReadMessage()
		if err != nil {
			log.Info(txt)
			if err == io.EOF {
				return nil
			}
			return err
		}
		txt = txt + string(body)
	}
}
