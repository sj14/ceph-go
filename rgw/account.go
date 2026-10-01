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

// GetAccountRequest identifies an account by its required ID.
type GetAccountRequest struct {
	ID string
}

// GetAccount retrieves an account by ID through GET /admin/account.
// Verified against Ceph v20.2.4's src/rgw/rgw_rest_account.cc
// (RGWOp_Account_Get) and src/rgw/rgw_account.cc (info).
func (client *Client) GetAccount(ctx context.Context, input GetAccountRequest) (Account, error) {
	query, err := accountLookupQuery(ctx, "id", input.ID, "")
	if err != nil {
		return Account{}, err
	}
	return client.accountRequest(ctx, http.MethodGet, query)
}

// GetAccountByNameRequest identifies an account by its required Name within
// Tenant. Empty Tenant selects the default tenant.
type GetAccountByNameRequest struct {
	Tenant string
	Name   string
}

// GetAccountByName retrieves an account by tenant and name through
// GET /admin/account?name=....
func (client *Client) GetAccountByName(ctx context.Context, input GetAccountByNameRequest) (Account, error) {
	query, err := accountLookupQuery(ctx, "name", input.Name, input.Tenant)
	if err != nil {
		return Account{}, err
	}
	return client.accountRequest(ctx, http.MethodGet, query)
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

// UpdateAccountRequest requires the target account's ID. Name and Email are
// update values; empty strings preserve stored values and cannot clear them.
// Nil limits preserve stored values. The account's tenant cannot be changed.
type UpdateAccountRequest struct {
	ID            string
	Name          string
	Email         string
	MaxUsers      *int64
	MaxRoles      *int64
	MaxGroups     *int64
	MaxAccessKeys *int64
	MaxBuckets    *int64
}

// UpdateAccount modifies an account directly through PUT /admin/account.
// Verified against Ceph v20.2.4's src/rgw/rgw_rest_account.cc and
// src/rgw/rgw_account.cc (modify): requiring ID avoids falling back to name or
// email lookup when those fields are intended as updates.
func (client *Client) UpdateAccount(ctx context.Context, input UpdateAccountRequest) (Account, error) {
	if ctx == nil {
		return Account{}, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(input.ID) == "" {
		return Account{}, errors.New("rgw: account ID must not be empty")
	}
	query := url.Values{"id": {input.ID}}
	setString(query, "name", input.Name)
	setString(query, "email", input.Email)
	setAccountLimits(query, input.MaxUsers, input.MaxRoles, input.MaxGroups, input.MaxAccessKeys, input.MaxBuckets)
	return client.accountRequest(ctx, http.MethodPut, query)
}

// SetAccountQuotaRequest modifies either the aggregate account quota or the
// per-bucket quota inherited by buckets in the account. Limits use int32 because
// Ceph v20.2.4's src/rgw/rgw_rest_account.cc parses both with get_int32, even
// though the stored RGWQuotaInfo and response fields use int64. MaxSize is bytes.
type SetAccountQuotaRequest struct {
	ID         string
	Scope      QuotaScope
	MaxSize    *int32
	MaxObjects *int32
	Enabled    *bool
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
	setInt32(query, "max-size", input.MaxSize)
	setInt32(query, "max-objects", input.MaxObjects)
	setBool(query, "enabled", input.Enabled)
	return client.accountRequest(ctx, http.MethodPut, query)
}

// DeleteAccountRequest identifies an account by its required ID.
// Ceph v20.2.4 only deletes an empty account through this REST endpoint.
type DeleteAccountRequest struct {
	ID string
}

// DeleteAccount deletes an empty account by ID through DELETE /admin/account.
// Verified against Ceph v20.2.4's src/rgw/rgw_rest_account.cc
// (RGWOp_Account_Delete) and src/rgw/rgw_account.cc (remove).
func (client *Client) DeleteAccount(ctx context.Context, input DeleteAccountRequest) error {
	query, err := accountLookupQuery(ctx, "id", input.ID, "")
	if err != nil {
		return err
	}
	return client.deleteAccount(ctx, query)
}

// DeleteAccountByNameRequest identifies an empty account by its required Name
// within Tenant. Empty Tenant selects the default tenant.
type DeleteAccountByNameRequest struct {
	Tenant string
	Name   string
}

// DeleteAccountByName deletes an empty account by tenant and name through
// DELETE /admin/account?name=....
func (client *Client) DeleteAccountByName(ctx context.Context, input DeleteAccountByNameRequest) error {
	query, err := accountLookupQuery(ctx, "name", input.Name, input.Tenant)
	if err != nil {
		return err
	}
	return client.deleteAccount(ctx, query)
}

func accountLookupQuery(ctx context.Context, selector, value, tenant string) (url.Values, error) {
	if ctx == nil {
		return nil, errors.New("rgw: context must not be nil")
	}
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("rgw: account %s must not be empty", selector)
	}
	query := url.Values{selector: {value}}
	setString(query, "tenant", tenant)
	return query, nil
}

func (client *Client) deleteAccount(ctx context.Context, query url.Values) error {
	request, err := client.newRequest(ctx, http.MethodDelete, "account", query)
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
