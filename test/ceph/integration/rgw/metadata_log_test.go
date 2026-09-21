package integration

import (
	"errors"
	"net/http"
	"testing"

	"github.com/sj14/rgw-go/rgw"
)

func TestRGWGetMetadataLogInfo(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).GetMetadataLogInfo(integrationContext(t))
	requireRGWAPIStatus(t, err, http.StatusNotFound)
}

func TestRGWGetMetadataLogShardInfo(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).GetMetadataLogShardInfo(
		integrationContext(t), rgw.GetMetadataLogShardInfoRequest{ShardID: 0},
	)
	requireRGWAPIStatus(t, err, http.StatusBadRequest)
}

func TestRGWListMetadataLogEntries(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).ListMetadataLogEntries(
		integrationContext(t), rgw.ListMetadataLogEntriesRequest{ShardID: 0, MaxEntries: new(int64(1))},
	)
	requireRGWAPIStatus(t, err, http.StatusBadRequest)
}

func TestRGWGetMetadataLogStatus(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).GetMetadataLogStatus(integrationContext(t))
	requireRGWAPIStatus(t, err, http.StatusNotFound)
}

func requireRGWAPIStatus(t *testing.T, err error, status int) {
	t.Helper()
	var apiError *rgw.APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != status {
		t.Fatalf("error = %v, want Admin Ops status %d", err, status)
	}
}
