#!/usr/bin/env bash
set -Eeuo pipefail

readonly cluster_name=ceph
readonly mon_id=test
readonly mgr_id=test
readonly osd_id=0
readonly rgw_id=test
readonly fsid=00000000-0000-0000-0000-000000000001
readonly ceph_dir=/etc/ceph
readonly data_dir=/var/lib/ceph
readonly run_dir=/run/ceph
readonly dashboard_user=${CEPH_DASHBOARD_USER:-admin}
readonly dashboard_password=${CEPH_DASHBOARD_PASSWORD:-admin}
readonly dashboard_port=${CEPH_DASHBOARD_PORT:-8443}
readonly rgw_port=${CEPH_RGW_PORT:-8000}
readonly test_rgw_user=${CEPH_TEST_RGW_USER:-ceph-go-test}
readonly admin_rgw_user=${CEPH_ADMIN_RGW_USER:-ceph-go-admin}
readonly admin_rgw_access_key=${CEPH_ADMIN_RGW_ACCESS_KEY:-RGWGOADMINACCESSKEY}
readonly admin_rgw_secret_key=${CEPH_ADMIN_RGW_SECRET_KEY:-ceph-go-admin-secret-key-for-integration-tests}
# Keep this in sync with RGWUserCaps::is_valid_cap_type() in Ceph v20.2.4's
# src/rgw/rgw_common.cc. Ceph supports wildcard permissions, but not a wildcard
# capability type.
readonly admin_rgw_caps=${CEPH_ADMIN_RGW_CAPS:-'user=*;users=*;buckets=*;metadata=*;info=*;usage=*;zone=*;bilog=*;mdlog=*;datalog=*;roles=*;user-policy=*;amz-cache=*;oidc-provider=*;user-info-without-keys=*;ratelimit=*;accounts=*'}

declare -a daemon_pids=()

log() {
  printf '[ceph-test] %s\n' "$*"
}

