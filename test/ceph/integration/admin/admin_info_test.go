package integration

import "testing"

func TestAdminGetGatewayInfo(t *testing.T) {
	t.Parallel()

	info, err := adminIntegrationClient(t).GetGatewayInfo(integrationContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(info.StorageBackends) != 1 || info.StorageBackends[0].Name != "rados" ||
		info.StorageBackends[0].ClusterID == "" {
		t.Fatalf("Admin Ops gateway info = %#v", info)
	}
}
