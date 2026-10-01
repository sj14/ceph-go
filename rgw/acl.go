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

// AccessControlPolicy contains the owner and ACL grants of a bucket or object.
// It is Ceph's RGWAccessControlPolicy representation.
type AccessControlPolicy struct {
	ACL   AccessControlList `json:"acl"`
	Owner ACLOwner          `json:"owner"`
}

// AccessControlList is Ceph's RGWAccessControlList representation, shared by
// bucket and object access-control policies.
type AccessControlList struct {
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

type GetBucketACLRequest struct {
	Name string
}

// GetBucketACL retrieves a bucket's owner and ACL grants through
// GET /admin/bucket?policy.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/driver/rados/rgw_rest_bucket.cc (RGWOp_Get_Policy)
//   - src/rgw/driver/rados/rgw_bucket.cc (RGWBucketAdminOp::get_policy)
//   - src/rgw/rgw_acl.cc and src/rgw/rgw_acl_types.h
func (client *Client) GetBucketACL(ctx context.Context, input GetBucketACLRequest) (AccessControlPolicy, error) {
	return client.getACL(ctx, input.Name, "")
}

type GetObjectACLRequest struct {
	Bucket string
	Object string
}

// GetObjectACL retrieves an object's owner and ACL grants through
// GET /admin/bucket?policy&object=.... Bucket and Object must be provided.
func (client *Client) GetObjectACL(ctx context.Context, input GetObjectACLRequest) (AccessControlPolicy, error) {
	if ctx == nil {
		return AccessControlPolicy{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.Object) == "" {
		return AccessControlPolicy{}, errors.New("rgw: object name must not be empty")
	}
	return client.getACL(ctx, input.Bucket, input.Object)
}

func (client *Client) getACL(ctx context.Context, bucket, object string) (AccessControlPolicy, error) {
	if ctx == nil {
		return AccessControlPolicy{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(bucket) == "" {
		return AccessControlPolicy{}, errors.New("rgw: bucket name must not be empty")
	}
	query := url.Values{
		"bucket": {bucket},
		"policy": {""},
	}
	setString(query, "object", object)
	request, err := client.newRequest(ctx, http.MethodGet, "bucket", query)
	if err != nil {
		return AccessControlPolicy{}, err
	}
	// RGWHTTPArgs::append (src/rgw/rgw_common.cc) registers only the first
	// Admin Ops subresource. Put policy before object, which also counts as a
	// subresource. SigV4 canonicalizes query order, so reordering the signed
	// parameters preserves the signature.
	query.Del("policy")
	request.URL.RawQuery = "policy=&" + query.Encode()
	body, err := client.do(request)
	if err != nil {
		return AccessControlPolicy{}, err
	}
	var policy AccessControlPolicy
	if err := json.Unmarshal(body, &policy); err != nil {
		return AccessControlPolicy{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return policy, nil
}
