package integration

import (
	"testing"

	rgw "github.com/sj14/rgw-go/dashboard"
)

// The manager creates .mgr as its built-in metadata pool. Using it keeps all
// pool endpoint tests read-only and avoids creating another cluster resource.
const testPoolName = ".mgr"

func TestListPools(t *testing.T) {
	t.Parallel()

	pools, err := integrationClient(t).ListPools(integrationContext(t), rgw.ListPoolsRequest{Stats: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, pool := range pools {
		if pool.Name == testPoolName {
			if pool.Type == "" || pool.Stats == nil || pool.PGStatus == nil {
				t.Fatalf("listed pool = %#v", pool)
			}
			return
		}
	}
	t.Fatalf("default pool %q is missing from ListPools", testPoolName)
}

func TestGetPool(t *testing.T) {
	t.Parallel()

	pool, err := integrationClient(t).GetPool(integrationContext(t), rgw.GetPoolRequest{
		Name:       testPoolName,
		Attributes: []string{"type", "flags", "stats"},
		Stats:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pool.Name != testPoolName || pool.Type == "" || pool.Stats == nil ||
		pool.Configuration == nil || pool.ScheduleInfo == nil {
		t.Fatalf("pool = %#v", pool)
	}
}

func TestGetPoolConfiguration(t *testing.T) {
	t.Parallel()

	configuration, err := integrationClient(t).GetPoolConfiguration(
		integrationContext(t),
		rgw.GetPoolConfigurationRequest{Name: testPoolName},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range configuration {
		if entry.Name == "" || entry.Source < rgw.RBDConfigurationSourceGlobal ||
			entry.Source > rgw.RBDConfigurationSourceImage {
			t.Fatalf("configuration entry = %#v", entry)
		}
	}
}

func TestGetPoolInfo(t *testing.T) {
	t.Parallel()

	info, err := integrationClient(t).GetPoolInfo(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if info.OSDCount != 1 || len(info.PoolNames) == 0 ||
		len(info.CrushRulesReplicated) == 0 || len(info.Nodes) == 0 ||
		len(info.CompressionAlgorithms) == 0 || len(info.CompressionModes) == 0 ||
		len(info.PGAutoscaleModes) == 0 {
		t.Fatalf("pool info = %#v", info)
	}
}