cleanup() {
  local pid
  for pid in "${daemon_pids[@]:-}"; do
    kill -TERM "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

wait_for_ceph() {
  local attempt
  for attempt in {1..60}; do
    if ceph status >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  log 'Ceph monitor did not become ready'
  return 1
}

wait_for_mgr() {
  local attempt
  for attempt in {1..60}; do
    if ceph mgr stat 2>/dev/null | grep -q 'active_name'; then
      return 0
    fi
    sleep 1
  done
  log 'Ceph manager did not become ready'
  return 1
}

wait_for_rgw() {
  local attempt
  for attempt in {1..60}; do
    if curl -fsS "http://127.0.0.1:${rgw_port}/" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  log 'RGW did not become ready'
  return 1
}

wait_for_dashboard() {
  local attempt
  for attempt in {1..60}; do
    if curl -fsS "http://127.0.0.1:${dashboard_port}/" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  log 'Ceph Dashboard did not become ready'
  return 1
}

mkdir -p \
  "$ceph_dir" \
  "$run_dir" \
  "$data_dir/mon/${cluster_name}-${mon_id}" \
  "$data_dir/mgr/${cluster_name}-${mgr_id}" \
  "$data_dir/osd/${cluster_name}-${osd_id}" \
  "$data_dir/radosgw/${cluster_name}-rgw.${rgw_id}"

cat >"$ceph_dir/ceph.conf" <<EOF
[global]
fsid = ${fsid}
mon host = 127.0.0.1
mon initial members = ${mon_id}
auth cluster required = cephx
auth service required = cephx
auth client required = cephx
osd pool default size = 1
osd pool default min size = 1
osd crush chooseleaf type = 0
osd objectstore = memstore
mon allow pool delete = true

[client.rgw.${rgw_id}]
rgw frontends = beast endpoint=0.0.0.0:${rgw_port}
rgw enable usage log = true
EOF

if [[ ! -e "$data_dir/mon/${cluster_name}-${mon_id}/store.db" ]]; then
  log 'Bootstrapping monitor'
  ceph-authtool --create-keyring /tmp/mon.keyring --gen-key -n mon. --cap mon 'allow *'
  ceph-authtool /tmp/mon.keyring --gen-key -n client.admin \
    --cap mon 'allow *' --cap mgr 'allow *' --cap osd 'allow *' --cap mds 'allow *'
  cp /tmp/mon.keyring "$ceph_dir/ceph.client.admin.keyring"
  monmaptool --create --add "$mon_id" 127.0.0.1 --fsid "$fsid" /tmp/monmap
  ceph-mon --mkfs -i "$mon_id" --monmap /tmp/monmap --keyring /tmp/mon.keyring
fi
chown -R ceph:ceph "$data_dir/mon/${cluster_name}-${mon_id}"

log 'Starting monitor'
ceph-mon -f -i "$mon_id" --setuser ceph --setgroup ceph &
daemon_pids+=("$!")
wait_for_ceph

log 'Bootstrapping manager'
ceph auth get-or-create "mgr.${mgr_id}" \
  mon 'allow profile mgr' osd 'allow *' mds 'allow *' \
  -o "$data_dir/mgr/${cluster_name}-${mgr_id}/keyring"
chown -R ceph:ceph "$data_dir/mgr/${cluster_name}-${mgr_id}"
ceph-mgr -f -i "$mgr_id" --setuser ceph --setgroup ceph &
daemon_pids+=("$!")
wait_for_mgr

if [[ ! -e "$data_dir/osd/${cluster_name}-${osd_id}/ready" ]]; then
  log 'Bootstrapping in-memory OSD'
  osd_uuid=$(cat /proc/sys/kernel/random/uuid)
  ceph osd create "$osd_uuid" "$osd_id"
  ceph auth get-or-create "osd.${osd_id}" \
    mon 'allow profile osd' mgr 'allow profile osd' osd 'allow *' \
    -o "$data_dir/osd/${cluster_name}-${osd_id}/keyring"
  chown -R ceph:ceph "$data_dir/osd/${cluster_name}-${osd_id}"
  ceph-osd --mkfs -i "$osd_id" --osd-uuid "$osd_uuid"
fi
chown -R ceph:ceph "$data_dir/osd/${cluster_name}-${osd_id}"
ceph-osd -f -i "$osd_id" --setuser ceph --setgroup ceph &
daemon_pids+=("$!")

log 'Bootstrapping RGW'
ceph auth get-or-create "client.rgw.${rgw_id}" \
  mon 'allow rw' osd 'allow rwx' \
  -o "$data_dir/radosgw/${cluster_name}-rgw.${rgw_id}/keyring"
chown -R ceph:ceph "$data_dir/radosgw/${cluster_name}-rgw.${rgw_id}"
radosgw -f -n "client.rgw.${rgw_id}" --setuser ceph --setgroup ceph &
daemon_pids+=("$!")
wait_for_rgw

if ! radosgw-admin user info --uid "$test_rgw_user" >/dev/null 2>&1; then
  radosgw-admin user create --uid "$test_rgw_user" --display-name 'ceph-go test user' >/dev/null
fi
if ! radosgw-admin user info --uid "$admin_rgw_user" >/dev/null 2>&1; then
  radosgw-admin user create \
    --uid "$admin_rgw_user" \
    --display-name 'ceph-go Admin Ops test user' \
    --system \
    --access-key "$admin_rgw_access_key" \
    --secret-key "$admin_rgw_secret_key" >/dev/null
fi
radosgw-admin caps add --uid "$admin_rgw_user" --caps "$admin_rgw_caps" >/dev/null

log 'Configuring dashboard'
ceph config set mgr mgr/dashboard/ssl false
ceph config set mgr mgr/dashboard/server_addr 0.0.0.0
ceph config set mgr mgr/dashboard/server_port "$dashboard_port"
ceph mgr module enable dashboard --force
printf '%s' "$dashboard_password" >/tmp/dashboard-password
if ! ceph dashboard ac-user-show "$dashboard_user" >/dev/null 2>&1; then
  ceph dashboard ac-user-create "$dashboard_user" \
    -i /tmp/dashboard-password administrator --force-password
fi
rm -f /tmp/dashboard-password
ceph dashboard set-rgw-credentials
wait_for_dashboard

log "Ready: dashboard=http://127.0.0.1:${dashboard_port} rgw=http://127.0.0.1:${rgw_port}"
wait -n "${daemon_pids[@]}"
