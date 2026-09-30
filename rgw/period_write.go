package rgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// PushPeriod pushes a period to a non-master zone through
// POST /admin/realm/period. The period must have a non-empty ID and its RealmID
// must match the serving gateway's realm. Requires zone=write.
//
// Verified against Ceph v20.2.4: src/rgw/driver/rados/rgw_rest_realm.cc
// (RGWOp_Period_Post), src/rgw/rgw_period.cc, and
// src/rgw/driver/rados/rgw_period.cc (commit, reflect and history updates).
func (client *Client) PushPeriod(ctx context.Context, input Period) (Period, error) {
	if strings.TrimSpace(input.ID) == "" {
		return Period{}, errors.New("rgw: pushed period ID must not be empty")
	}
	return client.postPeriod(ctx, input)
}

// CommitPeriod commits a proposed period through POST /admin/realm/period.
// Input.ID must be empty, which selects Ceph's commit behavior. The proposed
// period's RealmID must match the serving gateway's realm. Requires zone=write.
// Ceph returns the committed period and then notifies the realm to reload.
func (client *Client) CommitPeriod(ctx context.Context, input Period) (Period, error) {
	if input.ID != "" {
		return Period{}, errors.New("rgw: committed period input ID must be empty")
	}
	return client.postPeriod(ctx, input)
}

func (client *Client) postPeriod(ctx context.Context, input Period) (Period, error) {
	request, err := client.newJSONRequest(ctx, http.MethodPost, "realm/period", nil, input)
	if err != nil {
		return Period{}, err
	}
	body, err := client.do(request)
	if err != nil {
		return Period{}, err
	}
	var period Period
	if err := json.Unmarshal(body, &period); err != nil {
		return Period{}, fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return period, nil
}
