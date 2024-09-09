// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const defaultAPI = "https://api.crunchybridge.com"

var errAuthentication = errors.New("authentication failed")

type ClientInterface interface {
	ListClusters(ctx context.Context, apiKey, teamId string) ([]*ClusterApiResource, error)
	CreateCluster(ctx context.Context, apiKey string, clusterRequestPayload *PostClustersRequestPayload) (*ClusterApiResource, error)
	DeleteCluster(ctx context.Context, apiKey, id string) (*ClusterApiResource, bool, error)
	GetCluster(ctx context.Context, apiKey, id string) (*ClusterApiResource, error)
	GetClusterStatus(ctx context.Context, apiKey, id string) (*ClusterStatusApiResource, error)
	GetClusterUpgrade(ctx context.Context, apiKey, id string) (*ClusterUpgradeApiResource, error)
	UpgradeCluster(ctx context.Context, apiKey, id string, clusterRequestPayload *PostClustersUpgradeRequestPayload) (*ClusterUpgradeApiResource, error)
	UpgradeClusterHA(ctx context.Context, apiKey, id, action string) (*ClusterUpgradeApiResource, error)
	UpdateCluster(ctx context.Context, apiKey, id string, clusterRequestPayload *PatchClustersRequestPayload) (*ClusterApiResource, error)
	GetClusterRole(ctx context.Context, apiKey, clusterId, roleName string) (*ClusterRoleApiResource, error)
}

type Client struct {
	http.Client
	wait.Backoff

	BaseURL url.URL
	Version string
}

// BRIDGE API RESPONSE OBJECTS

// ClusterApiResource is used to hold cluster information received in Bridge API response.
type ClusterApiResource struct {
	ID                     string                       `json:"id,omitempty"`
	ClusterGroup           *ClusterGroupApiResource     `json:"cluster_group,omitempty"`
	PrimaryClusterID       string                       `json:"cluster_id,omitempty"`
	CPU                    int64                        `json:"cpu,omitempty"`
	CreatedAt              string                       `json:"created_at,omitempty"`
	DiskUsage              *ClusterDiskUsageApiResource `json:"disk_usage,omitempty"`
	Environment            string                       `json:"environment,omitempty"`
	Host                   string                       `json:"host,omitempty"`
	IsHA                   *bool                        `json:"is_ha,omitempty"`
	IsProtected            *bool                        `json:"is_protected,omitempty"`
	IsSuspended            *bool                        `json:"is_suspended,omitempty"`
	Keychain               string                       `json:"keychain_id,omitempty"`
	MaintenanceWindowStart int64                        `json:"maintenance_window_start,omitempty"`
	MajorVersion           int                          `json:"major_version,omitempty"`
	Memory                 float64                      `json:"memory,omitempty"`
	ClusterName            string                       `json:"name,omitempty"`
	Network                string                       `json:"network_id,omitempty"`
	Parent                 string                       `json:"parent_id,omitempty"`
	Plan                   string                       `json:"plan_id,omitempty"`
	PostgresVersion        intstr.IntOrString           `json:"postgres_version_id,omitempty"`
	Provider               string                       `json:"provider_id,omitempty"`
	Region                 string                       `json:"region_id,omitempty"`
	Replicas               []*ClusterApiResource        `json:"replicas,omitempty"`
	Storage                int64                        `json:"storage,omitempty"`
	Tailscale              *bool                        `json:"tailscale_active,omitempty"`
	Team                   string                       `json:"team_id,omitempty"`
	LastUpdate             string                       `json:"updated_at,omitempty"`
	ResponsePayload        v1beta1.SchemalessObject     `json:""`
}

func (c *ClusterApiResource) AddDataToClusterStatus(cluster *v1beta1.CrunchyBridgeCluster) {
	cluster.Status.ClusterName = c.ClusterName
	cluster.Status.Host = c.Host
	cluster.Status.ID = c.ID
	cluster.Status.IsHA = c.IsHA
	cluster.Status.IsProtected = c.IsProtected
	cluster.Status.MajorVersion = c.MajorVersion
	cluster.Status.Plan = c.Plan
	cluster.Status.Storage = FromGibibytes(c.Storage)
	cluster.Status.Responses.Cluster = c.ResponsePayload
}

type ClusterList struct {
	Clusters []*ClusterApiResource `json:"clusters"`
}

