package rgw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Realm is Ceph's RGWRealm configuration.
//
// Verified against Ceph v20.2.4:
//   - src/rgw/driver/rados/rgw_rest_realm.cc (dispatch, parameters and zone=read)
//   - src/rgw/rgw_realm.cc (RGWRealm::dump and default realm resolution)
//   - src/rgw/rgw_zone.cc (RGWSystemMetaObj::dump)
type Realm struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CurrentPeriod string `json:"current_period"`
	Epoch         uint32 `json:"epoch"`
}

// RealmList contains realm names and the ID of the default realm. Names are
// returned without fetching each realm's configuration.
type RealmList struct {
	DefaultID string   `json:"default_info"`
	Realms    []string `json:"realms"`
}

type GetRealmRequest struct {
	ID   string
	Name string
}

// GetRealm reads a realm through GET /admin/realm. ID takes precedence over
// Name; omitting both selects Ceph's default realm. Requires zone=read.
func (client *Client) GetRealm(ctx context.Context, input GetRealmRequest) (Realm, error) {
	query := url.Values{}
	setString(query, "id", input.ID)
	setString(query, "name", input.Name)
	var realm Realm
	if err := client.getConfiguration(ctx, "realm", query, &realm); err != nil {
		return Realm{}, err
	}
	return realm, nil
}

// ListRealms reads realm names and the default realm ID through
// GET /admin/realm?list. Requires zone=read.
func (client *Client) ListRealms(ctx context.Context) (RealmList, error) {
	var realms RealmList
	if err := client.getConfiguration(ctx, "realm", url.Values{"list": {""}}, &realms); err != nil {
		return RealmList{}, err
	}
	return realms, nil
}

type GetPeriodRequest struct {
	RealmID  string
	PeriodID string
	// Epoch zero selects the latest epoch.
	Epoch uint32
}

// GetPeriod reads a period through GET /admin/realm/period. An empty PeriodID
// selects the realm's current period, using the default realm when RealmID is
// also empty. An explicit PeriodID can be read without RealmID. Requires zone=read.
func (client *Client) GetPeriod(ctx context.Context, input GetPeriodRequest) (Period, error) {
	query := url.Values{}
	setString(query, "realm_id", input.RealmID)
	setString(query, "period_id", input.PeriodID)
	if input.Epoch != 0 {
		query.Set("epoch", strconv.FormatUint(uint64(input.Epoch), 10))
	}
	var period Period
	if err := client.getConfiguration(ctx, "realm/period", query, &period); err != nil {
		return Period{}, err
	}
	return period, nil
}

func (client *Client) getConfiguration(ctx context.Context, resource string, query url.Values, output any) error {
	request, err := client.newRequest(ctx, http.MethodGet, resource, query)
	if err != nil {
		return err
	}
	body, err := client.do(request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, output); err != nil {
		return fmt.Errorf("rgw: decode %s %s response: %w", request.Method, request.URL.Path, err)
	}
	return nil
}
