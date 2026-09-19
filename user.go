package rgw

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

// User is the RGW user representation assembled by Ceph Dashboard.
//
// Stats is nil when the endpoint did not request usage statistics. Keys and
// SwiftKeys can be nil when the authenticated Dashboard user is not allowed to
// view credentials.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwUser)
//   - src/pybind/mgr/dashboard/controllers/_rest_controller.py
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/rgw/models/rgw-user.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/rgw-user.service.ts
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwClient.proxy)
//   - src/rgw/rgw_common.cc (RGWUserInfo::dump)
type User struct {
	UID                 string           `json:"uid"`
	FullUserID          string           `json:"full_user_id"`
	Tenant              string           `json:"tenant"`
	UserID              string           `json:"user_id"`
	DisplayName         string           `json:"display_name"`
	Email               string           `json:"email"`
	Suspended           int64            `json:"suspended"`
	MaxBuckets          int64            `json:"max_buckets"`
	Subusers            []UserSubuser    `json:"subusers"`
	Keys                []UserAccessKey  `json:"keys"`
	SwiftKeys           []UserSwiftKey   `json:"swift_keys"`
	Capabilities        []UserCapability `json:"caps"`
	OperationMask       string           `json:"op_mask"`
	System              bool             `json:"system"`
	Admin               bool             `json:"admin"`
	DefaultPlacement    string           `json:"default_placement"`
	DefaultStorageClass string           `json:"default_storage_class"`
	PlacementTags       []string         `json:"placement_tags"`
	BucketQuota         UserQuota        `json:"bucket_quota"`
	Quota               UserQuota        `json:"user_quota"`
	TempURLKeys         []UserTempURLKey `json:"temp_url_keys"`
	Type                string           `json:"type"`
	MFAIDs              []string         `json:"mfa_ids"`
	AccountID           string           `json:"account_id"`
	Path                string           `json:"path"`
	CreateDate          string           `json:"create_date"`
	Tags                []UserTag        `json:"tags"`
	GroupIDs            []string         `json:"group_ids"`
	Stats               *UserStats       `json:"stats,omitempty"`
	ManagedPolicies     []string         `json:"managed_user_policies,omitempty"`
}

// UserSubuser is an RGW subuser belonging to a user.
type UserSubuser struct {
	ID          string `json:"id"`
	Permissions string `json:"permissions"`
}

// UserAccessKey is an S3 access credential returned with a user when the
// caller has permission to view credentials.
type UserAccessKey struct {
	User       string `json:"user"`
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	Active     bool   `json:"active"`
	CreateDate string `json:"create_date"`
}

// UserSwiftKey is a Swift credential returned with a user when the caller has
// permission to view credentials.
type UserSwiftKey struct {
	User       string `json:"user"`
	SecretKey  string `json:"secret_key"`
	Active     bool   `json:"active"`
	CreateDate string `json:"create_date"`
}

// UserCapability is one RGW administrative capability.
type UserCapability struct {
	Type       string `json:"type"`
	Permission string `json:"perm"`
}

// UserQuota contains either the user-level or bucket-level quota returned for
// an RGW user.
type UserQuota struct {
	Enabled    bool  `json:"enabled"`
	CheckOnRaw bool  `json:"check_on_raw"`
	MaxSize    int64 `json:"max_size"`
	MaxSizeKB  int64 `json:"max_size_kb"`
	MaxObjects int64 `json:"max_objects"`
}

// UserTempURLKey is one indexed Swift temporary URL key.
type UserTempURLKey struct {
	Key   int64  `json:"key"`
	Value string `json:"val"`
}

// UserTag is one IAM tag attached to an account user.
type UserTag struct {
	Key   string `json:"key"`
	Value string `json:"val"`
}

// UserStats contains aggregate object and byte usage for a user.
type UserStats struct {
	Size           int64 `json:"size"`
	SizeActual     int64 `json:"size_actual"`
	SizeUtilized   int64 `json:"size_utilized"`
	SizeKB         int64 `json:"size_kb"`
	SizeKBActual   int64 `json:"size_kb_actual"`
	SizeKBUtilized int64 `json:"size_kb_utilized"`
	NumObjects     int64 `json:"num_objects"`
}

// ListUsersRequest selects the RGW daemon through which Ceph Dashboard should
// retrieve users.
type ListUsersRequest struct {
	DaemonName string
}

// ListUsers retrieves detailed users through GET /api/rgw/user.
//
// Ceph also supports a lightweight response containing only user IDs. This
// method deliberately sends detailed=true so its return type is consistently
// []User.
func (client *Client) ListUsers(ctx context.Context, input ListUsersRequest) ([]User, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}

	endpoint := client.endpoint("api/rgw/user")
	query := url.Values{"detailed": {"true"}}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	body, err := client.do(request)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("rgw: decode GET %s response: %w", request.URL.Path, err)
	}
	return users, nil
}

// GetUserRequest identifies an RGW user. Stats defaults to true when omitted,
// matching Ceph Dashboard.
type GetUserRequest struct {
	UID        string
	DaemonName string
	Stats      *bool
}