// ClusterDiskUsageApiResource hold information on disk usage for a particular cluster.
type ClusterDiskUsageApiResource struct {
	DiskAvailableMB int64 `json:"disk_available_mb,omitempty"`
	DiskTotalSizeMB int64 `json:"disk_total_size_mb,omitempty"`
	DiskUsedMB      int64 `json:"disk_used_mb,omitempty"`
}

// ClusterGroupApiResource holds information on a ClusterGroup
type ClusterGroupApiResource struct {
	ID       string                `json:"id,omitempty"`
	Clusters []*ClusterApiResource `json:"clusters,omitempty"`
	Kind     string                `json:"kind,omitempty"`
	Name     string                `json:"name,omitempty"`
	Network  string                `json:"network_id,omitempty"`
	Provider string                `json:"provider_id,omitempty"`
	Region   string                `json:"region_id,omitempty"`
	Team     string                `json:"team_id,omitempty"`
}

type ClusterStatusApiResource struct {
	DiskUsage       *ClusterDiskUsageApiResource `json:"disk_usage,omitempty"`
	OldestBackup    string                       `json:"oldest_backup_at,omitempty"`
	OngoingUpgrade  *ClusterUpgradeApiResource   `json:"ongoing_upgrade,omitempty"`
	State           string                       `json:"state,omitempty"`
	ResponsePayload v1beta1.SchemalessObject     `json:""`
}

func (c *ClusterStatusApiResource) AddDataToClusterStatus(cluster *v1beta1.CrunchyBridgeCluster) {
	cluster.Status.State = c.State
	cluster.Status.Responses.Status = c.ResponsePayload
}

type ClusterUpgradeApiResource struct {
	ClusterID       string                      `json:"cluster_id,omitempty"`
	Operations      []*v1beta1.UpgradeOperation `json:"operations,omitempty"`
	Team            string                      `json:"team_id,omitempty"`
	ResponsePayload v1beta1.SchemalessObject    `json:""`
}

func (c *ClusterUpgradeApiResource) AddDataToClusterStatus(cluster *v1beta1.CrunchyBridgeCluster) {
	cluster.Status.OngoingUpgrade = c.Operations
	cluster.Status.Responses.Upgrade = c.ResponsePayload
}

type ClusterUpgradeOperationApiResource struct {
	Flavor       string `json:"flavor,omitempty"`
	StartingFrom string `json:"starting_from,omitempty"`
	State        string `json:"state,omitempty"`
}

