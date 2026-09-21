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

// SubuserAccess is the request spelling accepted by rgw_str_to_perm().
type SubuserAccess string

const (
	SubuserAccessRead      SubuserAccess = "read"
	SubuserAccessWrite     SubuserAccess = "write"
	SubuserAccessReadWrite SubuserAccess = "readwrite"
	SubuserAccessFull      SubuserAccess = "full"
)

type CreateSubuserRequest struct {
	UID               string
	Subuser           string
	Access            SubuserAccess
	KeyType           UserKeyType
	AccessKey         string
	SecretKey         string
	GenerateSecret    *bool
	GenerateAccessKey *bool
}

type UpdateSubuserRequest struct {
	UID            string
	Subuser        string
	Access         SubuserAccess
	KeyType        UserKeyType
	SecretKey      string
	GenerateSecret *bool
}

type DeleteSubuserRequest struct {
	UID       string
	Subuser   string
	PurgeKeys *bool
}

// CreateSubuser creates a subuser through PUT /admin/user?subuser and returns
// all subusers belonging to the user.
func (client *Client) CreateSubuser(ctx context.Context, input CreateSubuserRequest) ([]Subuser, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	if err := validateSubuser(input.UID, input.Subuser); err != nil {
		return nil, err
	}
	query := url.Values{
		"subuser": {input.Subuser},
		"uid":     {input.UID},
	}
	setString(query, "access", string(input.Access))
	setString(query, "key-type", string(input.KeyType))
	setString(query, "access-key", input.AccessKey)
	setString(query, "secret-key", input.SecretKey)
	setBool(query, "generate-secret", input.GenerateSecret)
	setBool(query, "gen-access-key", input.GenerateAccessKey)
	return client.subuserRequest(ctx, http.MethodPut, query)
}

// UpdateSubuser modifies a subuser through POST /admin/user?subuser and
// returns all subusers belonging to the user.
func (client *Client) UpdateSubuser(ctx context.Context, input UpdateSubuserRequest) ([]Subuser, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	if err := validateSubuser(input.UID, input.Subuser); err != nil {
		return nil, err
	}
	query := url.Values{
		"subuser": {input.Subuser},
		"uid":     {input.UID},
	}
	setString(query, "access", string(input.Access))
	setString(query, "key-type", string(input.KeyType))
	setString(query, "secret-key", input.SecretKey)
	setBool(query, "generate-secret", input.GenerateSecret)
	return client.subuserRequest(ctx, http.MethodPost, query)
}

// DeleteSubuser deletes a subuser through DELETE /admin/user?subuser.
func (client *Client) DeleteSubuser(ctx context.Context, input DeleteSubuserRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if err := validateSubuser(input.UID, input.Subuser); err != nil {
		return err
	}
	query := url.Values{
		"subuser": {input.Subuser},
		"uid":     {input.UID},
	}
	setBool(query, "purge-keys", input.PurgeKeys)
	_, _, err := client.userRawRequest(ctx, http.MethodDelete, query)
	return err
}

func (client *Client) subuserRequest(ctx context.Context, method string, query url.Values) ([]Subuser, error) {
	body, request, err := client.userRawRequest(ctx, method, query)
	if err != nil {
		return nil, err
	}
	var subusers []Subuser
	if err := json.Unmarshal(body, &subusers); err != nil {
		return nil, fmt.Errorf("rgw: decode %s %s response: %w", method, request.URL.Path, err)
	}
	return subusers, nil
}

func validateSubuser(uid, subuser string) error {
	if strings.TrimSpace(uid) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if strings.TrimSpace(subuser) == "" {
		return errors.New("rgw: subuser name must not be empty")
	}
	return nil
}
