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

	msgs "github.com/crunchydata/postgres-operator/apiservermsgs"
)

// CreateBackrestBackup creates a PG BackRest based backup for a cluster.
func (h *httpAPI) CreateBackrestBackup(ctx context.Context, r msgs.CreateBackrestBackupRequest) (msgs.CreateBackrestBackupResponse, error) {
	u := h.client.URL("backrestbackup", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)

	if err != nil {
		return msgs.CreateBackrestBackupResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreateBackrestBackupResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreateBackrestBackupResponse{}, err
	}

	var response msgs.CreateBackrestBackupResponse
	return response, json.Unmarshal(body, &response)
}

// ShowBackrest retrieves information about a PG BackRest backup for a cluster.
func (h *httpAPI) ShowBackrest(ctx context.Context, r msgs.ShowBackrestRequest) (msgs.ShowBackrestResponse, error) {
	vars := map[string]string{"name": r.Name}
	params := map[string]string{
		"version":   r.ClientVersion,
		"selector":  r.Selector,
		"namespace": r.Namespace,
	}

	u := h.client.URL("backrest/{name}", vars, params)

	req, err := h.createRequest(ctx, http.MethodGet, u.String(), r)

	if err != nil {
		return msgs.ShowBackrestResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowBackrestResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowBackrestResponse{}, err
	}

	var response msgs.ShowBackrestResponse
	return response, json.Unmarshal(body, &response)

}

// RestorebackrestBackup restores a cluster from a PG BackRest backup.
func (h *httpAPI) RestoreBackrestBackup(ctx context.Context, r msgs.RestoreRequest) (msgs.RestoreResponse, error) {
	u := h.client.URL("restore", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.RestoreResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.RestoreResponse{}, err
	}

	if err = statusCheck(resp); err != nil {
		return msgs.RestoreResponse{}, err
	}

	var response msgs.RestoreResponse
	return response, json.Unmarshal(body, &response)
}

// CreatePGDumpBackup creates a PGDump backup for a cluster.
func (h *httpAPI) CreatePGDumpBackup(ctx context.Context, r msgs.CreatePGDumpRequest) (msgs.CreatePGDumpResponse, error) {
	u := h.client.URL("pgdumpbackup", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.CreatePGDumpResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.CreatePGDumpResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.CreatePGDumpResponse{}, err
	}

	var response msgs.CreatePGDumpResponse
	return response, json.Unmarshal(body, &response)
}

// ShowPGDumpBackup retrieves information about a PGDump backup for a cluster.
func (h *httpAPI) ShowPGDumpBackup(ctx context.Context, r msgs.ShowPGDumpRequest) (msgs.ShowBackupResponse, error) {
	vars := map[string]string{"arg": r.Arg}
	params := map[string]string{
		"version":   r.ClientVersion,
		"selector":  r.Selector,
		"namespace": r.Namespace,
	}

	u := h.client.URL("pgdump/{arg}", vars, params)

	req, err := h.createRequest(ctx, http.MethodGet, u.String(), r)
	if err != nil {
		return msgs.ShowBackupResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.ShowBackupResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.ShowBackupResponse{}, err
	}

	var response msgs.ShowBackupResponse
	return response, json.Unmarshal(body, &response)
}

// RestorePGDumpBackup restores a cluster from a PGDump backup.
func (h *httpAPI) RestorePGDumpBackup(ctx context.Context, r msgs.PgRestoreRequest) (msgs.RestoreResponse, error) {
	u := h.client.URL("pgdumprestore", nil, nil)

	req, err := h.createRequest(ctx, http.MethodPost, u.String(), r)
	if err != nil {
		return msgs.RestoreResponse{}, err
	}

	resp, body, err := h.client.Do(req)
	if err != nil {
		return msgs.RestoreResponse{}, err
	}

	if err := statusCheck(resp); err != nil {
		return msgs.RestoreResponse{}, err
	}

	var response msgs.RestoreResponse
	return response, json.Unmarshal(body, &response)
}
