package integration

import (
	"net/http"
	"testing"

	"github.com/sj14/rgw-go/admin"
)

func TestAdminGetDataLogInfo(t *testing.T) {
	t.Parallel()

	info, err := adminIntegrationClient(t).GetDataLogInfo(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if info.NumShards <= 0 {
		t.Fatalf("data log info = %#v", info)
	}
}

func TestAdminGetDataLogShardInfo(t *testing.T) {
	t.Parallel()

	info, err := adminIntegrationClient(t).GetDataLogShardInfo(
		integrationContext(t), admin.GetDataLogShardInfoRequest{ShardID: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if info.LastUpdate == "" {
		t.Fatalf("data log shard info = %#v", info)
	}
}

func TestAdminListDataLogEntries(t *testing.T) {
	t.Parallel()

	entries, err := adminIntegrationClient(t).ListDataLogEntries(
		integrationContext(t), admin.ListDataLogEntriesRequest{
			ShardID: 0, MaxEntries: new(int64(10)), ExtraInfo: new(true),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if entries.LastUpdate == "" {
		t.Fatalf("data log entries = %#v", entries)
	}
}

func TestAdminGetDataLogStatus(t *testing.T) {
	t.Parallel()

	_, err := adminIntegrationClient(t).GetDataLogStatus(
		integrationContext(t), admin.GetDataLogStatusRequest{},
	)
	requireAdminAPIStatus(t, err, http.StatusNotFound)
}
