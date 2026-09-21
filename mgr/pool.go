package mgr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PoolType is the serialized Ceph pool type.
type PoolType string

const (
	PoolTypeReplicated PoolType = "replicated"
	PoolTypeErasure    PoolType = "erasure"
)

// PoolPGAutoscaleMode controls automatic placement-group scaling.
type PoolPGAutoscaleMode string

const (
	PoolPGAutoscaleModeOff  PoolPGAutoscaleMode = "off"
	PoolPGAutoscaleModeOn   PoolPGAutoscaleMode = "on"
	PoolPGAutoscaleModeWarn PoolPGAutoscaleMode = "warn"
)

// PoolCacheMode is the cache-tier behavior serialized by Ceph's pg_pool_t.
type PoolCacheMode string

const (
	PoolCacheModeNone        PoolCacheMode = "none"
	PoolCacheModeWriteback   PoolCacheMode = "writeback"
	PoolCacheModeForward     PoolCacheMode = "forward"
	PoolCacheModeReadonly    PoolCacheMode = "readonly"
	PoolCacheModeReadForward PoolCacheMode = "readforward"
	PoolCacheModeReadProxy   PoolCacheMode = "readproxy"
	PoolCacheModeProxy       PoolCacheMode = "proxy"
)

// PoolHitSetType is the cache hit-set implementation serialized by Ceph.
type PoolHitSetType string

const (
	PoolHitSetTypeNone           PoolHitSetType = "none"
	PoolHitSetTypeExplicitHash   PoolHitSetType = "explicit_hash"
	PoolHitSetTypeExplicitObject PoolHitSetType = "explicit_object"
	PoolHitSetTypeBloom          PoolHitSetType = "bloom"
)

// BlueStoreCompressionMode controls when BlueStore attempts compression.
type BlueStoreCompressionMode string

const (
	BlueStoreCompressionModeNone       BlueStoreCompressionMode = "none"
	BlueStoreCompressionModePassive    BlueStoreCompressionMode = "passive"
	BlueStoreCompressionModeAggressive BlueStoreCompressionMode = "aggressive"
	BlueStoreCompressionModeForce      BlueStoreCompressionMode = "force"
)

// RBDConfigurationSource identifies the level from which an RBD option is
// inherited. These integer values come from librbd's configuration source
// enum and are shared by pool and image configuration responses.
type RBDConfigurationSource int64

const (
	RBDConfigurationSourceGlobal RBDConfigurationSource = 0
	RBDConfigurationSourcePool   RBDConfigurationSource = 1
	RBDConfigurationSourceImage  RBDConfigurationSource = 2
)

// ListPoolsRequest controls the optional fields and statistics returned by
// Ceph. Attributes use the names from Pool's JSON tags. When Attributes is
// empty, Ceph returns all ordinary pool fields.
type ListPoolsRequest struct {
	Attributes []string
	Stats      bool
}

// GetPoolRequest identifies a pool and controls its optional response fields.
// Ceph always returns Name and Configuration even when Attributes filters the
// ordinary pool fields.
type GetPoolRequest struct {
	Name       string
	Attributes []string
	Stats      bool
}

// GetPoolConfigurationRequest identifies the pool whose effective RBD
// configuration should be returned.
type GetPoolConfigurationRequest struct {
	Name string
}

// PoolStat contains the latest value and calculated rates for one dynamically
// named pool statistic. Each Rates entry is a timestamp/value pair produced by
// Ceph's mgr_util.get_time_series_rates.
type PoolStat struct {
	Latest float64     `json:"latest"`
	Rate   float64     `json:"rate"`
	Rates  [][]float64 `json:"rates"`
}

// PoolStats maps Ceph's open-ended statistic names to their measurements.
type PoolStats map[string]PoolStat

