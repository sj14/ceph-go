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

// CreatedKeys contains the key collection returned by Ceph. Exactly one
// field is populated according to CreateKeyRequest.KeyType.
type CreatedKeys struct {
	AccessKeys []AccessKey
	SwiftKeys  []SwiftKey
}

type CreateKeyRequest struct {
	UID         string
	Subuser     string
	AccessKey   string
	SecretKey   string
	KeyType     UserKeyType
	GenerateKey *bool
	Active      *bool
}

type DeleteKeyRequest struct {
	UID       string
	Subuser   string
	AccessKey string
	KeyType   UserKeyType
}

// CreateKey creates an S3 or Swift key through PUT /admin/user?key. KeyType
// selects the response representation returned by Ceph.
func (client *Client) CreateKey(ctx context.Context, input CreateKeyRequest) (CreatedKeys, error) {
	if ctx == nil {
		return CreatedKeys{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return CreatedKeys{}, errors.New("rgw: user UID must not be empty")
	}
	if input.KeyType != UserKeyTypeS3 && input.KeyType != UserKeyTypeSwift {
		return CreatedKeys{}, errors.New("rgw: key type must be s3 or swift")
	}
	query := url.Values{
		"key": {""},
		"uid": {input.UID},
	}
	setString(query, "subuser", input.Subuser)
	setString(query, "access-key", input.AccessKey)
	setString(query, "secret-key", input.SecretKey)
	setString(query, "key-type", string(input.KeyType))
	setBool(query, "generate-key", input.GenerateKey)
	setBool(query, "active", input.Active)
	body, request, err := client.userRawRequest(ctx, http.MethodPut, query)
	if err != nil {
		return CreatedKeys{}, err
	}
	var keys CreatedKeys
	if input.KeyType == UserKeyTypeS3 {
		err = json.Unmarshal(body, &keys.AccessKeys)
	} else {
		err = json.Unmarshal(body, &keys.SwiftKeys)
	}
	if err != nil {
		return CreatedKeys{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return keys, nil
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
