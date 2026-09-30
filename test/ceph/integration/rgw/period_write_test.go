package integration

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

// Period mutations can change realm topology and trigger a gateway reload.
// Exercise the source-verified realm mismatch rejection before any writes.
func TestRGWPushPeriod(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).PushPeriod(integrationContext(t), rgw.Period{
		ID: uniqueResourceName(t, "period"), RealmID: uniqueResourceName(t, "unrelated-realm"),
	})
	requirePeriodRealmMismatch(t, err)
}

func TestRGWCommitPeriod(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).CommitPeriod(integrationContext(t), rgw.Period{
		RealmID: uniqueResourceName(t, "unrelated-realm"),
	})
	requirePeriodRealmMismatch(t, err)
}

func requirePeriodRealmMismatch(t *testing.T, err error) {
	t.Helper()
	requireRGWAPIStatus(t, err, http.StatusBadRequest)
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || !strings.Contains(apiError.Body, "doesn't match current realm") {
		t.Fatalf("period error = %v, want realm mismatch rejection", err)
	}
}