// PoolLastPGMergeMeta describes the most recent placement-group merge.
type PoolLastPGMergeMeta struct {
	ReadyEpoch       int64  `json:"ready_epoch"`
	LastEpochStarted int64  `json:"last_epoch_started"`
	LastEpochClean   int64  `json:"last_epoch_clean"`
	SourcePGID       string `json:"source_pgid"`
	SourceVersion    string `json:"source_version"`
	TargetVersion    string `json:"target_version"`
}

// PoolHitSetParameters contains the hit-set algorithm selected for a cache
// pool. The remaining values are present for Bloom hit sets.
type PoolHitSetParameters struct {
	Type                     PoolHitSetType `json:"type"`
	FalsePositiveProbability *float64       `json:"false_positive_probability,omitempty"`
	TargetSize               *int64         `json:"target_size,omitempty"`
	Seed                     *int64         `json:"seed,omitempty"`
}

// PoolScheduleInfo describes the RBD mirroring snapshot schedule inherited by
// or configured for a pool. An empty object means no schedule exists.
type PoolScheduleInfo struct {
	Name             string  `json:"name"`
	ScheduleInterval string  `json:"schedule_interval"`
	ScheduleTime     *string `json:"schedule_time"`
	Inherited        *string `json:"inherited"`
}

// RBDConfigurationEntry is one effective librbd configuration value. This
// model is shared with future image-level endpoints because both originate
// from librbd's config_list response.
type RBDConfigurationEntry struct {
	Name   string                 `json:"name"`
	Value  string                 `json:"value"`
	Source RBDConfigurationSource `json:"source"`
}