// ClusterRoleApiResource is used for retrieving details on ClusterRole from the Bridge API
type ClusterRoleApiResource struct {
	AccountEmail string `json:"account_email"`
	AccountId    string `json:"account_id"`
	ClusterId    string `json:"cluster_id"`
	Flavor       string `json:"flavor"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	Team         string `json:"team_id"`
	URI          string `json:"uri"`
}

// ClusterRoleList holds a slice of ClusterRoleApiResource
type ClusterRoleList struct {
	Roles []*ClusterRoleApiResource `json:"roles"`
}

// BRIDGE API REQUEST PAYLOADS

// PatchClustersRequestPayload is used for updating various properties of an existing cluster.
type PatchClustersRequestPayload struct {
	ClusterGroup string `json:"cluster_group_id,omitempty"`
	// DashboardSettings      *ClusterDashboardSettings `json:"dashboard_settings,omitempty"`
	// TODO (dsessler7): Find docs for DashboardSettings and create appropriate struct
	Environment            string `json:"environment,omitempty"`
	IsProtected            *bool  `json:"is_protected,omitempty"`
	MaintenanceWindowStart int64  `json:"maintenance_window_start,omitempty"`
	Name                   string `json:"name,omitempty"`
}

// PostClustersRequestPayload is used for creating a new cluster.
type PostClustersRequestPayload struct {
	Name            string             `json:"name"`
	Plan            string             `json:"plan_id"`
	Team            string             `json:"team_id"`
	ClusterGroup    string             `json:"cluster_group_id,omitempty"`
	Environment     string             `json:"environment,omitempty"`
	IsHA            bool               `json:"is_ha,omitempty"`
	Keychain        string             `json:"keychain_id,omitempty"`
	Network         string             `json:"network_id,omitempty"`
	PostgresVersion intstr.IntOrString `json:"postgres_version_id,omitempty"`
	Provider        string             `json:"provider_id,omitempty"`
	Region          string             `json:"region_id,omitempty"`
	Storage         int64              `json:"storage,omitempty"`
}

// PostClustersUpgradeRequestPayload is used for creating a new cluster upgrade which may include
// changing its plan, upgrading its major version, or increasing its storage size.
type PostClustersUpgradeRequestPayload struct {
	Plan             string             `json:"plan_id,omitempty"`
	PostgresVersion  intstr.IntOrString `json:"postgres_version_id,omitempty"`
	UpgradeStartTime string             `json:"starting_from,omitempty"`
	Storage          int64              `json:"storage,omitempty"`
}

// PutClustersUpgradeRequestPayload is used for updating an ongoing or scheduled upgrade.
// TODO: Implement the ability to update an upgrade (this isn't currently being used)
type PutClustersUpgradeRequestPayload struct {
	Plan                 string             `json:"plan_id,omitempty"`
	PostgresVersion      intstr.IntOrString `json:"postgres_version_id,omitempty"`
	UpgradeStartTime     string             `json:"starting_from,omitempty"`
	Storage              int64              `json:"storage,omitempty"`
	UseMaintenanceWindow *bool              `json:"use_cluster_maintenance_window,omitempty"`
}

// BRIDGE CLIENT FUNCTIONS AND METHODS

// NewClient creates a Client with backoff settings that amount to
// ~10 attempts over ~2 minutes. A default is used when apiURL is not
// an acceptable URL.
func NewClient(apiURL, version string) *Client {
	// Use the default URL when the argument (1) does not parse at all, or
	// (2) has the wrong scheme, or (3) has no hostname.
	base, err := url.Parse(apiURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" {
		base, _ = url.Parse(defaultAPI)
	}

	return &Client{
		Backoff: wait.Backoff{
			Duration: time.Second,
			Factor:   1.6,
			Jitter:   0.2,
			Steps:    10,
			Cap:      time.Minute,
		},
		BaseURL: *base,
		Version: version,
	}
}

// doWithBackoff performs HTTP requests until:
//  1. ctx is cancelled,
//  2. the server returns a status code below 500, "Internal Server Error", or
//  3. the backoff is exhausted.
//
// Be sure to close the [http.Response] Body when the returned error is nil.
// See [http.Client.Do] for more details.
func (c *Client) doWithBackoff(
	ctx context.Context, method, path string, params url.Values, body []byte, headers http.Header,
) (
	*http.Response, error,
) {
	var response *http.Response

	// Prepare a copy of the passed in headers so we can manipulate them.
	if headers = headers.Clone(); headers == nil {
		headers = make(http.Header)
	}

	// Send a value that identifies this PATCH or POST request so it is safe to
	// retry when the server does not respond.
	// - https://docs.crunchybridge.com/api-concepts/idempotency/
	if method == http.MethodPatch || method == http.MethodPost {
		headers.Set("Idempotency-Key", string(uuid.NewUUID()))
	}

	headers.Set("User-Agent", "PGO/"+c.Version)
	url := c.BaseURL.JoinPath(path)
	if params != nil {
		url.RawQuery = params.Encode()
	}
	urlString := url.String()

	err := wait.ExponentialBackoff(c.Backoff, func() (bool, error) {
		// NOTE: The [net/http] package treats an empty [bytes.Reader] the same as nil.
		request, err := http.NewRequestWithContext(ctx, method, urlString, bytes.NewReader(body))

		if err == nil {
			request.Header = headers.Clone()

			//nolint:bodyclose // This response is returned to the caller.
			response, err = c.Client.Do(request)
		}

		// An error indicates there was no response from the server, and the
		// request may not have finished. The "Idempotency-Key" header above
		// makes it safe to retry in this case.
		finished := err == nil

		// When the request finishes with a server error, discard the body and retry.
		// - https://docs.crunchybridge.com/api-concepts/getting-started/#status-codes
		if finished && response.StatusCode >= 500 {
			_ = response.Body.Close()
			finished = false
		}

		// Stop when the context is cancelled.
		return finished, ctx.Err()
	})

	// Discard the response body when there is a timeout from backoff.
	if response != nil && err != nil {
		_ = response.Body.Close()
	}

	// Return the last response, if any.
	// Return the cancellation or timeout from backoff, if any.
	return response, err
}

// doWithRetry performs HTTP requests until:
//  1. ctx is cancelled,
//  2. the server returns a status code below 500, "Internal Server Error",
//     that is not 429, "Too many requests", or
//  3. the backoff is exhausted.
//
// Be sure to close the [http.Response] Body when the returned error is nil.
// See [http.Client.Do] for more details.
func (c *Client) doWithRetry(
	ctx context.Context, method, path string, params url.Values, body []byte, headers http.Header,
) (
	*http.Response, error,
) {
	response, err := c.doWithBackoff(ctx, method, path, params, body, headers)

	// Retry the request when the server responds with "Too many requests".
	// - https://docs.crunchybridge.com/api-concepts/getting-started/#status-codes
	// - https://docs.crunchybridge.com/api-concepts/getting-started/#rate-limiting
	for err == nil && response.StatusCode == 429 {
		seconds, _ := strconv.Atoi(response.Header.Get("Retry-After"))

		// Only retry when the response indicates how long to wait.
		if seconds <= 0 {
			break
		}

		// Discard the "Too many requests" response body, and retry.
		_ = response.Body.Close()

		// Create a channel that sends after the delay indicated by the API.
		timer := time.NewTimer(time.Duration(seconds) * time.Second)
		defer timer.Stop()

		// Wait for the delay or context cancellation, whichever comes first.
		select {
		case <-timer.C:
			// Try the request again. Check it in the loop condition.
			response, err = c.doWithBackoff(ctx, method, path, params, body, headers)
			timer.Stop()

		case <-ctx.Done():
			// Exit the loop and return the context cancellation.
			err = ctx.Err()
		}
	}

	return response, err
}

func (c *Client) CreateAuthObject(ctx context.Context, authn AuthObject) (AuthObject, error) {
	var result AuthObject

	response, err := c.doWithRetry(ctx, "POST", "/vendor/operator/auth-objects", nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + authn.Secret},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		// 401, Unauthorized
		case response.StatusCode == 401:
			err = fmt.Errorf("%w: %s", errAuthentication, body)

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

func (c *Client) CreateInstallation(ctx context.Context) (Installation, error) {
	var result Installation

	response, err := c.doWithRetry(ctx, "POST", "/vendor/operator/installations", nil, nil, http.Header{
		"Accept": []string{"application/json"},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// CRUNCHYBRIDGECLUSTER CRUD METHODS

// ListClusters makes a GET request to the "/clusters" endpoint to retrieve a list of all clusters
// in Bridge that are owned by the team specified by the provided team id.
func (c *Client) ListClusters(ctx context.Context, apiKey, teamId string) ([]*ClusterApiResource, error) {
	result := &ClusterList{}

	params := url.Values{}
	if len(teamId) > 0 {
		params.Add("team_id", teamId)
	}
	response, err := c.doWithRetry(ctx, "GET", "/clusters", params, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result.Clusters, err
}

// CreateCluster makes a POST request to the "/clusters" endpoint thereby creating a cluster
// in Bridge with the settings specified in the request payload.
func (c *Client) CreateCluster(
	ctx context.Context, apiKey string, clusterRequestPayload *PostClustersRequestPayload,
) (*ClusterApiResource, error) {
	result := &ClusterApiResource{}

	clusterbyte, err := json.Marshal(clusterRequestPayload)
	if err != nil {
		return result, err
	}

	response, err := c.doWithRetry(ctx, "POST", "/clusters", nil, clusterbyte, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// DeleteCluster calls the delete endpoint, returning
//
//	the cluster,
//	whether the cluster is deleted already,
//	and an error.
func (c *Client) DeleteCluster(ctx context.Context, apiKey, id string) (*ClusterApiResource, bool, error) {
	result := &ClusterApiResource{}
	var deletedAlready bool

	response, err := c.doWithRetry(ctx, "DELETE", "/clusters/"+id, nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		// Already deleted
		// Bridge API returns 410 Gone for previously deleted clusters
		// --https://docs.crunchybridge.com/api-concepts/idempotency#delete-semantics
		// But also, if we can't find it...
		// Maybe if no ID we return already deleted?
		case response.StatusCode == 410:
			fallthrough
		case response.StatusCode == 404:
			deletedAlready = true
			err = nil

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, deletedAlready, err
}

// GetCluster makes a GET request to the "/clusters/<id>" endpoint, thereby retrieving details
// for a given cluster in Bridge specified by the provided cluster id.
func (c *Client) GetCluster(ctx context.Context, apiKey, id string) (*ClusterApiResource, error) {
	result := &ClusterApiResource{}

	response, err := c.doWithRetry(ctx, "GET", "/clusters/"+id, nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// GetClusterStatus makes a GET request to the "/clusters/<id>/status" endpoint, thereby retrieving details
// for a given cluster's status in Bridge, specified by the provided cluster id.
func (c *Client) GetClusterStatus(ctx context.Context, apiKey, id string) (*ClusterStatusApiResource, error) {
	result := &ClusterStatusApiResource{}

	response, err := c.doWithRetry(ctx, "GET", "/clusters/"+id+"/status", nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// GetClusterUpgrade makes a GET request to the "/clusters/<id>/upgrade" endpoint, thereby retrieving details
// for a given cluster's upgrade status in Bridge, specified by the provided cluster id.
func (c *Client) GetClusterUpgrade(ctx context.Context, apiKey, id string) (*ClusterUpgradeApiResource, error) {
	result := &ClusterUpgradeApiResource{}

	response, err := c.doWithRetry(ctx, "GET", "/clusters/"+id+"/upgrade", nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// UpgradeCluster makes a POST request to the "/clusters/<id>/upgrade" endpoint, thereby attempting
// to upgrade certain settings for a given cluster in Bridge.
func (c *Client) UpgradeCluster(
	ctx context.Context, apiKey, id string, clusterRequestPayload *PostClustersUpgradeRequestPayload,
) (*ClusterUpgradeApiResource, error) {
	result := &ClusterUpgradeApiResource{}

	clusterbyte, err := json.Marshal(clusterRequestPayload)
	if err != nil {
		return result, err
	}

	response, err := c.doWithRetry(ctx, "POST", "/clusters/"+id+"/upgrade", nil, clusterbyte, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// UpgradeClusterHA makes a PUT request to the "/clusters/<id>/actions/<action>" endpoint,
// where <action> is either "enable-ha" or "disable-ha", thereby attempting to change the
// HA setting for a given cluster in Bridge.
func (c *Client) UpgradeClusterHA(ctx context.Context, apiKey, id, action string) (*ClusterUpgradeApiResource, error) {
	result := &ClusterUpgradeApiResource{}

	response, err := c.doWithRetry(ctx, "PUT", "/clusters/"+id+"/actions/"+action, nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// UpdateCluster makes a PATCH request to the "/clusters/<id>" endpoint, thereby attempting to
// update certain settings for a given cluster in Bridge.
func (c *Client) UpdateCluster(
	ctx context.Context, apiKey, id string, clusterRequestPayload *PatchClustersRequestPayload,
) (*ClusterApiResource, error) {
	result := &ClusterApiResource{}

	clusterbyte, err := json.Marshal(clusterRequestPayload)
	if err != nil {
		return result, err
	}

	response, err := c.doWithRetry(ctx, "PATCH", "/clusters/"+id, nil, clusterbyte, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
				return result, err
			}
			if err = json.Unmarshal(body, &result.ResponsePayload); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// GetClusterRole sends a GET request to the "/clusters/<id>/roles/<roleName>" endpoint, thereby retrieving
// Role information for a specific role from a specific cluster in Bridge.
func (c *Client) GetClusterRole(ctx context.Context, apiKey, clusterId, roleName string) (*ClusterRoleApiResource, error) {
	result := &ClusterRoleApiResource{}

	response, err := c.doWithRetry(ctx, "GET", "/clusters/"+clusterId+"/roles/"+roleName, nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result, err
}

// ListClusterRoles sends a GET request to the "/clusters/<id>/roles" endpoint thereby retrieving
// a list of all cluster roles for a specific cluster in Bridge.
func (c *Client) ListClusterRoles(ctx context.Context, apiKey, id string) ([]*ClusterRoleApiResource, error) {
	result := ClusterRoleList{}

	response, err := c.doWithRetry(ctx, "GET", "/clusters/"+id+"/roles", nil, nil, http.Header{
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + apiKey},
	})

	if err == nil {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)

		switch {
		// 2xx, Successful
		case response.StatusCode >= 200 && response.StatusCode < 300:
			if err = json.Unmarshal(body, &result); err != nil {
				err = fmt.Errorf("%w: %s", err, body)
			}

		default:
			//nolint:goerr113 // This is intentionally dynamic.
			err = fmt.Errorf("%v: %s", response.Status, body)
		}
	}

	return result.Roles, err
}
