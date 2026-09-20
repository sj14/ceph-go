package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// UserQuotaConfiguration contains both quota scopes returned by Ceph.
type UserQuotaConfiguration struct {
	User   UserQuota `json:"user_quota"`
	Bucket UserQuota `json:"bucket_quota"`
}

// GetUserQuotaRequest identifies the RGW user whose quotas should be read.
type GetUserQuotaRequest struct {
	UID        string
	DaemonName string
}

// GetUserQuota retrieves both quota scopes through
// GET /api/rgw/user/{uid}/quota.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwUser.get_quota)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-user.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwClient.proxy)
//   - qa/tasks/mgr/dashboard/test_rgw.py (RgwUserQuotaTest)
func (client *Client) GetUserQuota(ctx context.Context, input GetUserQuotaRequest) (UserQuotaConfiguration, error) {
	if ctx == nil {
		return UserQuotaConfiguration{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return UserQuotaConfiguration{}, errors.New("rgw: user UID must not be empty")
	}

	endpoint := client.userResourceEndpoint(input.UID, "quota")
	query := url.Values{}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return UserQuotaConfiguration{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return UserQuotaConfiguration{}, err
	}

	var quota UserQuotaConfiguration
	if err := json.Unmarshal(body, &quota); err != nil {
		return UserQuotaConfiguration{}, fmt.Errorf("rgw: decode GET %s response: %w", request.URL.Path, err)
	}
	return quota, nil
}

// UserQuotaType selects the quota scope to update.
type UserQuotaType string

const (
	UserQuotaTypeUser   UserQuotaType = "user"
	UserQuotaTypeBucket UserQuotaType = "bucket"
)

// UpdateUserQuotaRequest contains the quota values required by Ceph.
type UpdateUserQuotaRequest struct {
	UID        string
	Type       UserQuotaType
	Enabled    bool
	MaxSizeKB  int64
	MaxObjects int64
	DaemonName string
}

// UpdateUserQuota updates one quota scope through
// PUT /api/rgw/user/{uid}/quota.
func (client *Client) UpdateUserQuota(ctx context.Context, input UpdateUserQuotaRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if input.Type != UserQuotaTypeUser && input.Type != UserQuotaTypeBucket {
		return errors.New("rgw: quota type must be user or bucket")
	}

	endpoint := client.userResourceEndpoint(input.UID, "quota")
	query := url.Values{
		"quota_type":  {string(input.Type)},
		"enabled":     {strconv.FormatBool(input.Enabled)},
		"max_size_kb": {strconv.FormatInt(input.MaxSizeKB, 10)},
		"max_objects": {strconv.FormatInt(input.MaxObjects, 10)},
	}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}