// Pool is the pool representation serialized from Ceph's OSD map. Stats and
// PGStatus are present only when Stats was requested; Configuration and
// ScheduleInfo are added only by GetPool. Fields with a monitor-defined or
// version-dependent shape remain raw JSON so information is not discarded.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/pybind/mgr/dashboard/controllers/pool.py (Pool)
//   - src/pybind/mgr/dashboard/services/ceph_service.py
//   - src/pybind/mgr/dashboard/services/rbd.py (RbdConfiguration)
//   - src/pybind/mgr/dashboard/controllers/_rest_controller.py
//   - src/pybind/mgr/dashboard/frontend/src/app/shared/api/pool.service.ts
//   - src/pybind/mgr/dashboard/frontend/src/app/ceph/pool/pool.ts
//   - qa/tasks/mgr/dashboard/test_pool.py
//   - src/osd/osd_types.h (pg_pool_t)
//   - src/osd/HitSet.h and src/osd/HitSet.cc
type Pool struct {
	ID                             int64                      `json:"pool"`
	Name                           string                     `json:"pool_name"`
	Flags                          int64                      `json:"flags"`
	FlagNames                      string                     `json:"flags_names"`
	Type                           PoolType                   `json:"type"`
	Size                           int64                      `json:"size"`
	MinSize                        int64                      `json:"min_size"`
	CrushRule                      string                     `json:"crush_rule"`
	PeeringCrushBucketCount        int64                      `json:"peering_crush_bucket_count"`
	PeeringCrushBucketTarget       int64                      `json:"peering_crush_bucket_target"`
	PeeringCrushBucketBarrier      int64                      `json:"peering_crush_bucket_barrier"`
	PeeringCrushMandatoryMember    int64                      `json:"peering_crush_bucket_mandatory_member"`
	StretchPool                    bool                       `json:"is_stretch_pool"`
	ObjectHash                     int64                      `json:"object_hash"`
	PGAutoscaleMode                PoolPGAutoscaleMode        `json:"pg_autoscale_mode"`
	PGNum                          int64                      `json:"pg_num"`
	PGPlacementNum                 int64                      `json:"pg_placement_num"`
	PGPlacementNumTarget           int64                      `json:"pg_placement_num_target"`
	PGNumTarget                    int64                      `json:"pg_num_target"`
	PGNumPending                   int64                      `json:"pg_num_pending"`
	LastPGMergeMeta                *PoolLastPGMergeMeta       `json:"last_pg_merge_meta,omitempty"`
	AUID                           int64                      `json:"auid"`
	SnapMode                       string                     `json:"snap_mode"`
	SnapSequence                   int64                      `json:"snap_seq"`
	SnapEpoch                      int64                      `json:"snap_epoch"`
	PoolSnapshots                  []json.RawMessage          `json:"pool_snaps"`
	QuotaMaxBytes                  int64                      `json:"quota_max_bytes"`
	QuotaMaxObjects                int64                      `json:"quota_max_objects"`
	Tiers                          []int64                    `json:"tiers"`
	TierOf                         int64                      `json:"tier_of"`
	ReadTier                       int64                      `json:"read_tier"`
	WriteTier                      int64                      `json:"write_tier"`
	CacheMode                      PoolCacheMode              `json:"cache_mode"`
	TargetMaxBytes                 int64                      `json:"target_max_bytes"`
	TargetMaxObjects               int64                      `json:"target_max_objects"`
	CacheTargetDirtyRatioMicro     int64                      `json:"cache_target_dirty_ratio_micro"`
	CacheTargetDirtyHighRatioMicro int64                      `json:"cache_target_dirty_high_ratio_micro"`
	CacheTargetFullRatioMicro      int64                      `json:"cache_target_full_ratio_micro"`
	CacheMinFlushAge               int64                      `json:"cache_min_flush_age"`
	CacheMinEvictAge               int64                      `json:"cache_min_evict_age"`
	ErasureCodeProfile             string                     `json:"erasure_code_profile"`
	HitSetParameters               PoolHitSetParameters       `json:"hit_set_params"`
	HitSetPeriod                   int64                      `json:"hit_set_period"`
	HitSetCount                    int64                      `json:"hit_set_count"`
	UseGMTHitSet                   bool                       `json:"use_gmt_hitset"`
	MinReadRecencyForPromote       int64                      `json:"min_read_recency_for_promote"`
	MinWriteRecencyForPromote      int64                      `json:"min_write_recency_for_promote"`
	HitSetGradeDecayRate           int64                      `json:"hit_set_grade_decay_rate"`
	HitSetSearchLastN              int64                      `json:"hit_set_search_last_n"`
	GradeTable                     json.RawMessage            `json:"grade_table"`
	StripeWidth                    int64                      `json:"stripe_width"`
	ExpectedNumObjects             int64                      `json:"expected_num_objects"`
	FastRead                       bool                       `json:"fast_read"`
	Options                        map[string]json.RawMessage `json:"options"`
	ApplicationMetadata            []string                   `json:"application_metadata"`
	CreateTime                     string                     `json:"create_time"`
	LastChange                     string                     `json:"last_change"`
	LastForceOpResend              string                     `json:"last_force_op_resend"`
	LastForceOpResendPreNautilus   string                     `json:"last_force_op_resend_prenautilus"`
	LastForceOpResendPreLuminous   string                     `json:"last_force_op_resend_preluminous"`
	RemovedSnapshots               json.RawMessage            `json:"removed_snaps"`
	NonPrimaryShards               json.RawMessage            `json:"nonprimary_shards"`
	ReadBalance                    map[string]json.RawMessage `json:"read_balance"`
	PGStatus                       map[string]int64           `json:"pg_status,omitempty"`
	Stats                          PoolStats                  `json:"stats,omitempty"`
	ScheduleInfo                   *PoolScheduleInfo          `json:"schedule_info,omitempty"`
	Configuration                  []RBDConfigurationEntry    `json:"configuration,omitempty"`
}

// CrushRuleStep is one operation in a CRUSH rule.
type CrushRuleStep struct {
	Operation string  `json:"op"`
	ItemName  *string `json:"item_name,omitempty"`
	Item      *int64  `json:"item,omitempty"`
	Type      *string `json:"type,omitempty"`
	Num       *int64  `json:"num,omitempty"`
}

