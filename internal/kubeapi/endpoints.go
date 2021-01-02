package kubeapi

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

import (
	"context"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetEndpointRequest is used for the GetEndpoint function, which includes the
// current Kubernetes request context, as well as the namespace / endpoint name
// being requested
type GetEndpointRequest struct {
	Clientset kubernetes.Interface // Kubernetes Clientset that interfaces with the Kubernetes cluster
	Name      string               // Name of the endpoint that is being queried
	Namespace string               // Namespace the endpoint being queried resides in
}

// GetEndpointResponse contains the results from a successful request to the
// endpoint API, including the Kubernetes Endpoint as well as the original
// request data
type GetEndpointResponse struct {
	Endpoint  *v1.Endpoints // Kubernetes Endpoint object that specifics about the endpoint
	Name      string        // Name of the endpoint
	Namespace string        // Namespace that the endpoint is in
}

// GetEndpoint tries to find an individual endpoint in a namespace. Returns the
// endpoint object if it can be IsNotFound
// If no endpoint can be found, then an error is returned
func GetEndpoint(request *GetEndpointRequest) (*GetEndpointResponse, error) {
	ctx := context.TODO()
	log.Debugf("GetEndpointResponse Called: (%s,%s,%s)", request.Clientset, request.Name, request.Namespace)
	// set the endpoints interfaces that will be used to make the query
	endpointsInterface := request.Clientset.CoreV1().Endpoints(request.Namespace)
	// make the query to Kubernetes to see if the specific endpoint exists
	endpoint, err := endpointsInterface.Get(ctx, request.Name, metav1.GetOptions{})
	// return at this point if there is an error
	if err != nil {
		log.Errorf("GetEndpointResponse(%s,%s): Endpoint Not Found: %s",
			request.Name, request.Namespace, err.Error())
		return nil, err
	}
	// create a response and return
	response := &GetEndpointResponse{
		Endpoint:  endpoint,
		Name:      request.Name,
		Namespace: request.Namespace,
	}

	log.Debugf("GetEndpointResponse Response: (%s,%s,%s)",
		response.Namespace, response.Name, response.Endpoint)

	return response, nil
}
