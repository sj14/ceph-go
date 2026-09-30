package rgw

// Period contains a realm's versioned multisite configuration.
//
// Verified against Ceph v20.2.4:
//   - src/rgw/driver/rados/rgw_rest_realm.cc (GET parameters and response)
//   - src/rgw/rgw_period.cc (RGWPeriod::init and dump)
//   - src/rgw/rgw_zone.cc (period map/config, zonegroups and placement tiers)
//   - src/rgw/rgw_zone_types.h and src/rgw/rgw_zone_features.h
//   - src/rgw/rgw_sync_policy.cc and .h (sync policies and pipe formatters)
//   - src/rgw/rgw_basic_types.cc (zone IDs and user IDs encode as strings)
type Period struct {
	ID              string              `json:"id"`
	Epoch           uint32              `json:"epoch"`
	PredecessorUUID string              `json:"predecessor_uuid"`
	SyncStatus      []string            `json:"sync_status"`
	Map             PeriodMap           `json:"period_map"`
	MasterZonegroup string              `json:"master_zonegroup"`
	MasterZone      string              `json:"master_zone"`
	Configuration   PeriodConfiguration `json:"period_config"`
	RealmID         string              `json:"realm_id"`
	RealmEpoch      uint32              `json:"realm_epoch"`
}

type PeriodMap struct {
	ID           string             `json:"id"`
	Zonegroups   []Zonegroup        `json:"zonegroups"`
	ShortZoneIDs []KeyValue[uint32] `json:"short_zone_ids"`
}

type PeriodConfiguration struct {
	BucketQuota        Quota     `json:"bucket_quota"`
	UserQuota          Quota     `json:"user_quota"`
	UserRateLimit      RateLimit `json:"user_ratelimit"`
	BucketRateLimit    RateLimit `json:"bucket_ratelimit"`
	AnonymousRateLimit RateLimit `json:"anonymous_ratelimit"`
}

// ZoneFeature identifies a feature supported by a zone or enabled on a
// zonegroup. Unknown feature names can be explicitly converted to this type.
type ZoneFeature string

const (
	ZoneFeatureResharding        ZoneFeature = "resharding"
	ZoneFeatureCompressEncrypted ZoneFeature = "compress-encrypted"
	ZoneFeatureNotificationV2    ZoneFeature = "notification_v2"
)

type Zonegroup struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	APIName          string            `json:"api_name"`
	IsMaster         bool              `json:"is_master"`
	Endpoints        []string          `json:"endpoints"`
	Hostnames        []string          `json:"hostnames"`
	WebsiteHostnames []string          `json:"hostnames_s3website"`
	MasterZone       string            `json:"master_zone"`
	Zones            []Zone            `json:"zones"`
	PlacementTargets []PlacementTarget `json:"placement_targets"`
	DefaultPlacement string            `json:"default_placement"`
	RealmID          string            `json:"realm_id"`
	SyncPolicy       SyncPolicy        `json:"sync_policy"`
	EnabledFeatures  []ZoneFeature     `json:"enabled_features"`
}

// Zone is RGWZone, the zonegroup's description of a member zone. TierType is
// an open-ended module name, distinct from placement tier types.
type Zone struct {
	ID                   string        `json:"id"`
	Name                 string        `json:"name"`
	Endpoints            []string      `json:"endpoints"`
	LogMetadata          bool          `json:"log_meta"`
	LogData              bool          `json:"log_data"`
	BucketIndexMaxShards uint32        `json:"bucket_index_max_shards"`
	ReadOnly             bool          `json:"read_only"`
	TierType             string        `json:"tier_type"`
	SyncFromAll          bool          `json:"sync_from_all"`
	SyncFrom             []string      `json:"sync_from"`
	RedirectZone         string        `json:"redirect_zone"`
	SupportedFeatures    []ZoneFeature `json:"supported_features"`
}

type PlacementTarget struct {
	Name           string                    `json:"name"`
	Tags           []string                  `json:"tags"`
	StorageClasses []string                  `json:"storage_classes"`
	TierTargets    []KeyValue[PlacementTier] `json:"tier_targets,omitempty"`
}

type PlacementTierType string

const (
	PlacementTierCloudS3        PlacementTierType = "cloud-s3"
	PlacementTierCloudS3Glacier PlacementTierType = "cloud-s3-glacier"
)

type PlacementTier struct {
	Type                   PlacementTierType     `json:"tier_type"`
	StorageClass           string                `json:"storage_class"`
	RetainHeadObject       bool                  `json:"retain_head_object"`
	S3                     *PlacementTierS3      `json:"s3,omitempty"`
	AllowReadThrough       bool                  `json:"allow_read_through"`
	ReadThroughRestoreDays uint64                `json:"read_through_restore_days"`
	RestoreStorageClass    string                `json:"restore_storage_class"`
	Glacier                *PlacementTierGlacier `json:"s3-glacier,omitempty"`
}

