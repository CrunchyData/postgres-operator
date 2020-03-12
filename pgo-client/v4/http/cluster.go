/*
Copyright 2020 Crunchy Data Solutions, Inc.
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

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
	"github.com/crunchydata/postgres-operator/config"
)

// CreateCluster creates a cluster described by the request.
func (h *httpAPI) CreateCluster(ctx context.Context, r msgs.CreateClusterRequest) (msgs.CreateClusterResponse, error) {
	u := h.client.URL("clusters", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreateClusterResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreateClusterResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreateClusterResponse{}, err
	}

	var response msgs.CreateClusterResponse
	return response, json.Unmarshal(body, &response)
}

// DeleteCLuster deletes the cluster described by the request.
func (h *httpAPI) DeleteCluster(ctx context.Context, r msgs.DeleteClusterRequest) (msgs.DeleteClusterResponse, error) {
	u := h.client.URL("clustersdelete", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.DeleteClusterResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.DeleteClusterResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.DeleteClusterResponse{}, err
	}

	var response msgs.DeleteClusterResponse
	return response, json.Unmarshal(body, &response)
}

// ShowCluster retrieves information about the cluster that is described by the
// request.
func (h *httpAPI) ShowCluster(ctx context.Context, r msgs.ShowClusterRequest) (msgs.ShowClusterResponse, error) {
	u := h.client.URL("showclusters", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.ShowClusterResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowClusterResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowClusterResponse{}, err
	}

	var response msgs.ShowClusterResponse
	return response, json.Unmarshal(body, &response)
}

// UpdateCluster updates the cluster described by the request.
func (h *httpAPI) UpdateCluster(ctx context.Context, r msgs.UpdateClusterRequest) (msgs.UpdateClusterResponse, error) {
	u := h.client.URL("clustersupdate", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.UpdateClusterResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.UpdateClusterResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.UpdateClusterResponse{}, err
	}

	var response msgs.UpdateClusterResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) Clone(ctx context.Context, r msgs.CloneRequest) (msgs.CloneResponse, error) {
	u := h.client.URL("clone", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CloneResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CloneResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CloneResponse{}, err
	}

	var response msgs.CloneResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) Reload(ctx context.Context, r msgs.ReloadRequest) (msgs.ReloadResponse, error) {
	u := h.client.URL("reload", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.ReloadResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ReloadResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ReloadResponse{}, err
	}

	var response msgs.ReloadResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) Load(ctx context.Context, r msgs.LoadRequest) (msgs.LoadResponse, error) {
	u := h.client.URL("load", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.LoadResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.LoadResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.LoadResponse{}, err
	}

	var response msgs.LoadResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) ScaleCluster(ctx context.Context, r msgs.ClusterScaleRequest) (msgs.ClusterScaleResponse, error) {
	vars := map[string]string{"name": r.Name}
	params := map[string]string{
		"replica-count":    strconv.Itoa(r.ReplicaCount),
		"resources-config": r.ContainerResources,
		"storage-config":   r.StorageConfig,
		"node-label":       r.NodeLabel,
		"ccp-image-tag":    r.CCPImageTag,
		"service-type":     r.ServiceType,
		"namespace":        r.Namespace,
		"version:":         r.ClientVersion,
	}
	u := h.client.URL("clusters/scale/{name}", vars, params)

	req, err := h.createRequest(ctx, http.MethodGet, u.String(), r)
	if err != nil {
		return msgs.ClusterScaleResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ClusterScaleResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ClusterScaleResponse{}, err
	}

	var response msgs.ClusterScaleResponse
	return response, json.Unmarshal(body, response)
}

func (h *httpAPI) ScaleDownCluster(ctx context.Context, r msgs.ScaleDownRequest) (msgs.ScaleDownResponse, error) {
	vars := map[string]string{"name": r.Name}
	params := map[string]string{
		config.LABEL_REPLICA_NAME: r.ScaleDownTarget,
		config.LABEL_DELETE_DATA:  strconv.FormatBool(r.DeleteData),
		"namespace":               r.Namespace,
		"version":                 r.ClientVersion,
	}

	u := h.client.URL("scaledown/{name}", vars, params)

	req, err := h.createRequest(ctx, http.MethodGet, u.String(), r)
	if err != nil {
		return msgs.ScaleDownResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ScaleDownResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ScaleDownResponse{}, err
	}

	var response msgs.ScaleDownResponse
	return response, json.Unmarshal(body, &response)
}

func (h *httpAPI) ScaleQuery(ctx context.Context, r msgs.ScaleQueryRequest) (msgs.ScaleQueryResponse, error) {
	vars := map[string]string{"name": r.Name}
	params := map[string]string{
		"namespace": r.Namespace,
		"version":   r.ClientVersion,
	}

	u := h.client.URL("scale/{name}", vars, params)

	req, err := h.createRequest(ctx, http.MethodGet, u.String(), r)
	if err != nil {
		return msgs.ScaleQueryResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ScaleQueryResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ScaleQueryResponse{}, err
	}

	var response msgs.ScaleQueryResponse
	return response, json.Unmarshal(body, &response)
}
