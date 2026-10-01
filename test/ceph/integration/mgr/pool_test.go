package integration

import (
	"testing"

	mgr "github.com/sj14/ceph-go/mgr"
)

// The manager creates .mgr as its built-in metadata pool. Using it keeps all
// pool endpoint tests read-only and avoids creating another cluster resource.
const testPoolName = ".mgr"

func TestListPools(t *testing.T) {
	t.Parallel()

	for _, stats := range []bool{false, true} {
		name := "without statistics"
		if stats {
			name = "with statistics"
		}
		t.Run(name, func(t *testing.T) {
			pools, err := integrationClient(t).ListPools(integrationContext(t), mgr.ListPoolsRequest{Stats: stats})
			if err != nil {
				t.Fatal(err)
			}
			for _, pool := range pools {
				if pool.Name == testPoolName {
					checkPoolFields(t, pool, stats)
					if pool.Configuration != nil || pool.ScheduleInfo != nil {
						t.Fatalf("list unexpectedly contains pool detail fields: %#v", pool)
					}
					return
				}
			}
			t.Fatalf("default pool %q is missing from ListPools", testPoolName)
		})
	}
}

func TestGetPool(t *testing.T) {
	t.Parallel()

	for _, stats := range []bool{false, true} {
		name := "without statistics"
		if stats {
			name = "with statistics"
		}
		t.Run(name, func(t *testing.T) {
			pool, err := integrationClient(t).GetPool(integrationContext(t), mgr.GetPoolRequest{
				Name:  testPoolName,
				Stats: stats,
			})
			if err != nil {
				t.Fatal(err)
			}
			checkPoolFields(t, pool, stats)
			if pool.Configuration == nil || pool.ScheduleInfo == nil {
				t.Fatalf("pool detail fields missing: %#v", pool)
			}
		})
	}
}

func checkPoolFields(t *testing.T, pool mgr.Pool, stats bool) {
	t.Helper()

	if pool.Name != testPoolName || pool.Type == "" || pool.Size <= 0 ||
		pool.MinSize <= 0 || pool.PGNum <= 0 || pool.CrushRule == "" {
		t.Fatalf("pool = %#v", pool)
	}
	if (pool.Stats != nil) != stats || (pool.PGStatus != nil) != stats {
		t.Fatalf("pool statistics presence does not match Stats=%t: %#v", stats, pool)
	}
}

func TestGetPoolConfiguration(t *testing.T) {
	t.Parallel()

	configuration, err := integrationClient(t).GetPoolConfiguration(
		integrationContext(t),
		mgr.GetPoolConfigurationRequest{Name: testPoolName},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range configuration {
		if entry.Name == "" || entry.Source < mgr.RBDConfigurationSourceGlobal ||
			entry.Source > mgr.RBDConfigurationSourceImage {
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
