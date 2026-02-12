# Autogrow Volume Tests

These tests validate the automatic volume expansion (autogrow) feature of PGO.
They create PostgresCluster instances with 1Gi volumes, write data to trigger the
autogrow threshold, and verify that PVC requests and capacities are updated correctly.

## Kubernetes Environment Requirements

These tests require a storage provider that supports **real volume expansion**, meaning
the CSI driver must implement `ControllerExpandVolume` and/or `NodeExpandVolume` so
that PVC `status.capacity` is updated after a resize request.

The default StorageClass must have `allowVolumeExpansion: true`.

### Supported environments

- **GKE** with `pd-standard` or `pd-ssd` StorageClass (tests were designed for this)
- **AWS EKS** with `gp2` or `gp3` StorageClass
- **Azure AKS** with `managed-csi` StorageClass
- Any cluster with a CSI driver that supports volume expansion

### Unsupported environments

- **k3d / k3s** with the default `local-path` provisioner — `local-path` does not
  implement volume expansion. Setting `allowVolumeExpansion: true` on the StorageClass
  allows PVC `spec.resources.requests` to be updated, but `status.capacity` will not
  change, causing test assertions to fail.
- **kind** with the default local provisioner (same limitation)

### Running in CI

These tests are excluded from the GitHub Actions e2e workflow (`--exclude-test-regex autogrow`)
because it uses k3d with `local-path`. They should be run in a CI environment that
provides a storage backend with volume expansion support (e.g., GKE).
