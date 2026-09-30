package rgw

import (
	"context"
	"net/url"
)

// KeyValue is an entry in Ceph's JSON map representation: an array of objects
// with key and val fields, as encoded by src/common/ceph_json.h.
type KeyValue[T any] struct {
	Key   string `json:"key"`
	Value T      `json:"val"`
}

// ZoneConfiguration is RGWZoneParams for the zone serving the request.
// TierConfig is open-ended configuration interpreted by the zone's tier module.
//
// Verified against Ceph v20.2.4:
//   - src/rgw/rgw_rest_config.cc and .h (dispatch and zone=read)
//   - src/rgw/rgw_zone.cc (RGWZoneParams and placement formatters)
//   - src/rgw/driver/rados/rgw_zone.h and src/rgw/rgw_zone_types.h
//   - src/rgw/rgw_common.cc (pool strings and RGWAccessKey::dump_plain)
//   - src/rgw/rgw_bucket_layout.h (numeric BucketIndexType)
type ZoneConfiguration struct {
	ID                string                        `json:"id"`
	Name              string                        `json:"name"`
	DomainRoot        string                        `json:"domain_root"`
	ControlPool       string                        `json:"control_pool"`
	DedupPool         string                        `json:"dedup_pool"`
	GCPool            string                        `json:"gc_pool"`
	LCPool            string                        `json:"lc_pool"`
	LogPool           string                        `json:"log_pool"`
	IntentLogPool     string                        `json:"intent_log_pool"`
	UsageLogPool      string                        `json:"usage_log_pool"`
	RolesPool         string                        `json:"roles_pool"`
	ReshardPool       string                        `json:"reshard_pool"`
	UserKeysPool      string                        `json:"user_keys_pool"`
	UserEmailPool     string                        `json:"user_email_pool"`
	UserSwiftPool     string                        `json:"user_swift_pool"`
	UserUIDPool       string                        `json:"user_uid_pool"`
	OTPPool           string                        `json:"otp_pool"`
	NotificationPool  string                        `json:"notif_pool"`
	TopicsPool        string                        `json:"topics_pool"`
	AccountPool       string                        `json:"account_pool"`
	GroupPool         string                        `json:"group_pool"`
	BucketLoggingPool string                        `json:"bucket_logging_pool"`
	SystemKey         SystemKey                     `json:"system_key"`
	PlacementPools    []KeyValue[ZonePlacementPool] `json:"placement_pools"`
	TierConfig        map[string]any                `json:"tier_config"`
	RealmID           string                        `json:"realm_id"`
	RestorePool       string                        `json:"restore_pool"`
}

// SystemKey is RGWAccessKey's plain representation. It omits the user and
// active status included in the user endpoint's AccessKey representation.
type SystemKey struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// PlacementIndexType is the numeric index type in RGWZonePlacementInfo.
// Bucket statistics instead use the string-valued BucketIndexType.
type PlacementIndexType uint32

const (
	PlacementIndexNormal    PlacementIndexType = 0
	PlacementIndexIndexless PlacementIndexType = 1
)

type ZonePlacementPool struct {
	IndexPool      string                      `json:"index_pool"`
	StorageClasses map[string]ZoneStorageClass `json:"storage_classes"`
	DataExtraPool  string                      `json:"data_extra_pool"`
	IndexType      PlacementIndexType          `json:"index_type"`
	InlineData     bool                        `json:"inline_data"`
}

type ZoneStorageClass struct {
	DataPool        *string `json:"data_pool,omitempty"`
	CompressionType *string `json:"compression_type,omitempty"`
}

// GetZoneConfiguration retrieves the serving zone's configuration through
// GET /admin/config?type=zone. Requires zone=read.
func (client *Client) GetZoneConfiguration(ctx context.Context) (ZoneConfiguration, error) {
	var zone ZoneConfiguration
	if err := client.getConfiguration(ctx, "config", url.Values{"type": {"zone"}}, &zone); err != nil {
		return ZoneConfiguration{}, err
	}
	return zone, nil
}
