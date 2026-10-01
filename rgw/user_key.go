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

// CreateS3KeyRequest creates an S3 credential for UID, optionally associated
// with Subuser. GenerateKey defaults to true when omitted.
//
// Verified against Ceph v20.2.4's src/rgw/driver/rados/rgw_rest_user.cc
// (RGWOp_Key_Create) and rgw_user.cc (RGWUserAdminOp_Key::create).
type CreateS3KeyRequest struct {
	UID         string
	Subuser     string
	AccessKey   string
	SecretKey   string
	GenerateKey *bool
	Active      *bool
}

// CreateS3Key creates an S3 credential through PUT /admin/user?key&key-type=s3.
// It returns all S3 keys belonging to the user after creation, including
// existing keys; the result is not limited to the new credential.
func (client *Client) CreateS3Key(ctx context.Context, input CreateS3KeyRequest) ([]AccessKey, error) {
	query := url.Values{
		"key-type": {string(UserKeyTypeS3)},
	}
	setString(query, "subuser", input.Subuser)
	setString(query, "access-key", input.AccessKey)
	setString(query, "secret-key", input.SecretKey)
	setBool(query, "generate-key", input.GenerateKey)
	setBool(query, "active", input.Active)
	var keys []AccessKey
	if err := client.createKey(ctx, input.UID, query, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// CreateSwiftKeyRequest creates a Swift credential for a subuser. UID and
// Subuser are required. GenerateKey defaults to true when omitted. Ceph
// derives the Swift identity from UID and Subuser, so there is no AccessKey.
//
// Verified against Ceph v20.2.4's src/rgw/driver/rados/rgw_rest_user.cc
// (RGWOp_Key_Create) and rgw_user.cc (RGWAccessKeyPool::generate_key and
// RGWUserAdminOp_Key::create).
type CreateSwiftKeyRequest struct {
	UID         string
	Subuser     string
	SecretKey   string
	GenerateKey *bool
	Active      *bool
}

// CreateSwiftKey creates a Swift credential through
// PUT /admin/user?key&key-type=swift. It returns all Swift keys belonging to
// the user after creation, including keys for other subusers.
func (client *Client) CreateSwiftKey(ctx context.Context, input CreateSwiftKeyRequest) ([]SwiftKey, error) {
	query := url.Values{
		"key-type": {string(UserKeyTypeSwift)},
	}
	setString(query, "subuser", input.Subuser)
	setString(query, "secret-key", input.SecretKey)
	setBool(query, "generate-key", input.GenerateKey)
	setBool(query, "active", input.Active)
	var keys []SwiftKey
	if err := client.createKey(ctx, input.UID, query, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (client *Client) createKey(ctx context.Context, uid string, query url.Values, keys any) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if query.Get("key-type") == string(UserKeyTypeSwift) && strings.TrimSpace(query.Get("subuser")) == "" {
		return errors.New("rgw: subuser name must not be empty")
	}
	query.Set("key", "")
	query.Set("uid", uid)
	body, request, err := client.userRawRequest(ctx, http.MethodPut, query)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, keys); err != nil {
		return fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return nil
}

// DeleteS3KeyRequest identifies one S3 credential belonging to UID.
type DeleteS3KeyRequest struct {
	UID       string
	AccessKey string
}

// DeleteS3Key deletes one S3 credential through
// DELETE /admin/user?key&key-type=s3. UID and AccessKey are required.
//
// Verified against Ceph v20.2.4's src/rgw/driver/rados/rgw_rest_user.cc
// (RGWOp_Key_Remove) and rgw_user.cc (RGWAccessKeyPool::check_op,
// check_existing_key, execute_remove, and RGWUserAdminOp_Key::remove).
func (client *Client) DeleteS3Key(ctx context.Context, input DeleteS3KeyRequest) error {
	return client.deleteKey(ctx, input.UID, UserKeyTypeS3, "access-key", input.AccessKey)
}

// DeleteSwiftKeyRequest identifies the Swift credential of a subuser of UID.
// Subuser is the unqualified name, without a UID prefix.
type DeleteSwiftKeyRequest struct {
	UID     string
	Subuser string
}

// DeleteSwiftKey deletes one Swift credential through
// DELETE /admin/user?key&key-type=swift. UID and Subuser are required.
// The subuser itself and its S3 credentials are preserved.
func (client *Client) DeleteSwiftKey(ctx context.Context, input DeleteSwiftKeyRequest) error {
	return client.deleteKey(ctx, input.UID, UserKeyTypeSwift, "subuser", input.Subuser)
}

func (client *Client) deleteKey(ctx context.Context, uid string, keyType UserKeyType, selector, value string) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("rgw: %s must not be empty", selector)
	}
	// Ceph's set_subuser interprets a UID prefix as an override of uid.
	if keyType == UserKeyTypeSwift && strings.Contains(value, ":") {
		return errors.New("rgw: subuser must be an unqualified name without a UID prefix")
	}
	query := url.Values{
		"key":      {""},
		"uid":      {uid},
		"key-type": {string(keyType)},
		selector:   {value},
	}
	_, _, err := client.userRawRequest(ctx, http.MethodDelete, query)
	return err
}
