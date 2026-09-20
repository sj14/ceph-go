package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// GatewayInfo contains the extensible information returned by RGW about its
// storage backends.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/rgw_rest_info.cc
//   - src/rgw/rgw_rest_info.h
type GatewayInfo struct {
	StorageBackends []StorageBackend `json:"storage_backends"`
}

// StorageBackend identifies a storage backend available to RGW.
type StorageBackend struct {
	Name      string `json:"name"`
	ClusterID string `json:"cluster_id"`
}

// GetGatewayInfo retrieves general RGW information through GET /admin/info.
func (client *Client) GetGatewayInfo(ctx context.Context) (GatewayInfo, error) {
	if ctx == nil {
		return GatewayInfo{}, errors.New("admin: context must not be nil")
	}
	request, err := client.newRequest(ctx, http.MethodGet, "info", nil)
	if err != nil {
		return GatewayInfo{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return GatewayInfo{}, err
	}
	var response struct {
		Info GatewayInfo `json:"info"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return GatewayInfo{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return response.Info, nil
}
