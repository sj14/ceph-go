package integration

import (
	"net/http"
	"slices"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

// The minimal container runs without a realm, as do the metadata-log tests.
// Exercise successful lookups too when running against a realm-enabled cluster.
// Verified against Ceph v20.2.4's rgw_rest_realm.cc, rgw_realm.cc and rgw_period.cc.
func TestRGWGetRealm(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	list, err := client.ListRealms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Realms) == 0 {
		_, err := client.GetRealm(ctx, rgw.GetRealmRequest{})
		requireRGWAPIStatus(t, err, http.StatusNotFound)
	} else {
		byName, err := client.GetRealm(ctx, rgw.GetRealmRequest{Name: list.Realms[0]})
		if err != nil {
			t.Fatal(err)
		}
		if byName.ID == "" || byName.Name != list.Realms[0] || byName.CurrentPeriod == "" {
			t.Fatalf("realm = %#v", byName)
		}
		byID, err := client.GetRealm(ctx, rgw.GetRealmRequest{ID: byName.ID})
		if err != nil {
			t.Fatal(err)
		}
		if byID != byName {
			t.Fatalf("realm by ID = %#v, want %#v", byID, byName)
		}
	}
	_, err = client.GetRealm(ctx, rgw.GetRealmRequest{Name: uniqueResourceName(t, "missing-realm")})
	requireRGWAPIStatus(t, err, http.StatusNotFound)
}

func TestRGWListRealms(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	list, err := client.ListRealms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if list.Realms == nil {
		t.Fatal("realm list omitted the realms array")
	}
	if list.DefaultID != "" {
		defaultRealm, err := client.GetRealm(ctx, rgw.GetRealmRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if defaultRealm.ID != list.DefaultID || !slices.Contains(list.Realms, defaultRealm.Name) {
			t.Fatalf("default realm = %#v, list = %#v", defaultRealm, list)
		}
	}
}

func TestRGWGetPeriod(t *testing.T) {
	t.Parallel()

	client := rgwIntegrationClient(t)
	ctx := integrationContext(t)
	list, err := client.ListRealms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Realms) == 0 {
		_, err := client.GetPeriod(ctx, rgw.GetPeriodRequest{})
		requireRGWAPIStatus(t, err, http.StatusNotFound)
	} else {
		realm, err := client.GetRealm(ctx, rgw.GetRealmRequest{Name: list.Realms[0]})
		if err != nil {
			t.Fatal(err)
		}
		current, err := client.GetPeriod(ctx, rgw.GetPeriodRequest{RealmID: realm.ID})
		if err != nil {
			t.Fatal(err)
		}
		if current.ID != realm.CurrentPeriod || current.RealmID != realm.ID || current.Epoch == 0 {
			t.Fatalf("current period = %#v, realm = %#v", current, realm)
		}
		explicit, err := client.GetPeriod(ctx, rgw.GetPeriodRequest{
			PeriodID: current.ID, Epoch: current.Epoch,
		})
		if err != nil {
			t.Fatal(err)
		}
		if explicit.ID != current.ID || explicit.Epoch != current.Epoch || explicit.RealmID != realm.ID {
			t.Fatalf("explicit period = %#v, want period %s epoch %d", explicit, current.ID, current.Epoch)
		}
	}
	_, err = client.GetPeriod(ctx, rgw.GetPeriodRequest{PeriodID: uniqueResourceName(t, "missing-period")})
	requireRGWAPIStatus(t, err, http.StatusNotFound)
}
