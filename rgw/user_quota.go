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

// QuotaScope identifies a quota object accepted by user and account Admin Ops
// endpoints.
type QuotaScope string

const (
	QuotaScopeUser    QuotaScope = "user"
	QuotaScopeBucket  QuotaScope = "bucket"
	QuotaScopeAccount QuotaScope = "account"
)

type UserQuotas struct {
	Bucket Quota `json:"bucket_quota"`
	User   Quota `json:"user_quota"`
}

// GetUserQuotaRequest identifies the user whose quotas should be read.
// All three quota getters use the same identity and have fixed response shapes.
type GetUserQuotaRequest struct {
	UID string
}

// GetUserQuotas retrieves both user and per-bucket quotas through
// GET /admin/user?quota.
//
// Verified against Ceph v20.2.4: src/rgw/driver/rados/rgw_rest_user.cc
// (RGWOp_Quota_Info and UserQuotas) and src/rgw/rgw_quota.cc (RGWQuotaInfo::dump).
func (client *Client) GetUserQuotas(ctx context.Context, input GetUserQuotaRequest) (UserQuotas, error) {
	var quotas UserQuotas
	if err := client.getUserQuota(ctx, input, "", &quotas); err != nil {
		return UserQuotas{}, err
	}
	return quotas, nil
}

// GetUserQuota retrieves the user's aggregate quota through
// GET /admin/user?quota&quota-type=user.
func (client *Client) GetUserQuota(ctx context.Context, input GetUserQuotaRequest) (Quota, error) {
	var quota Quota
	if err := client.getUserQuota(ctx, input, QuotaScopeUser, &quota); err != nil {
		return Quota{}, err
	}
	return quota, nil
}

// GetUserBucketQuota retrieves the user's per-bucket quota through
// GET /admin/user?quota&quota-type=bucket. This is distinct from the quota
// configured on a particular bucket, which is returned by GetBucket.
func (client *Client) GetUserBucketQuota(ctx context.Context, input GetUserQuotaRequest) (Quota, error) {
	var quota Quota
	if err := client.getUserQuota(ctx, input, QuotaScopeBucket, &quota); err != nil {
		return Quota{}, err
	}
	return quota, nil
}

func (client *Client) getUserQuota(ctx context.Context, input GetUserQuotaRequest, scope QuotaScope, result any) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	query := url.Values{
		"quota": {""},
		"uid":   {input.UID},
	}
	setString(query, "quota-type", string(scope))
	body, request, err := client.userRawRequest(ctx, http.MethodGet, query)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return nil
}

type SetUserQuotaRequest struct {
	UID        string
	Scope      QuotaScope
	MaxObjects *int64
	MaxSize    *int64
	MaxSizeKB  *int64
	Enabled    *bool
}

// SetUserQuota updates a user or bucket quota through PUT /admin/user?quota.
func (client *Client) SetUserQuota(ctx context.Context, input SetUserQuotaRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("rgw: user UID must not be empty")
	}
	if input.Scope != QuotaScopeUser && input.Scope != QuotaScopeBucket {
		return errors.New("rgw: quota scope must be user or bucket")
	}
	query := url.Values{
		"quota":      {""},
		"quota-type": {string(input.Scope)},
		"uid":        {input.UID},
	}
	setInt64(query, "max-objects", input.MaxObjects)
	setInt64(query, "max-size", input.MaxSize)
	setInt64(query, "max-size-kb", input.MaxSizeKB)
	setBool(query, "enabled", input.Enabled)
	_, _, err := client.userRawRequest(ctx, http.MethodPut, query)
	return err
}
