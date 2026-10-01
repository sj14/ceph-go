package mgr

// Quota is the RGWQuotaInfo representation shared by Dashboard user and bucket
// responses. Verified against Ceph v20.2.4: src/rgw/rgw_quota.cc
// (RGWQuotaInfo::dump), src/pybind/mgr/dashboard/controllers/rgw.py, and
// src/pybind/mgr/dashboard/services/rgw_client.py (proxied Admin Ops responses).
type Quota struct {
	Enabled    bool  `json:"enabled"`
	CheckOnRaw bool  `json:"check_on_raw"`
	MaxSize    int64 `json:"max_size"`
	MaxSizeKB  int64 `json:"max_size_kb"`
	MaxObjects int64 `json:"max_objects"`
}
