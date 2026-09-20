package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserKeyType is an Admin Ops credential type accepted when creating or
// modifying a user.
type UserKeyType string

const (
	UserKeyTypeS3    UserKeyType = "s3"
	UserKeyTypeSwift UserKeyType = "swift"
)

// UserType identifies the source of an RGW user.
type UserType string

const (
	UserTypeRGW      UserType = "rgw"
	UserTypeKeystone UserType = "keystone"
	UserTypeLDAP     UserType = "ldap"
	UserTypeNone     UserType = "none"
	UserTypeRoot     UserType = "root"
)

// UserSuspensionState is the integer suspension state returned by Admin Ops.
type UserSuspensionState int64

const (
	UserActive    UserSuspensionState = 0
	UserSuspended UserSuspensionState = 1
)

// SubuserPermission is the permission spelling returned by Admin Ops.
type SubuserPermission string

const (
	SubuserPermissionNone      SubuserPermission = "<none>"
	SubuserPermissionRead      SubuserPermission = "read"
	SubuserPermissionWrite     SubuserPermission = "write"
	SubuserPermissionReadWrite SubuserPermission = "read-write"
	SubuserPermissionFull      SubuserPermission = "full-control"
	SubuserPermissionReadACP   SubuserPermission = "read-acp"
	SubuserPermissionWriteACP  SubuserPermission = "write-acp"
)

// CapabilityType identifies an RGW administrative capability namespace.
type CapabilityType string

const (
	CapabilityTypeUser                CapabilityType = "user"
	CapabilityTypeUsers               CapabilityType = "users"
	CapabilityTypeBuckets             CapabilityType = "buckets"
	CapabilityTypeMetadata            CapabilityType = "metadata"
	CapabilityTypeInfo                CapabilityType = "info"
	CapabilityTypeUsage               CapabilityType = "usage"
	CapabilityTypeZone                CapabilityType = "zone"
	CapabilityTypeBILog               CapabilityType = "bilog"
	CapabilityTypeMDLog               CapabilityType = "mdlog"
	CapabilityTypeDataLog             CapabilityType = "datalog"
	CapabilityTypeRoles               CapabilityType = "roles"
	CapabilityTypeUserPolicy          CapabilityType = "user-policy"
	CapabilityTypeAMZCache            CapabilityType = "amz-cache"
	CapabilityTypeOIDCProvider        CapabilityType = "oidc-provider"
	CapabilityTypeUserInfoWithoutKeys CapabilityType = "user-info-without-keys"
	CapabilityTypeRateLimit           CapabilityType = "ratelimit"
	CapabilityTypeAccounts            CapabilityType = "accounts"
)

// CapabilityPermission is the permission portion of an RGW administrative
// capability.
type CapabilityPermission string

const (
	CapabilityPermissionRead      CapabilityPermission = "read"
	CapabilityPermissionWrite     CapabilityPermission = "write"
	CapabilityPermissionReadWrite CapabilityPermission = "read, write"
	CapabilityPermissionAll       CapabilityPermission = "*"
)

// User is the direct JSON representation produced by RGW's dump_user_info().
// It intentionally does not include fields added by Ceph Dashboard.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_user.cc
//   - src/rgw/driver/rados/rgw_rest_user.h
//   - src/rgw/driver/rados/rgw_user.cc (dump_user_info and RGWUserAdminOp_User)
//   - src/rgw/rgw_user_types.h
type User struct {
	FullUserID          string              `json:"full_user_id"`
	Tenant              string              `json:"tenant"`
	Namespace           string              `json:"namespace,omitempty"`
	ID                  string              `json:"user_id"`
	DisplayName         string              `json:"display_name"`
	Email               string              `json:"email"`
	Suspended           UserSuspensionState `json:"suspended"`
	MaxBuckets          int64               `json:"max_buckets"`
	Subusers            []Subuser           `json:"subusers"`
	Keys                []AccessKey         `json:"keys"`
	SwiftKeys           []SwiftKey          `json:"swift_keys"`
	Capabilities        []Capability        `json:"caps"`
	OperationMask       string              `json:"op_mask"`
	System              bool                `json:"system"`
	Admin               bool                `json:"admin"`
	DefaultPlacement    string              `json:"default_placement"`
	DefaultStorageClass string              `json:"default_storage_class"`
	PlacementTags       []string            `json:"placement_tags"`
	BucketQuota         Quota               `json:"bucket_quota"`
	Quota               Quota               `json:"user_quota"`
	TempURLKeys         []TempURLKey        `json:"temp_url_keys"`
	Type                UserType            `json:"type"`
	MFAIDs              []string            `json:"mfa_ids"`
	AccountID           string              `json:"account_id"`
	Path                string              `json:"path"`
	CreateDate          time.Time           `json:"create_date"`
	Tags                []Tag               `json:"tags"`
	GroupIDs            []string            `json:"group_ids"`
	Stats               *StorageStats       `json:"stats,omitempty"`
}

