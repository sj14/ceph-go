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

type DeleteKeyRequest struct {
	UID       string
	Subuser   string
	AccessKey string
	KeyType   UserKeyType
}

// DeleteKey deletes an S3 or Swift key through DELETE /admin/user?key.
func (client *Client) DeleteKey(ctx context.Context, input DeleteKeyRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	query := url.Values{
		"key": {""},
		"uid": {input.UID},
	}
	setString(query, "subuser", input.Subuser)
	setString(query, "access-key", input.AccessKey)
	setString(query, "key-type", string(input.KeyType))
	_, _, err := client.userRawRequest(ctx, http.MethodDelete, query)
	return err
}
