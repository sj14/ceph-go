package rgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AddUserCapabilityRequest identifies a capability to grant to an RGW user.
type AddUserCapabilityRequest struct {
	UID        string
	Type       string
	Permission string
	DaemonName string
}

// AddUserCapability grants a capability through
// POST /api/rgw/user/{uid}/capability. Ceph returns all capabilities belonging
// to the user after the change.
//
// Common capability Type values exposed by Ceph Dashboard are "users",
// "buckets", "metadata", "usage", and "zone". Permission accepts "read",
// "write", or "*" for both permissions. Ceph's backend supports additional
// administrative capability types beyond these common examples.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwUser.create_cap)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-user.service.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-user-capabilities.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwClient.proxy)
//   - src/rgw/rgw_common.cc (RGWUserCaps)
//   - qa/tasks/mgr/dashboard/test_rgw.py (RgwUserCapabilityTest)
func (client *Client) AddUserCapability(ctx context.Context, input AddUserCapabilityRequest) ([]UserCapability, error) {
	if err := validateUserCapabilityRequest(ctx, input.UID, input.Type, input.Permission); err != nil {
		return nil, err
	}

	endpoint := client.userResourceEndpoint(input.UID, "capability")
	query := url.Values{
		"type": {input.Type},
		"perm": {input.Permission},
	}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := client.do(request)
	if err != nil {
		return nil, err
	}

	var capabilities []UserCapability
	if err := json.Unmarshal(body, &capabilities); err != nil {
		return nil, fmt.Errorf("rgw: decode POST %s response: %w", request.URL.Path, err)
	}
	return capabilities, nil
}

// DeleteUserCapabilityRequest identifies a capability to revoke from an RGW
// user.
type DeleteUserCapabilityRequest struct {
	UID        string
	Type       string
	Permission string
	DaemonName string
}

// DeleteUserCapability revokes a capability through
// DELETE /api/rgw/user/{uid}/capability.
func (client *Client) DeleteUserCapability(ctx context.Context, input DeleteUserCapabilityRequest) error {
	if err := validateUserCapabilityRequest(ctx, input.UID, input.Type, input.Permission); err != nil {
		return err
	}

	endpoint := client.userResourceEndpoint(input.UID, "capability")
	query := url.Values{
		"type": {input.Type},
		"perm": {input.Permission},
	}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func validateUserCapabilityRequest(ctx context.Context, uid, capabilityType, permission string) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if strings.TrimSpace(capabilityType) == "" {
		return errors.New("rgw: capability type must not be empty")
	}
	if strings.TrimSpace(permission) == "" {
		return errors.New("rgw: capability permission must not be empty")
	}
	return nil
}
