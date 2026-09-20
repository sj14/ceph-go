package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type ACLPermission int64

const (
	ACLPermissionNone         ACLPermission = 0x00
	ACLPermissionRead         ACLPermission = 0x01
	ACLPermissionWrite        ACLPermission = 0x02
	ACLPermissionReadACP      ACLPermission = 0x04
	ACLPermissionWriteACP     ACLPermission = 0x08
	ACLPermissionReadObjects  ACLPermission = 0x10
	ACLPermissionWriteObjects ACLPermission = 0x20
	ACLPermissionFullControl  ACLPermission = 0x0f
)

type ACLGranteeType int64

const (
	ACLGranteeCanonicalUser ACLGranteeType = iota
	ACLGranteeEmailUser
	ACLGranteeGroup
	ACLGranteeUnknown
	ACLGranteeReferer
)

type ACLGroupType int64

const (
	ACLGroupNone ACLGroupType = iota
	ACLGroupAllUsers
	ACLGroupAuthenticatedUsers
)

type BucketPolicy struct {
	ACL   BucketACL `json:"acl"`
	Owner ACLOwner  `json:"owner"`
}

type BucketACL struct {
	Users  []ACLUserEntry  `json:"acl_user_map"`
	Groups []ACLGroupEntry `json:"acl_group_map"`
	Grants []ACLGrantEntry `json:"grant_map"`
}

type ACLUserEntry struct {
	User       string        `json:"user"`
	Permission ACLPermission `json:"acl"`
}

type ACLGroupEntry struct {
	Group      ACLGroupType  `json:"group"`
	Permission ACLPermission `json:"acl"`
}

type ACLGrantEntry struct {
	ID    string   `json:"id"`
	Grant ACLGrant `json:"grant"`
}

type ACLGrant struct {
	Type       ACLGrantType       `json:"type"`
	ID         string             `json:"id,omitempty"`
	Name       string             `json:"name,omitempty"`
	Email      string             `json:"email,omitempty"`
	Group      ACLGroupType       `json:"group,omitempty"`
	URLSpec    string             `json:"url_spec,omitempty"`
	Permission ACLGrantPermission `json:"permission"`
}

type ACLGrantType struct {
	Type ACLGranteeType `json:"type"`
}

type ACLGrantPermission struct {
	Flags ACLPermission `json:"flags"`
}

type ACLOwner struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type GetBucketPolicyRequest struct {
	Name   string
	Object string
}

// GetBucketPolicy retrieves a bucket or object ACL through
// GET /admin/bucket?policy.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_bucket.cc (RGWOp_Get_Policy)
//   - src/rgw/driver/rados/rgw_bucket.cc (RGWBucketAdminOp::get_policy)
//   - src/rgw/rgw_acl.cc and src/rgw/rgw_acl_types.h
func (client *Client) GetBucketPolicy(ctx context.Context, input GetBucketPolicyRequest) (BucketPolicy, error) {
	if ctx == nil {
		return BucketPolicy{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return BucketPolicy{}, errors.New("admin: bucket name must not be empty")
	}
	query := url.Values{
		"bucket": {input.Name},
		"policy": {""},
	}
	setString(query, "object", input.Object)
	body, request, err := client.bucketRequest(ctx, http.MethodGet, query)
	if err != nil {
		return BucketPolicy{}, err
	}
	var policy BucketPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return BucketPolicy{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return policy, nil
}