type Subuser struct {
	ID          string            `json:"id"`
	Permissions SubuserPermission `json:"permissions"`
}

type AccessKey struct {
	User      string `json:"user"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Active    bool   `json:"active"`
}

type SwiftKey struct {
	User      string `json:"user"`
	SecretKey string `json:"secret_key"`
	Active    bool   `json:"active"`
}

type Capability struct {
	Type       CapabilityType       `json:"type"`
	Permission CapabilityPermission `json:"perm"`
}

type Quota struct {
	Enabled    bool  `json:"enabled"`
	CheckOnRaw bool  `json:"check_on_raw"`
	MaxSize    int64 `json:"max_size"`
	MaxSizeKB  int64 `json:"max_size_kb"`
	MaxObjects int64 `json:"max_objects"`
}

type TempURLKey struct {
	Key   int64  `json:"key"`
	Value string `json:"val"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"val"`
}

type StorageStats struct {
	Size           int64 `json:"size"`
	SizeActual     int64 `json:"size_actual"`
	SizeUtilized   int64 `json:"size_utilized"`
	SizeKB         int64 `json:"size_kb"`
	SizeKBActual   int64 `json:"size_kb_actual"`
	SizeKBUtilized int64 `json:"size_kb_utilized"`
	NumObjects     int64 `json:"num_objects"`
}

// GetUserRequest identifies a user by UID or S3 access key. Stats adds current
// storage statistics; Sync asks RGW to synchronize them before reading.
type GetUserRequest struct {
	UID       string
	AccessKey string
	Stats     *bool
	Sync      *bool
}

// UserList is the paginated response returned by GET /admin/user?list.
type UserList struct {
	IDs       []string `json:"keys"`
	Truncated bool     `json:"truncated"`
	Count     int64    `json:"count"`
	Marker    string   `json:"marker,omitempty"`
}

type ListUsersRequest struct {
	Marker     string
	MaxEntries *int64
}

// CreateUserRequest contains the parameters accepted by PUT /admin/user.
type CreateUserRequest struct {
	UID                 string
	DisplayName         string
	Email               string
	AccessKey           string
	SecretKey           string
	KeyType             UserKeyType
	Capabilities        string
	Tenant              string
	GenerateKey         *bool
	Suspended           *bool
	MaxBuckets          *int64
	System              *bool
	AccountRoot         *bool
	Exclusive           *bool
	OperationMask       string
	DefaultPlacement    string
	DefaultStorageClass string
	PlacementTags       []string
	AccountID           string
	Path                string
}

// UpdateUserRequest contains the parameters accepted by POST /admin/user.
// String pointers preserve the distinction between omission and clearing a
// value where RGW supports it, notably for Email.
type UpdateUserRequest struct {
	UID                 string
	DisplayName         *string
	Email               *string
	AccessKey           *string
	SecretKey           *string
	KeyType             *UserKeyType
	GenerateKey         *bool
	Suspended           *bool
	MaxBuckets          *int64
	System              *bool
	AccountRoot         *bool
	OperationMask       *string
	DefaultPlacement    *string
	DefaultStorageClass *string
	PlacementTags       []string
	AccountID           *string
	Path                *string
}

// DeleteUserRequest identifies a user and optionally allows RGW to delete all
// of the user's data before deleting the user.
type DeleteUserRequest struct {
	UID       string
	PurgeData *bool
}

// ListUsers lists RGW user IDs directly through GET /admin/user?list.
func (client *Client) ListUsers(ctx context.Context, input ListUsersRequest) (UserList, error) {
	if ctx == nil {
		return UserList{}, errors.New("admin: context must not be nil")
	}
	query := url.Values{"list": {""}}
	setString(query, "marker", input.Marker)
	setInt64(query, "max-entries", input.MaxEntries)
	body, request, err := client.userRawRequest(ctx, http.MethodGet, query)
	if err != nil {
		return UserList{}, err
	}
	var users UserList
	if err := json.Unmarshal(body, &users); err != nil {
		return UserList{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return users, nil
}

// GetUser retrieves a user directly through GET /admin/user.
func (client *Client) GetUser(ctx context.Context, input GetUserRequest) (User, error) {
	if ctx == nil {
		return User{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" && strings.TrimSpace(input.AccessKey) == "" {
		return User{}, errors.New("admin: user UID or access key must not be empty")
	}
	query := url.Values{}
	setString(query, "uid", input.UID)
	setString(query, "access-key", input.AccessKey)
	setBool(query, "stats", input.Stats)
	setBool(query, "sync", input.Sync)
	return client.userRequest(ctx, http.MethodGet, query)
}

// CreateUser creates a user directly through PUT /admin/user.
func (client *Client) CreateUser(ctx context.Context, input CreateUserRequest) (User, error) {
	if ctx == nil {
		return User{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return User{}, errors.New("admin: user UID must not be empty")
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return User{}, errors.New("admin: user display name must not be empty")
	}
	query := url.Values{
		"uid":          {input.UID},
		"display-name": {input.DisplayName},
	}
	setString(query, "email", input.Email)
	setString(query, "access-key", input.AccessKey)
	setString(query, "secret-key", input.SecretKey)
	setString(query, "key-type", string(input.KeyType))
	setString(query, "user-caps", input.Capabilities)
	setString(query, "tenant", input.Tenant)
	setBool(query, "generate-key", input.GenerateKey)
	setBool(query, "suspended", input.Suspended)
	setInt64(query, "max-buckets", input.MaxBuckets)
	setBool(query, "system", input.System)
	setBool(query, "account-root", input.AccountRoot)
	setBool(query, "exclusive", input.Exclusive)
	setString(query, "op-mask", input.OperationMask)
	setString(query, "default-placement", input.DefaultPlacement)
	setString(query, "default-storage-class", input.DefaultStorageClass)
	if len(input.PlacementTags) > 0 {
		query.Set("placement-tags", strings.Join(input.PlacementTags, ","))
	}
	setString(query, "account-id", input.AccountID)
	setString(query, "path", input.Path)
	return client.userRequest(ctx, http.MethodPut, query)
}

// UpdateUser modifies a user directly through POST /admin/user.
func (client *Client) UpdateUser(ctx context.Context, input UpdateUserRequest) (User, error) {
	if ctx == nil {
		return User{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return User{}, errors.New("admin: user UID must not be empty")
	}
	query := url.Values{"uid": {input.UID}}
	setStringPointer(query, "display-name", input.DisplayName)
	setStringPointer(query, "email", input.Email)
	setStringPointer(query, "access-key", input.AccessKey)
	setStringPointer(query, "secret-key", input.SecretKey)
	if input.KeyType != nil {
		query.Set("key-type", string(*input.KeyType))
	}
	setBool(query, "generate-key", input.GenerateKey)
	setBool(query, "suspended", input.Suspended)
	setInt64(query, "max-buckets", input.MaxBuckets)
	setBool(query, "system", input.System)
	setBool(query, "account-root", input.AccountRoot)
	setStringPointer(query, "op-mask", input.OperationMask)
	setStringPointer(query, "default-placement", input.DefaultPlacement)
	setStringPointer(query, "default-storage-class", input.DefaultStorageClass)
	if input.PlacementTags != nil {
		query.Set("placement-tags", strings.Join(input.PlacementTags, ","))
	}
	setStringPointer(query, "account-id", input.AccountID)
	setStringPointer(query, "path", input.Path)
	return client.userRequest(ctx, http.MethodPost, query)
}

// DeleteUser deletes a user directly through DELETE /admin/user.
func (client *Client) DeleteUser(ctx context.Context, input DeleteUserRequest) error {
	if ctx == nil {
		return errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("admin: user UID must not be empty")
	}
	query := url.Values{"uid": {input.UID}}
	setBool(query, "purge-data", input.PurgeData)
	request, err := client.newRequest(ctx, http.MethodDelete, "user", query)
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func (client *Client) userRequest(ctx context.Context, method string, query url.Values) (User, error) {
	body, request, err := client.userRawRequest(ctx, method, query)
	if err != nil {
		return User{}, err
	}
	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, fmt.Errorf("admin: decode %s %s response: %w", method, request.URL.Path, err)
	}
	return user, nil
}

func (client *Client) userRawRequest(ctx context.Context, method string, query url.Values) ([]byte, *http.Request, error) {
	request, err := client.newRequest(ctx, method, "user", query)
	if err != nil {
		return nil, nil, err
	}
	body, err := client.do(request)
	if err != nil {
		return nil, request, err
	}
	return body, request, nil
}