// CrushRuleType is the numeric pool type used in CRUSH rule responses. Ceph
// uses strings for Pool.Type, so this is intentionally a separate type.
type CrushRuleType int64

const (
	CrushRuleTypeReplicated CrushRuleType = 1
	CrushRuleTypeErasure    CrushRuleType = 3
)

// CrushRule is a CRUSH rule returned by Ceph's OSD map. The same model can be
// reused by future CRUSH controller methods.
type CrushRule struct {
	ID      int64           `json:"rule_id"`
	Name    string          `json:"rule_name"`
	Ruleset *int64          `json:"ruleset,omitempty"`
	Type    CrushRuleType   `json:"type"`
	MinSize *int64          `json:"min_size,omitempty"`
	MaxSize *int64          `json:"max_size,omitempty"`
	Steps   []CrushRuleStep `json:"steps"`
}

// ErasureCodeProfileScalarMDS is the finite scalar MDS implementation value
// represented by Ceph Dashboard's erasure-code profile model.
type ErasureCodeProfileScalarMDS string

const (
	ErasureCodeProfileScalarMDSJerasure ErasureCodeProfileScalarMDS = "jerasure"
	ErasureCodeProfileScalarMDSISA      ErasureCodeProfileScalarMDS = "isa"
	ErasureCodeProfileScalarMDSSHEC     ErasureCodeProfileScalarMDS = "shec"
)

// ErasureCodeProfile is an erasure-code profile returned by Ceph. Optional
// values vary with the selected plugin.
type ErasureCodeProfile struct {
	Name                      string                      `json:"name"`
	Plugin                    string                      `json:"plugin"`
	K                         *int64                      `json:"k,omitempty"`
	M                         *int64                      `json:"m,omitempty"`
	C                         string                      `json:"c,omitempty"`
	L                         string                      `json:"l,omitempty"`
	D                         string                      `json:"d,omitempty"`
	PacketSize                string                      `json:"packetsize,omitempty"`
	Technique                 string                      `json:"technique,omitempty"`
	ScalarMDS                 ErasureCodeProfileScalarMDS `json:"scalar_mds,omitempty"`
	CrushRoot                 string                      `json:"crush-root,omitempty"`
	CrushLocality             string                      `json:"crush-locality,omitempty"`
	CrushFailureDomain        string                      `json:"crush-failure-domain,omitempty"`
	CrushNumFailureDomains    string                      `json:"crush-num-failure-domains,omitempty"`
	CrushOSDsPerFailureDomain string                      `json:"crush-osds-per-failure-domain,omitempty"`
	CrushDeviceClass          string                      `json:"crush-device-class,omitempty"`
	Directory                 string                      `json:"directory,omitempty"`
}

// CrushNode is a node in Ceph's CRUSH hierarchy.
type CrushNode struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	TypeID          int64           `json:"type_id"`
	Children        []int64         `json:"children,omitempty"`
	PoolWeights     json.RawMessage `json:"pool_weights,omitempty"`
	DeviceClass     string          `json:"device_class,omitempty"`
	CrushWeight     *float64        `json:"crush_weight,omitempty"`
	Depth           *int64          `json:"depth,omitempty"`
	Exists          *int64          `json:"exists,omitempty"`
	PrimaryAffinity *float64        `json:"primary_affinity,omitempty"`
	Reweight        *float64        `json:"reweight,omitempty"`
	Status          string          `json:"status,omitempty"`
	Content         string          `json:"content,omitempty"`
}

