package integration

import (
	"net/http"
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetDataLogInfo(t *testing.T) {
	t.Parallel()

	info, err := rgwIntegrationClient(t).GetDataLogInfo(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if info.NumShards <= 0 {
		t.Fatalf("data log info = %#v", info)
	}
}

func TestRGWGetDataLogShardInfo(t *testing.T) {
	t.Parallel()

	info, err := rgwIntegrationClient(t).GetDataLogShardInfo(
		integrationContext(t), rgw.GetDataLogShardInfoRequest{ShardID: 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	if info.LastUpdate == "" {
		t.Fatalf("data log shard info = %#v", info)
	}
}

func TestRGWListDataLogEntries(t *testing.T) {
	t.Parallel()

	entries, err := rgwIntegrationClient(t).ListDataLogEntries(
		integrationContext(t), rgw.ListDataLogEntriesRequest{
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

func TestRGWGetDataLogStatus(t *testing.T) {
	t.Parallel()

	_, err := rgwIntegrationClient(t).GetDataLogStatus(
		integrationContext(t), rgw.GetDataLogStatusRequest{},
	)
	requireRGWAPIStatus(t, err, http.StatusNotFound)
}
