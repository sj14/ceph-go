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

// CreateSwiftKeyRequest contains the parameters accepted by Ceph for a Swift
// subuser credential. UID and Subuser are required. GenerateKey defaults to
// true when omitted.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/rgw.py (RgwUser.create_key)
//   - src/pybind/mgr/dashboard/services/rgw_client.py (RgwClient.proxy)
//   - qa/tasks/mgr/dashboard/test_rgw.py (RgwUserKeyTest.test_create_swift)
type CreateSwiftKeyRequest struct {
	UID         string
	Subuser     string
	GenerateKey *bool
	SecretKey   *string
	DaemonName  string
}

// CreateSwiftKey creates a credential through
// POST /api/rgw/user/{uid}/key. Ceph returns all Swift keys belonging to the
// user after creation.
func (client *Client) CreateSwiftKey(ctx context.Context, input CreateSwiftKeyRequest) ([]UserSwiftKey, error) {
	if err := validateSwiftKeyRequest(ctx, input.UID, input.Subuser); err != nil {
		return nil, err
	}

	endpoint := client.userResourceEndpoint(input.UID, "key")
	query := url.Values{
		"key_type": {"swift"},
		"subuser":  {input.Subuser},
	}
	setOptionalBool(query, "generate_key", input.GenerateKey)
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

	var keys []UserSwiftKey
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, fmt.Errorf("mgr: decode POST %s response: %w", request.URL.Path, err)
	}
	return keys, nil
}

// DeleteSwiftKeyRequest identifies a Swift subuser credential to remove. UID
// and Subuser are required.
type DeleteSwiftKeyRequest struct {
	UID        string
	Subuser    string
	DaemonName string
}

// DeleteSwiftKey deletes a credential through
// DELETE /api/rgw/user/{uid}/key.
func (client *Client) DeleteSwiftKey(ctx context.Context, input DeleteSwiftKeyRequest) error {
	if err := validateSwiftKeyRequest(ctx, input.UID, input.Subuser); err != nil {
		return err
	}

	endpoint := client.userResourceEndpoint(input.UID, "key")
	query := url.Values{
		"key_type": {"swift"},
		"subuser":  {input.Subuser},
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

func validateSwiftKeyRequest(ctx context.Context, uid, subuser string) error {
	if ctx == nil {
		return errors.New("mgr: context must not be nil")
	}
	if strings.TrimSpace(uid) == "" {
		return errors.New("mgr: user UID must not be empty")
	}
	if strings.TrimSpace(subuser) == "" {
		return errors.New("mgr: subuser name must not be empty")
	}
	return nil
}
