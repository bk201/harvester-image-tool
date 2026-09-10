# `rancher-logging` subsystem

```sh
image-tool create-list rancher-logging \
  '109.0.0+up4.10.0-rancher.23' \
  --chart-branch release-v2.15 -o logging-images.txt
```

The version is an exact chart version. `--chart-branch` is required and applies
to both `rancher-logging` and `rancher-logging-crd` at that version.

The CLI reads individual files under
`https://raw.githubusercontent.com/rancher/charts/<branch>/charts/<chart>/<version>/`:

- `rancher-logging`: `Chart.yaml` and `values.yaml`.
- `rancher-logging-crd`: `Chart.yaml` only.

Both metadata files must match the requested chart name and version. The CRD
chart contains only definitions in the tested version; it contributes an empty
component result. The collector does not scan CRD schema properties for images.
There are three HTTP requests, through the existing cached, rate-limited
fetcher. No chart archives are downloaded at runtime.

## Components

| Component | Fields in main `values.yaml` | Example image |
| --- | --- | --- |
| Logging operator | `.image.repository`, `.image.tag` (falls back to main `appVersion`) | `rancher/mirrored-kube-logging-logging-operator:4.10.0` |
| Fluentd | `.images.fluentd.repository`, `.images.fluentd.tag` | `rancher/mirrored-kube-logging-fluentd:v1.16-4.10-full` |
| Fluent Bit | `.images.fluentbit.repository`, `.images.fluentbit.tag` | `rancher/mirrored-fluent-fluent-bit:3.1.8` |
| Config reloader | `.images.config_reloader.repository`, `.images.config_reloader.tag` | `rancher/mirrored-kube-logging-config-reloader:v0.0.6` |

## Component examples

These examples use chart version `109.0.0+up4.10.0-rancher.23` on
`release-v2.15`, with the default empty Rancher registry prefix.

### logging-operator

Main `values.yaml: .image.repository`:`.image.tag`. If the tag is empty,
use the main chart’s `Chart.yaml: .appVersion`.

Example: [rancher-logging/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-logging/109.0.0+up4.10.0-rancher.23/values.yaml) (excerpt):

```yaml
image:
  repository: rancher/mirrored-kube-logging-logging-operator
  tag: 4.10.0
```

→ `rancher/mirrored-kube-logging-logging-operator:4.10.0`

### fluentd

Main `values.yaml: .images.fluentd.repository`:`.images.fluentd.tag`.

Example: [rancher-logging/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-logging/109.0.0+up4.10.0-rancher.23/values.yaml) (excerpt):

```yaml
images:
  fluentd:
    repository: rancher/mirrored-kube-logging-fluentd
    tag: v1.16-4.10-full
```

→ `rancher/mirrored-kube-logging-fluentd:v1.16-4.10-full`

### fluentbit

Main `values.yaml: .images.fluentbit.repository`:`.images.fluentbit.tag`.
This is the normal Linux image selected for Harvester logging.

Example: [rancher-logging/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-logging/109.0.0+up4.10.0-rancher.23/values.yaml) (excerpt):

```yaml
images:
  fluentbit:
    repository: rancher/mirrored-fluent-fluent-bit
    tag: 3.1.8
```

→ `rancher/mirrored-fluent-fluent-bit:3.1.8`

### config-reloader

Main `values.yaml: .images.config_reloader.repository`:`.images.config_reloader.tag`.

Example: [rancher-logging/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-logging/109.0.0+up4.10.0-rancher.23/values.yaml) (excerpt):

```yaml
images:
  config_reloader:
    repository: rancher/mirrored-kube-logging-config-reloader
    tag: v0.0.6
```

→ `rancher/mirrored-kube-logging-config-reloader:v0.0.6`

### rancher-logging-crd

Validate the separate CRD chart at the same requested version. This chart
contains definitions only in the tested version and contributes no images.

Example: [rancher-logging-crd/Chart.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-logging-crd/109.0.0+up4.10.0-rancher.23/Chart.yaml) (excerpt):

```yaml
name: rancher-logging-crd
version: 109.0.0+up4.10.0-rancher.23
```

→ No image references; the CRD component result is empty.

## Output and exclusions

These are the **four unique images** selected for Harvester's Linux RKE2 and
audit logging component set, assuming the addon will be enabled. The Rancher
`global.cattle.systemDefaultRegistry` prefix is applied when present.

The selection excludes Windows and debug Fluent Bit variants, test receivers,
and optional tailer images. Harvester's injected event-router version cannot be
derived from these charts and is excluded. There is no Harvester release input
or arbitrary values-file support.

Output uses the existing sorted, deduplicated format and optional headers. No
`Verifier` is implemented: the manual combined monitoring/logging list is not an
authoritative upstream release list. `--strict` and `--no-verify` therefore have
no effect for this subsystem.

Missing required source files or image fields and metadata mismatches fail
before writing output. The supplied version is the tested baseline; new chart
layouts or CRD workloads need an explicit collector update.

See [fixture provenance and template evidence](../pkg/subsystem/ranchercharts/testdata/README.md).