type HostStyle string

const (
	HostStylePath    HostStyle = "path"
	HostStyleVirtual HostStyle = "virtual"
)

type PlacementTierS3 struct {
	Endpoint               string                     `json:"endpoint"`
	AccessKey              string                     `json:"access_key"`
	Secret                 string                     `json:"secret"`
	Region                 string                     `json:"region"`
	HostStyle              HostStyle                  `json:"host_style"`
	TargetStorageClass     string                     `json:"target_storage_class"`
	TargetPath             string                     `json:"target_path"`
	ACLMappings            []KeyValue[TierACLMapping] `json:"acl_mappings"`
	MultipartSyncThreshold uint64                     `json:"multipart_sync_threshold"`
	MultipartMinPartSize   uint64                     `json:"multipart_min_part_size"`
}

type TierACLMappingType string

const (
	TierACLMappingID    TierACLMappingType = "id"
	TierACLMappingEmail TierACLMappingType = "email"
	TierACLMappingURI   TierACLMappingType = "uri"
)

type TierACLMapping struct {
	Type          TierACLMappingType `json:"type"`
	SourceID      string             `json:"source_id"`
	DestinationID string             `json:"dest_id"`
}

type GlacierRestoreTier string

const (
	GlacierRestoreStandard  GlacierRestoreTier = "Standard"
	GlacierRestoreExpedited GlacierRestoreTier = "Expedited"
)

type PlacementTierGlacier struct {
	RestoreDays uint64             `json:"glacier_restore_days"`
	RestoreTier GlacierRestoreTier `json:"glacier_restore_tier_type"`
}

type SyncPolicy struct {
	Groups []SyncPolicyGroup `json:"groups"`
}

type SyncPolicyStatus string

const (
	SyncPolicyForbidden SyncPolicyStatus = "forbidden"
	SyncPolicyAllowed   SyncPolicyStatus = "allowed"
	SyncPolicyEnabled   SyncPolicyStatus = "enabled"
	SyncPolicyUnknown   SyncPolicyStatus = "unknown"
)

type SyncPolicyGroup struct {
	ID       string           `json:"id"`
	DataFlow SyncDataFlow     `json:"data_flow"`
	Pipes    []SyncPipe       `json:"pipes"`
	Status   SyncPolicyStatus `json:"status"`
}

type SyncDataFlow struct {
	Symmetrical []SyncSymmetricalFlow `json:"symmetrical,omitempty"`
	Directional []SyncDirectionalFlow `json:"directional,omitempty"`
}

type SyncSymmetricalFlow struct {
	ID    string   `json:"id"`
	Zones []string `json:"zones"`
}

type SyncDirectionalFlow struct {
	SourceZone      string `json:"source_zone"`
	DestinationZone string `json:"dest_zone"`
}

type SyncPipe struct {
	ID          string             `json:"id"`
	Source      SyncBucketEntities `json:"source"`
	Destination SyncBucketEntities `json:"dest"`
	Parameters  SyncPipeParameters `json:"params"`
}

// SyncBucketEntities selects buckets and zones, with "*" representing all.
// An absent Zones field is distinct from an explicit empty list.
type SyncBucketEntities struct {
	Bucket string    `json:"bucket"`
	Zones  *[]string `json:"zones,omitempty"`
}

type SyncPipeMode string

const (
	SyncPipeModeSystem SyncPipeMode = "system"
	SyncPipeModeUser   SyncPipeMode = "user"
)

type SyncPipeParameters struct {
	Source      SyncPipeSource      `json:"source"`
	Destination SyncPipeDestination `json:"dest"`
	Priority    int32               `json:"priority"`
	Mode        SyncPipeMode        `json:"mode"`
	User        *string             `json:"user,omitempty"`
}

type SyncPipeSource struct {
	Filter SyncPipeFilter `json:"filter"`
}

type SyncPipeFilter struct {
	Prefix *string         `json:"prefix,omitempty"`
	Tags   []SyncFilterTag `json:"tags"`
}

// SyncFilterTag uses value, whereas user metadata's Tag uses val.
type SyncFilterTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SyncPipeDestination struct {
	ACLTranslation *SyncACLTranslation `json:"acl_translation,omitempty"`
	StorageClass   *string             `json:"storage_class,omitempty"`
}

type SyncACLTranslation struct {
	Owner string `json:"owner"`
}
