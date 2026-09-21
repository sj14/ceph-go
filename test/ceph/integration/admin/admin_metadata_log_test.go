package integration

import (
	"errors"
	"net/http"
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminGetMetadataLogInfo(t *testing.T) {
	t.Parallel()

	_, err := adminIntegrationClient(t).GetMetadataLogInfo(integrationContext(t))
	requireAdminAPIStatus(t, err, http.StatusNotFound)
}

func TestAdminGetMetadataLogShardInfo(t *testing.T) {
	t.Parallel()

	_, err := adminIntegrationClient(t).GetMetadataLogShardInfo(
		integrationContext(t), admin.GetMetadataLogShardInfoRequest{ShardID: 0},
	)
	requireAdminAPIStatus(t, err, http.StatusBadRequest)
}

func TestAdminListMetadataLogEntries(t *testing.T) {
	t.Parallel()

	_, err := adminIntegrationClient(t).ListMetadataLogEntries(
		integrationContext(t), admin.ListMetadataLogEntriesRequest{ShardID: 0, MaxEntries: new(int64(1))},
	)
	requireAdminAPIStatus(t, err, http.StatusBadRequest)
}

func TestAdminGetMetadataLogStatus(t *testing.T) {
	t.Parallel()

	_, err := adminIntegrationClient(t).GetMetadataLogStatus(integrationContext(t))
	requireAdminAPIStatus(t, err, http.StatusNotFound)
}

func requireAdminAPIStatus(t *testing.T, err error, status int) {
	t.Helper()
	var apiError *admin.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != status {
		t.Fatalf("error = %v, want Admin Ops status %d", err, status)
	}
}
