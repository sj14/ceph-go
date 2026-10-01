# API usability review

Review potential misuse against the latest stable Ceph source using AGENTS.md.
These are checks to perform, not confirmed defects. Address one issue at a time;
present the options when multiple fixes are reasonable, and wait for the user's
OK before continuing to the next issue.

## RGW follow-up checks

- [ ] Review `GetUserRequest.Sync` and `GetUserByAccessKeyRequest.Sync`: these
  getters can update stored statistics, including account-wide statistics for
  account members. Decide whether clearer naming/documentation or a separate
  synchronization action would make the side effect sufficiently explicit.

## Dashboard (`mgr`) checks

- [ ] Review user list/detail response completeness, optional statistics, and
  create/update defaults for surprising omissions or mutations.
- [ ] Review S3/Swift credential, subuser, and capability requests for competing
  selectors, ignored fields, misleading collection results, and required values.
- [ ] Review quota and rate-limit getters/setters for scope-dependent models,
  units, zero values, and omission semantics.
- [ ] Review remaining bucket creation/update/deletion options for defaults,
  incompatible settings, ignored fields, and unexpected side effects.
- [ ] Review full/minimal health responses and `GetHealthSnapshot` for partially
  populated models and ambiguous absent versus zero values.
- [ ] Review remaining pool information/configuration models for conditional
  fields and absence semantics.
- [ ] Check response models for shared Ceph domain types and finite wire values
  that should use named types and constants.