// PoolInfo contains the cluster-wide values used to construct or edit pools.
// Although exposed below /ui-api, it is a read-only Dashboard endpoint.
type PoolInfo struct {
	PoolNames                     []string                   `json:"pool_names"`
	CrushRulesReplicated          []CrushRule                `json:"crush_rules_replicated"`
	CrushRulesErasure             []CrushRule                `json:"crush_rules_erasure"`
	AllBlueStore                  bool                       `json:"is_all_bluestore"`
	OSDCount                      int64                      `json:"osd_count"`
	BlueStoreCompressionAlgorithm string                     `json:"bluestore_compression_algorithm"`
	CompressionAlgorithms         []string                   `json:"compression_algorithms"`
	CompressionModes              []BlueStoreCompressionMode `json:"compression_modes"`
	PGAutoscaleDefaultMode        PoolPGAutoscaleMode        `json:"pg_autoscale_default_mode"`
	PGAutoscaleModes              []PoolPGAutoscaleMode      `json:"pg_autoscale_modes"`
	ErasureCodeProfiles           []ErasureCodeProfile       `json:"erasure_code_profiles"`
	UsedRules                     map[string][]string        `json:"used_rules"`
	UsedProfiles                  map[string][]string        `json:"used_profiles"`
	Nodes                         []CrushNode                `json:"nodes"`
}

// ListPools retrieves pools through GET /api/pool.
func (client *Client) ListPools(ctx context.Context, input ListPoolsRequest) ([]Pool, error) {
	if ctx == nil {
		return nil, errors.New("mgr: context must not be nil")
	}

	endpoint := client.endpoint("api/pool")
	endpoint.RawQuery = poolQuery(input.Attributes, input.Stats).Encode()

	var pools []Pool
	if err := client.getPoolResource(ctx, endpoint, &pools); err != nil {
		return nil, err
	}
	return pools, nil
}

// GetPool retrieves one pool through GET /api/pool/{pool_name}.
func (client *Client) GetPool(ctx context.Context, input GetPoolRequest) (Pool, error) {
	if ctx == nil {
		return Pool{}, errors.New("mgr: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return Pool{}, errors.New("mgr: pool name must not be empty")
	}

	endpoint := client.poolEndpoint(input.Name)
	endpoint.RawQuery = poolQuery(input.Attributes, input.Stats).Encode()

	var pool Pool
	err := client.getPoolResource(ctx, endpoint, &pool)
	return pool, err
}

// GetPoolConfiguration retrieves a pool's effective RBD configuration through
// GET /api/pool/{pool_name}/configuration.
func (client *Client) GetPoolConfiguration(ctx context.Context, input GetPoolConfigurationRequest) ([]RBDConfigurationEntry, error) {
	if ctx == nil {
		return nil, errors.New("mgr: context must not be nil")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("mgr: pool name must not be empty")
	}

	endpoint := client.poolEndpoint(input.Name)
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/configuration"
	endpoint.RawPath = strings.TrimRight(endpoint.RawPath, "/") + "/configuration"

	var configuration []RBDConfigurationEntry
	if err := client.getPoolResource(ctx, endpoint, &configuration); err != nil {
		return nil, err
	}
	return configuration, nil
}

// GetPoolInfo retrieves cluster-wide pool form information through
// GET /ui-api/pool/info.
func (client *Client) GetPoolInfo(ctx context.Context) (PoolInfo, error) {
	if ctx == nil {
		return PoolInfo{}, errors.New("mgr: context must not be nil")
	}

	var info PoolInfo
	err := client.getPoolResource(ctx, client.endpoint("ui-api/pool/info"), &info)
	return info, err
}

func poolQuery(attributes []string, stats bool) url.Values {
	query := url.Values{}
	if len(attributes) > 0 {
		query.Set("attrs", strings.Join(attributes, ","))
	}
	if stats {
		query.Set("stats", strconv.FormatBool(stats))
	}
	return query
}

func (client *Client) poolEndpoint(name string) *url.URL {
	endpoint := client.endpoint("api/pool")
	baseEscapedPath := strings.TrimRight(endpoint.EscapedPath(), "/")
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + name
	endpoint.RawPath = baseEscapedPath + "/" + url.PathEscape(name)
	return endpoint
}

func (client *Client) getPoolResource(ctx context.Context, endpoint *url.URL, result any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	body, err := client.do(request)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("mgr: decode GET %s response: %w", request.URL.Path, err)
	}
	return nil
}
