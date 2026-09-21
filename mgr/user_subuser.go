package mgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// SubuserKeyType is the credential type Ceph creates for a subuser.
type SubuserKeyType string

const (
	SubuserKeyTypeS3    SubuserKeyType = "s3"
	SubuserKeyTypeSwift SubuserKeyType = "swift"
)

// SubuserAccess is the access level granted to an RGW subuser.
type SubuserAccess string

const (
	SubuserAccessRead      SubuserAccess = "read"
	SubuserAccessWrite     SubuserAccess = "write"
	SubuserAccessReadWrite SubuserAccess = "readwrite"
	SubuserAccessFull      SubuserAccess = "full"
)

// SubuserPermission is the normalized access level returned by RGW. Ceph uses
// different spellings for the combined and full permissions in responses.
type SubuserPermission string

const (
	SubuserPermissionNone        SubuserPermission = "<none>"
	SubuserPermissionRead        SubuserPermission = "read"
	SubuserPermissionWrite       SubuserPermission = "write"
	SubuserPermissionReadWrite   SubuserPermission = "read-write"
	SubuserPermissionFullControl SubuserPermission = "full-control"
	SubuserPermissionReadACP     SubuserPermission = "read-acp"
	SubuserPermissionWriteACP    SubuserPermission = "write-acp"
)

// CreateSubuserRequest contains the parameters accepted by Ceph's RGW
// subuser controller. UID, Subuser, and Access are required. Access accepts
// "read", "write", "readwrite", or "full".
type CreateSubuserRequest struct {
	UID     string
	Subuser string
	Access  SubuserAccess

	KeyType        SubuserKeyType
	GenerateSecret *bool
	AccessKey      *string
	SecretKey      *string
	DaemonName     string
}

// CreateSubuser creates or updates a subuser through
// POST /api/rgw/user/{uid}/subuser. Ceph returns all of the user's subusers.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwUser.create_subuser)
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-user.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwClient.proxy)
//   - src/rgw/rgw_common.cc (rgw_str_to_perm)
//   - qa/tasks/mgr/dashboard/test_rgw.py (RgwUserSubuserTest)
func (client *Client) CreateSubuser(ctx context.Context, input CreateSubuserRequest) ([]UserSubuser, error) {
	if ctx == nil {
		return nil, errors.New("mgr: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return nil, errors.New("mgr: user UID must not be empty")
	}
	if strings.TrimSpace(input.Subuser) == "" {
		return nil, errors.New("mgr: subuser name must not be empty")
	}
	if strings.TrimSpace(string(input.Access)) == "" {
		return nil, errors.New("mgr: subuser access must not be empty")
	}

	endpoint := client.userResourceEndpoint(input.UID, "subuser")
	query := url.Values{
		"subuser": {input.Subuser},
		"access":  {string(input.Access)},
	}
	setOptional(query, "key_type", string(input.KeyType))
	setOptionalBool(query, "generate_secret", input.GenerateSecret)
	setOptionalString(query, "access_key", input.AccessKey)
	setOptionalString(query, "secret_key", input.SecretKey)
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

	var subusers []UserSubuser
	if err := json.Unmarshal(body, &subusers); err != nil {
		return nil, fmt.Errorf("mgr: decode POST %s response: %w", request.URL.Path, err)
	}
	return subusers, nil
}

// DeleteSubuserRequest identifies a subuser to delete. PurgeKeys defaults to
// true in Ceph when omitted.
type DeleteSubuserRequest struct {
	UID        string
	Subuser    string
	PurgeKeys  *bool
	DaemonName string
}

// DeleteSubuser deletes a subuser through
// DELETE /api/rgw/user/{uid}/subuser/{subuser}.
func (client *Client) DeleteSubuser(ctx context.Context, input DeleteSubuserRequest) error {
	if ctx == nil {
		return errors.New("mgr: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("mgr: user UID must not be empty")
	}
	if strings.TrimSpace(input.Subuser) == "" {
		return errors.New("mgr: subuser name must not be empty")
	}

	endpoint := client.userResourceItemEndpoint(input.UID, "subuser", input.Subuser)
	query := url.Values{}
	setOptionalBool(query, "purge_keys", input.PurgeKeys)
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}
