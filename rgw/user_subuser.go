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
	// SubuserAccessNone explicitly grants no permissions. Ceph's request value
	// is empty; the corresponding response permission is "<none>".
	SubuserAccessNone      SubuserAccess = ""
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

// UpdateSubuserRequest requires Access on every update, including credential
// changes. Use a pointer to SubuserAccessNone to explicitly clear permissions.
type UpdateSubuserRequest struct {
	UID            string
	Subuser        string
	Access         *SubuserAccess
	KeyType        UserKeyType
	SecretKey      string
	GenerateSecret *bool
}

// UpdateSubuser modifies a subuser through POST /admin/user?subuser and
// returns all subusers belonging to the user.
// Access must not be nil; Ceph treats an omitted access value as no permissions.
//
// Verified against Ceph v20.2.4:
//   - src/rgw/driver/rados/rgw_rest_user.cc (RGWOp_Subuser_Modify)
//   - src/rgw/driver/rados/rgw_user.h (RGWUserAdminOpState::set_perm)
//   - src/rgw/driver/rados/rgw_user.cc (RGWSubUserPool::execute_modify)
//   - src/rgw/rgw_common.cc (rgw_str_to_perm)
func (client *Client) UpdateSubuser(ctx context.Context, input UpdateSubuserRequest) ([]Subuser, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	if err := validateSubuser(input.UID, input.Subuser); err != nil {
		return nil, err
	}
	if input.Access == nil {
		return nil, errors.New("rgw: subuser access must be provided")
	}
	query := url.Values{
		"subuser": {input.Subuser},
		"uid":     {input.UID},
	}
	query.Set("access", string(*input.Access))
	setString(query, "key-type", string(input.KeyType))
	setString(query, "secret-key", input.SecretKey)
	setBool(query, "generate-secret", input.GenerateSecret)
	return client.subuserRequest(ctx, http.MethodPost, query)
}

type DeleteSubuserRequest struct {
	UID       string
	Subuser   string
	PurgeKeys *bool
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
