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

// Account is the direct JSON representation of RGWAccountInfo.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/rgw_rest_account.cc
//   - src/rgw/rgw_account.cc
//   - src/rgw/rgw_common.h (RGWAccountInfo)
//   - src/rgw/rgw_common.cc (RGWAccountInfo::dump)
type Account struct {
	ID            string `json:"id"`
	Tenant        string `json:"tenant"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Quota         Quota  `json:"quota"`
	BucketQuota   Quota  `json:"bucket_quota"`
	MaxUsers      int64  `json:"max_users"`
	MaxRoles      int64  `json:"max_roles"`
	MaxGroups     int64  `json:"max_groups"`
	MaxBuckets    int64  `json:"max_buckets"`
	MaxAccessKeys int64  `json:"max_access_keys"`
}

// GetAccountRequest identifies an account by ID or by tenant and name.
type GetAccountRequest struct {
	ID     string
	Tenant string
	Name   string
}

// CreateAccountRequest contains the parameters accepted by POST /admin/account.
// ID is optional; RGW generates an ID when it is omitted.
type CreateAccountRequest struct {
	ID            string
	Tenant        string
	Name          string
	Email         string
	MaxUsers      *int64
	MaxRoles      *int64
	MaxGroups     *int64
	MaxAccessKeys *int64
	MaxBuckets    *int64
}

// UpdateAccountRequest identifies an account by ID, tenant and name, or email,
// and contains the values accepted by PUT /admin/account. RGW ignores empty
// string values, so the string fields cannot be cleared through this endpoint.
type UpdateAccountRequest struct {
	ID            string
	Tenant        string
	Name          string
	Email         string
	MaxUsers      *int64
	MaxRoles      *int64
	MaxGroups     *int64
	MaxAccessKeys *int64
	MaxBuckets    *int64
}

// SetAccountQuotaRequest modifies either the aggregate account quota or the
// per-bucket quota inherited by buckets in the account.
type SetAccountQuotaRequest struct {
	ID         string
	Scope      QuotaScope
	MaxSize    *int64
	MaxObjects *int64
	Enabled    *bool
}

// DeleteAccountRequest identifies an account by ID or by tenant and name.
// Ceph v20.2.4 only deletes an empty account through this REST endpoint.
type DeleteAccountRequest struct {
	ID     string
	Tenant string
	Name   string
}

// GetAccount retrieves an account directly through GET /admin/account.
func (client *Client) GetAccount(ctx context.Context, input GetAccountRequest) (Account, error) {
	if ctx == nil {
		return Account{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.ID) == "" && strings.TrimSpace(input.Name) == "" {
		return Account{}, errors.New("rgw: account ID or name must not be empty")
	}
	query := accountIdentityQuery(input.ID, input.Tenant, input.Name)
	return client.accountRequest(ctx, http.MethodGet, query)
}

// CreateAccount creates an account directly through POST /admin/account.
func (client *Client) CreateAccount(ctx context.Context, input CreateAccountRequest) (Account, error) {
	if ctx == nil {
		return Account{}, errors.New("rgw: context must not be nil")
	}
	query := accountIdentityQuery(input.ID, input.Tenant, input.Name)
	setString(query, "email", input.Email)
	setAccountLimits(query, input.MaxUsers, input.MaxRoles, input.MaxGroups, input.MaxAccessKeys, input.MaxBuckets)
	return client.accountRequest(ctx, http.MethodPost, query)
}

// UpdateAccount modifies an account directly through PUT /admin/account.
func (client *Client) UpdateAccount(ctx context.Context, input UpdateAccountRequest) (Account, error) {
	if ctx == nil {
		return Account{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.ID) == "" && strings.TrimSpace(input.Name) == "" && strings.TrimSpace(input.Email) == "" {
		return Account{}, errors.New("rgw: account ID, name, or email must not be empty")
	}
	query := accountIdentityQuery(input.ID, input.Tenant, input.Name)
	setString(query, "email", input.Email)
	setAccountLimits(query, input.MaxUsers, input.MaxRoles, input.MaxGroups, input.MaxAccessKeys, input.MaxBuckets)
	return client.accountRequest(ctx, http.MethodPut, query)
}

// SetAccountQuota modifies an account quota directly through
// PUT /admin/account?quota.
func (client *Client) SetAccountQuota(ctx context.Context, input SetAccountQuotaRequest) (Account, error) {
	if ctx == nil {
		return Account{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.ID) == "" {
		return Account{}, errors.New("rgw: account ID must not be empty")
	}
	if input.Scope != QuotaScopeAccount && input.Scope != QuotaScopeBucket {
		return Account{}, errors.New("rgw: account quota type must be account or bucket")
	}
	query := url.Values{
		"quota":      {""},
		"id":         {input.ID},
		"quota-type": {string(input.Scope)},
	}
	setInt64(query, "max-size", input.MaxSize)
	setInt64(query, "max-objects", input.MaxObjects)
	setBool(query, "enabled", input.Enabled)
	return client.accountRequest(ctx, http.MethodPut, query)
}

// DeleteAccount deletes an empty account directly through DELETE /admin/account.
func (client *Client) DeleteAccount(ctx context.Context, input DeleteAccountRequest) error {
	if ctx == nil {
		return errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.ID) == "" && strings.TrimSpace(input.Name) == "" {
		return errors.New("rgw: account ID or name must not be empty")
	}
	request, err := client.newRequest(ctx, http.MethodDelete, "account", accountIdentityQuery(input.ID, input.Tenant, input.Name))
	if err != nil {
		return err
	}
	_, err = client.do(request)
	return err
}

func accountIdentityQuery(id, tenant, name string) url.Values {
	query := url.Values{}
	setString(query, "id", id)
	setString(query, "tenant", tenant)
	setString(query, "name", name)
	return query
}

func setAccountLimits(query url.Values, maxUsers, maxRoles, maxGroups, maxAccessKeys, maxBuckets *int64) {
	setInt64(query, "max-users", maxUsers)
	setInt64(query, "max-roles", maxRoles)
	setInt64(query, "max-groups", maxGroups)
	setInt64(query, "max-access-keys", maxAccessKeys)
	setInt64(query, "max-buckets", maxBuckets)
}

func (client *Client) accountRequest(ctx context.Context, method string, query url.Values) (Account, error) {
	request, err := client.newRequest(ctx, method, "account", query)
	if err != nil {
		return Account{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return Account{}, err
	}
	var account Account
	if err := json.Unmarshal(body, &account); err != nil {
		return Account{}, fmt.Errorf("rgw: decode %s %s response: %w", method, request.URL.Path, err)
	}
	return account, nil
}
