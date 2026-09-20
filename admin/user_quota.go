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

type QuotaScope string

const (
	QuotaScopeUser   QuotaScope = "user"
	QuotaScopeBucket QuotaScope = "bucket"
)

type UserQuotas struct {
	Bucket Quota `json:"bucket_quota"`
	User   Quota `json:"user_quota"`
}

type GetUserQuotaRequest struct {
	UID   string
	Scope QuotaScope
}

type SetUserQuotaRequest struct {
	UID        string
	Scope      QuotaScope
	MaxObjects *int64
	MaxSize    *int64
	MaxSizeKB  *int64
	Enabled    *bool
}

// GetUserQuota gets both quotas, or one selected quota, through
// GET /admin/user?quota. A selected quota is normalized into the matching
// field of UserQuotas.
func (client *Client) GetUserQuota(ctx context.Context, input GetUserQuotaRequest) (UserQuotas, error) {
	if ctx == nil {
		return UserQuotas{}, errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return UserQuotas{}, errors.New("admin: user UID must not be empty")
	}
	if input.Scope != "" && input.Scope != QuotaScopeUser && input.Scope != QuotaScopeBucket {
		return UserQuotas{}, errors.New("admin: quota scope must be empty, user, or bucket")
	}
	query := url.Values{
		"quota": {""},
		"uid":   {input.UID},
	}
	setString(query, "quota-type", string(input.Scope))
	body, request, err := client.userRawRequest(ctx, http.MethodGet, query)
	if err != nil {
		return UserQuotas{}, err
	}
	var quotas UserQuotas
	if input.Scope == "" {
		err = json.Unmarshal(body, &quotas)
	} else {
		var quota Quota
		err = json.Unmarshal(body, &quota)
		if input.Scope == QuotaScopeUser {
			quotas.User = quota
		} else {
			quotas.Bucket = quota
		}
	}
	if err != nil {
		return UserQuotas{}, fmt.Errorf("admin: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return quotas, nil
}

// SetUserQuota updates a user or bucket quota through PUT /admin/user?quota.
func (client *Client) SetUserQuota(ctx context.Context, input SetUserQuotaRequest) error {
	if ctx == nil {
		return errors.New("admin: context must not be nil")
	}
	if strings.TrimSpace(input.UID) == "" {
		return errors.New("admin: user UID must not be empty")
	}
	if input.Scope != QuotaScopeUser && input.Scope != QuotaScopeBucket {
		return errors.New("admin: quota scope must be user or bucket")
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