// GetUser retrieves a user through GET /api/rgw/user/{uid}.
func (client *Client) GetUser(ctx context.Context, input GetUserRequest) (User, error) {
	if ctx == nil {
		return User{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return User{}, errors.New("rgw: user UID must not be empty")
	}

	endpoint := client.userEndpoint(input.UID)
	query := url.Values{}
	setOptional(query, "daemon_name", input.DaemonName)
	setOptionalBool(query, "stats", input.Stats)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return User{}, err
	}
	return client.doUserRequest(request)
}

// UserAccountPolicyChanges contains managed policies to attach to or detach
// from a user associated with an RGW account.
type UserAccountPolicyChanges struct {
	Attach []string `json:"attach"`
	Detach []string `json:"detach"`
}

// CreateUserRequest contains the parameters accepted by Ceph's RGW user
// create controller. UID and DisplayName are required. Pointer fields preserve
// the distinction between an omitted parameter and an explicit zero value.
type CreateUserRequest struct {
	UID         string
	DisplayName string

	Email           *string
	MaxBuckets      *int64
	System          *bool
	Suspended       *bool
	GenerateKey     *bool
	AccessKey       *string
	SecretKey       *string
	DaemonName      string
	AccountID       *string
	AccountRootUser *bool
	AccountPolicies *UserAccountPolicyChanges
}

// CreateUser creates a user through POST /api/rgw/user.
func (client *Client) CreateUser(ctx context.Context, input CreateUserRequest) (User, error) {
	if ctx == nil {
		return User{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return User{}, errors.New("rgw: user UID must not be empty")
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return User{}, errors.New("rgw: user display name must not be empty")
	}

	endpoint := client.endpoint("api/rgw/user")
	query := url.Values{
		"uid":          {input.UID},
		"display_name": {input.DisplayName},
	}
	if err := setUserMutationQuery(query, input.Email, input.MaxBuckets, input.System, input.Suspended,
		input.DaemonName, input.AccountID, input.AccountRootUser, input.AccountPolicies); err != nil {
		return User{}, err
	}
	setOptionalBool(query, "generate_key", input.GenerateKey)
	setOptionalString(query, "access_key", input.AccessKey)
	setOptionalString(query, "secret_key", input.SecretKey)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return User{}, err
	}
	return client.doUserRequest(request)
}

// UpdateUserRequest contains the parameters accepted by Ceph's RGW user
// update controller. Pointer fields allow callers to explicitly clear strings
// or send false and zero values.
type UpdateUserRequest struct {
	UID string

	DisplayName     *string
	Email           *string
	MaxBuckets      *int64
	System          *bool
	Suspended       *bool
	DaemonName      string
	AccountID       *string
	AccountRootUser *bool
	AccountPolicies *UserAccountPolicyChanges
}

// UpdateUser updates a user through PUT /api/rgw/user/{uid}.
func (client *Client) UpdateUser(ctx context.Context, input UpdateUserRequest) (User, error) {
	if ctx == nil {
		return User{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return User{}, errors.New("rgw: user UID must not be empty")
	}

	endpoint := client.userEndpoint(input.UID)
	query := url.Values{}
	setOptionalString(query, "display_name", input.DisplayName)
	if err := setUserMutationQuery(query, input.Email, input.MaxBuckets, input.System, input.Suspended,
		input.DaemonName, input.AccountID, input.AccountRootUser, input.AccountPolicies); err != nil {
		return User{}, err
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), nil)
	if err != nil {
		return User{}, err
	}
	return client.doUserRequest(request)
}

// DeleteUserRequest identifies the RGW user to delete.
type DeleteUserRequest struct {
	UID        string
	DaemonName string
}

// DeleteUser deletes a user through DELETE /api/rgw/user/{uid}.
func (client *Client) DeleteUser(ctx context.Context, input DeleteUserRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}

	endpoint := client.userEndpoint(input.UID)
	query := url.Values{}
	setOptional(query, "daemon_name", input.DaemonName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), nil)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func (client *Client) userEndpoint(uid string) *url.URL {
	endpoint := client.endpoint("api/rgw/user")
	baseEscapedPath := strings.TrimRight(endpoint.EscapedPath(), "/")
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + uid
	endpoint.RawPath = baseEscapedPath + "/" + url.PathEscape(uid)
	return endpoint
}

func (client *Client) doUserRequest(request *http.Request) (User, error) {
	body, err := client.do(request)
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return user, nil
}

func setUserMutationQuery(query url.Values, email *string, maxBuckets *int64,
	system, suspended *bool, daemonName string, accountID *string, accountRootUser *bool,
	accountPolicies *UserAccountPolicyChanges) error {
	setOptionalString(query, "email", email)
	setOptionalInt(query, "max_buckets", maxBuckets)
	setOptionalBool(query, "system", system)
	setOptionalBool(query, "suspended", suspended)
	setOptional(query, "daemon_name", daemonName)
	setOptionalString(query, "account_id", accountID)
	setOptionalBool(query, "account_root_user", accountRootUser)
	if accountPolicies != nil {
		encoded, err := json.Marshal(accountPolicies)
		if err != nil {
			return fmt.Errorf("rgw: encode account policies: %w", err)
		}
		query.Set("account_policies", string(encoded))
	}
	return nil
}

func setOptionalString(values url.Values, name string, value *string) {
	if value != nil {
		values.Set(name, *value)
	}
}

func setOptionalBool(values url.Values, name string, value *bool) {
	if value != nil {
		values.Set(name, strconv.FormatBool(*value))
	}
}
