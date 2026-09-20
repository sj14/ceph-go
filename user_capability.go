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

// UserCapabilityType identifies an RGW administrative capability namespace.
// These values are accepted by Ceph v20.2.4; callers can convert a string to
// this type for capability namespaces added by other Ceph versions.
type UserCapabilityType string

const (
	UserCapabilityTypeUser                UserCapabilityType = "user"
	UserCapabilityTypeUsers               UserCapabilityType = "users"
	UserCapabilityTypeBuckets             UserCapabilityType = "buckets"
	UserCapabilityTypeMetadata            UserCapabilityType = "metadata"
	UserCapabilityTypeInfo                UserCapabilityType = "info"
	UserCapabilityTypeUsage               UserCapabilityType = "usage"
	UserCapabilityTypeZone                UserCapabilityType = "zone"
	UserCapabilityTypeBILog               UserCapabilityType = "bilog"
	UserCapabilityTypeMDLog               UserCapabilityType = "mdlog"
	UserCapabilityTypeDataLog             UserCapabilityType = "datalog"
	UserCapabilityTypeRoles               UserCapabilityType = "roles"
	UserCapabilityTypeUserPolicy          UserCapabilityType = "user-policy"
	UserCapabilityTypeAMZCache            UserCapabilityType = "amz-cache"
	UserCapabilityTypeOIDCProvider        UserCapabilityType = "oidc-provider"
	UserCapabilityTypeUserInfoWithoutKeys UserCapabilityType = "user-info-without-keys"
	UserCapabilityTypeRateLimit           UserCapabilityType = "ratelimit"
	UserCapabilityTypeAccounts            UserCapabilityType = "accounts"
)

// UserCapabilityPermission identifies an RGW capability permission. Callers
// can convert a string to express a permission combination supported by Ceph.
type UserCapabilityPermission string

const (
	UserCapabilityPermissionRead  UserCapabilityPermission = "read"
	UserCapabilityPermissionWrite UserCapabilityPermission = "write"
	UserCapabilityPermissionAll   UserCapabilityPermission = "*"
)

// AddUserCapabilityRequest identifies a capability to grant to an RGW user.
type AddUserCapabilityRequest struct {
	UID        string
	Type       UserCapabilityType
	Permission UserCapabilityPermission
	DaemonName string
}

// AddUserCapability grants a capability through
// POST /api/rgw/user/{uid}/capability. Ceph returns all capabilities belonging
// to the user after the change.
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
		"type": {string(input.Type)},
		"perm": {string(input.Permission)},
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
	Type       UserCapabilityType
	Permission UserCapabilityPermission
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
		"type": {string(input.Type)},
		"perm": {string(input.Permission)},
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

func validateUserCapabilityRequest(ctx context.Context, uid string, capabilityType UserCapabilityType,
	permission UserCapabilityPermission) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if strings.TrimSpace(string(capabilityType)) == "" {
		return errors.New("rgw: capability type must not be empty")
	}
	if strings.TrimSpace(string(permission)) == "" {
		return errors.New("rgw: capability permission must not be empty")
	}
	return nil
}
