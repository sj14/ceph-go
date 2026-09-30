package integration

import (
	"testing"

	"github.com/sj14/ceph-go/rgw"
)

func TestRGWGetZoneConfiguration(t *testing.T) {
	t.Parallel()

	zone, err := rgwIntegrationClient(t).GetZoneConfiguration(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if zone.ID == "" || zone.Name == "" || zone.DomainRoot == "" || zone.UserUIDPool == "" {
		t.Fatalf("zone configuration missing identity or pools: id=%q name=%q domain_root=%q user_uid_pool=%q",
			zone.ID, zone.Name, zone.DomainRoot, zone.UserUIDPool)
	}
	if len(zone.PlacementPools) == 0 {
		t.Fatal("zone configuration has no placement pools")
	}
	for _, placement := range zone.PlacementPools {
		if placement.Key == "" || placement.Value.IndexPool == "" {
			t.Fatalf("zone placement pool = %#v", placement)
		}
		standard, ok := placement.Value.StorageClasses["STANDARD"]
		if !ok || standard.DataPool == nil || *standard.DataPool == "" {
			t.Fatalf("zone placement storage classes = %#v", placement.Value.StorageClasses)
		}
		if placement.Value.IndexType != rgw.PlacementIndexNormal {
			t.Fatalf("zone placement index type = %d", placement.Value.IndexType)
		}
	}
}
