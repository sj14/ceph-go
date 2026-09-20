package rgw

// StorageStats contains RGW object and byte usage counters. Ceph uses this
// model both for aggregate user statistics and for each bucket usage category.
//
// Verified against Ceph v20.2.4 (tag commit 7f793731f1b3):
//   - src/rgw/rgw_common.cc (RGWStorageStats::dump)
//   - src/rgw/driver/rados/rgw_user.cc (dump_user_info)
//   - src/rgw/driver/rados/rgw_bucket.cc (dump_bucket_usage)
type StorageStats struct {
	Size           int64 `json:"size"`
	SizeActual     int64 `json:"size_actual"`
	SizeUtilized   int64 `json:"size_utilized"`
	SizeKB         int64 `json:"size_kb"`
	SizeKBActual   int64 `json:"size_kb_actual"`
	SizeKBUtilized int64 `json:"size_kb_utilized"`
	NumObjects     int64 `json:"num_objects"`
}
