# `rancher-monitoring` subsystem

```sh
image-tool create-list rancher-monitoring \
  '109.0.3+up80.9.1-rancher.14' \
  --chart-branch release-v2.15 -o monitoring-images.txt
```

The version is the exact **chart version**, not a Rancher or Harvester release.
`--chart-branch` is required; no branch is guessed. Both `rancher-monitoring` and
`rancher-monitoring-crd` are read at this version and branch.

## Sources

Files are read directly from:

```text
https://raw.githubusercontent.com/rancher/charts/<branch>/charts/<chart>/<version>/<file>
```

The command reads `Chart.yaml` and `values.yaml` for the main and CRD charts.
For each selected, enabled bundled subchart below, it also reads
`charts/<subchart>/Chart.yaml` and `charts/<subchart>/values.yaml` inside the main
chart directory. Metadata must match the requested main/CRD name and version;
subchart metadata must match the main chart's declared dependency version.

Parent overrides are merged over bundled subchart defaults. Parent globals are
propagated into subcharts. Images retain their chart-defined tags; repositories
are never inferred from the manual Harvester image list.

There are 12 HTTP requests for the tested version. The existing HTTP fetcher
caches requests and defaults to an 800 ms minimum interval. No archive download,
GitHub directory enumeration, clone, or Helm execution occurs at runtime.

## Harvester selection and components

This is a built-in selection of upstream chart images for Harvester's addon
configuration, not a rendering of arbitrary user values. It assumes the addon
will be enabled, enables Grafana PVC persistence, and disables
`rancherMonitoring.enabled`. Other relevant enablement switches come from the
chart. Harvester-specific images injected outside the upstream charts are not
included.

Paths below are relative to the owning chart's `values.yaml` unless stated
otherwise. Each image mapping supplies `repository` and `tag`.

| Component | Source fields and conditions |
| --- | --- |
| Prometheus | `prometheus.prometheusSpec.image` and `prometheus.prometheusSpec.proxy.image`, when `prometheus.enabled` |
| Alertmanager | `alertmanager.alertmanagerSpec.image`, when `alertmanager.enabled` |
| Prometheus Operator | `prometheusOperator.image` and `prometheusOperator.prometheusConfigReloader.image`, when `prometheusOperator.enabled` |
| Grafana | Bundled `grafana`: `image`, parent-provided `proxy.image`, `sidecar.image` when dashboards or datasources are enabled, and `initChownData.image` when the ownership init container is enabled; gated by parent `grafana.enabled` |
| kube-state-metrics | Bundled `kube-state-metrics`: `image`, when parent `kubeStateMetrics.enabled` |
| Node exporter | Bundled `prometheus-node-exporter`: `image`, when parent `nodeExporter.enabled` |
| Prometheus adapter | Bundled `prometheus-adapter`: `image`, when parent `prometheus-adapter.enabled` |
| Admission webhook | `prometheusOperator.admissionWebhooks.deployment.image`, always included even when its deployment is disabled |
| Admission patch | `prometheusOperator.admissionWebhooks.patch.image`, when operator, admission webhooks, and patch are enabled and cert-manager is disabled |
| Main upgrade | `upgrade.image`, when `upgrade.enabled` |
| Main CRD upgrade | `crds.upgradeJob.image.busybox` always; `.kubectl` when both `crds.enabled` and `crds.upgradeJob.enabled` |
| Monitoring CRD | Separate `rancher-monitoring-crd`: `image`, for installation/upgrade jobs |

The nginx proxies are part of this fixed Harvester selection. Their image
mappings are required. Disabled Thanos, Windows exporters, PushProx subcharts,
Grafana renderer and test workloads
are outside this selection. The collector does not inventory all optional chart
features or images declared only in CRD schemas.

## Component examples

These examples use chart version `109.0.3+up80.9.1-rancher.14` on
`release-v2.15`. YAML excerpts retain the source field hierarchy. Explicit
`docker.io/` prefixes are normalized away in the output.

### prometheus

Main `values.yaml: .prometheus.prometheusSpec.image` and
`.prometheus.prometheusSpec.proxy.image` → repository/tag references for
Prometheus and its nginx proxy, when `.prometheus.enabled` is true.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
prometheus:
  enabled: true
  prometheusSpec:
    image:
      repository: rancher/prom-prometheus
      tag: v3.8.1
    proxy:
      image:
        repository: rancher/mirrored-library-nginx
        tag: 1.29.1-alpine
```

→ `rancher/prom-prometheus:v3.8.1`, `rancher/mirrored-library-nginx:1.29.1-alpine`

### alertmanager

Main `values.yaml: .alertmanager.alertmanagerSpec.image.repository`:
`.alertmanager.alertmanagerSpec.image.tag`, when `.alertmanager.enabled` is true.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
alertmanager:
  enabled: true
  alertmanagerSpec:
    image:
      repository: rancher/appco-alertmanager
      tag: 0.30.0-12.16
```

→ `rancher/appco-alertmanager:0.30.0-12.16`

### prometheus-operator

Main `values.yaml: .prometheusOperator.image` and
`.prometheusOperator.prometheusConfigReloader.image` → operator and config
reloader images. Both tags fall back to main `Chart.yaml: .appVersion` if empty.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
prometheusOperator:
  enabled: true
  image:
    repository: rancher/mirrored-prometheus-operator-prometheus-operator
    tag: v0.87.1
  prometheusConfigReloader:
    image:
      repository: rancher/mirrored-prometheus-operator-prometheus-config-reloader
      tag: v0.87.1
```

→ `rancher/mirrored-prometheus-operator-prometheus-operator:v0.87.1`, `rancher/mirrored-prometheus-operator-prometheus-config-reloader:v0.87.1`

### grafana

Bundled `charts/grafana/values.yaml` supplies `.image`, `.sidecar.image`,
and `.initChownData.image`. Parent `.grafana` values override matching child
fields and supply `.proxy.image`. The Harvester selection enables PVC persistence,
so the enabled ownership init container contributes its image.

Example: [rancher-monitoring/charts/grafana/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/charts/grafana/values.yaml) (excerpt):

```yaml
image:
  repository: rancher/appco-grafana
  tag: 12.3.1-1.12
sidecar:
  image:
    repository: rancher/appco-k8s-sidecar
    tag: 2.1.2-1.10
initChownData:
  enabled: true
  image:
    repository: rancher/mirrored-library-busybox
    tag: 1.31.1
```

The parent enables Grafana and its dashboard/datasource sidecars and supplies
the nginx proxy:

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
grafana:
  enabled: true
  sidecar:
    dashboards:
      enabled: true
    datasources:
      enabled: true
  proxy:
    image:
      repository: rancher/mirrored-library-nginx
      tag: 1.29.1-alpine
```

→ `rancher/appco-grafana:12.3.1-1.12`, `rancher/appco-k8s-sidecar:2.1.2-1.10`, `rancher/mirrored-library-busybox:1.31.1`, `rancher/mirrored-library-nginx:1.29.1-alpine`

The nginx reference also comes from Prometheus’s proxy; the final output
contains it only once. The two BusyBox tags in this document remain distinct.

### kube-state-metrics

Parent `.kubeStateMetrics.enabled` enables the bundled `kube-state-metrics` chart.
Read its `values.yaml: .image.repository`:`.image.tag`, after applying any
parent overrides. An empty tag falls back to `v<subchart appVersion>`.

Example: [rancher-monitoring/charts/kube-state-metrics/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/charts/kube-state-metrics/values.yaml) (excerpt):

```yaml
image:
  repository: rancher/appco-kube-state-metrics
  tag: 2.17.0-10.14
```

→ `rancher/appco-kube-state-metrics:2.17.0-10.14`

### prometheus-node-exporter

Parent `.nodeExporter.enabled` enables the bundled `prometheus-node-exporter` chart.
Read its `values.yaml: .image.repository`:`.image.tag`, after applying any
parent overrides. An empty tag falls back to `v<subchart appVersion>`.

Example: [rancher-monitoring/charts/prometheus-node-exporter/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/charts/prometheus-node-exporter/values.yaml) (excerpt):

```yaml
image:
  repository: rancher/appco-node-exporter
  tag: 1.10.2-14.3
```

→ `rancher/appco-node-exporter:1.10.2-14.3`

### prometheus-adapter

Parent `.prometheus-adapter.enabled` enables the bundled `prometheus-adapter` chart.
Read its `values.yaml: .image.repository`:`.image.tag`, after applying any
parent overrides. An empty tag falls back to the subchart’s `appVersion`.

Example: [rancher-monitoring/charts/prometheus-adapter/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/charts/prometheus-adapter/values.yaml) (excerpt):

```yaml
image:
  repository: rancher/mirrored-prometheus-adapter-prometheus-adapter
  tag: v0.12.0
```

→ `rancher/mirrored-prometheus-adapter-prometheus-adapter:v0.12.0`

### admission-webhook

Main `values.yaml: .prometheusOperator.admissionWebhooks.deployment.image`
→ repository/tag reference, falling back to main `appVersion` for an empty tag.
This image is explicitly included even when the deployment is disabled.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
prometheusOperator:
  admissionWebhooks:
    deployment:
      enabled: false
      image:
        repository: rancher/mirrored-prometheus-operator-admission-webhook
        tag: v0.87.1
```

→ `rancher/mirrored-prometheus-operator-admission-webhook:v0.87.1`

### admission-patch

Main `values.yaml: .prometheusOperator.admissionWebhooks.patch.image`
→ repository/tag reference. Include it when the operator, admission webhooks,
and patch jobs are enabled and cert-manager is disabled.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
prometheusOperator:
  enabled: true
  admissionWebhooks:
    enabled: true
    certManager:
      enabled: false
    patch:
      enabled: true
      image:
        repository: rancher/mirrored-jkroepke-kube-webhook-certgen
        tag: 1.7.4
```

→ `rancher/mirrored-jkroepke-kube-webhook-certgen:1.7.4`

### monitoring-upgrade

Main `values.yaml: .upgrade.image.repository`:`.upgrade.image.tag`,
when `.upgrade.enabled` is true.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
upgrade:
  enabled: true
  image:
    repository: rancher/kuberlr-kubectl
    tag: v7.1.0
```

→ `rancher/kuberlr-kubectl:v7.1.0`

### monitoring-crd-upgrade

Main `values.yaml: .crds.upgradeJob.image.busybox` is always included.
The sibling `.kubectl` image is included in this component only when both
`.crds.enabled` and `.crds.upgradeJob.enabled` are true.

Example: [rancher-monitoring/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
crds:
  enabled: true
  upgradeJob:
    enabled: false
    image:
      busybox:
        repository: rancher/mirrored-library-busybox
        tag: 1.37.0
      kubectl:
        repository: rancher/kuberlr-kubectl
        tag: v7.1.0
```

→ `rancher/mirrored-library-busybox:1.37.0`

The job is disabled in this example, so its kubectl image is not added by
this component. `rancher/kuberlr-kubectl:v7.1.0` is already collected from
`monitoring-upgrade`.

### rancher-monitoring-crd

Separate `rancher-monitoring-crd` chart at the requested version →
`values.yaml: .image.repository`:`.image.tag` for installation/upgrade jobs.
Its tag is independent of the main chart’s upgrade-job tag.

Example: [rancher-monitoring-crd/values.yaml](https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-monitoring-crd/109.0.3+up80.9.1-rancher.14/values.yaml) (excerpt):

```yaml
image:
  repository: rancher/kuberlr-kubectl
  tag: v7.1.1
```

→ `rancher/kuberlr-kubectl:v7.1.1`

## Image template rules

- Standalone admission-webhook images use `global.imageRegistry`, falling back
  to their image registry, and ignore the Rancher registry. Their tag falls back
  to main `appVersion`; `sha` is prefixed with `sha256:`.
- Operator and config-reloader tags fall back to main `Chart.yaml.appVersion`.
  Grafana and adapter use their own subchart `appVersion`; exporters use
  `v<subchart appVersion>`. Other collected mappings require explicit tags,
  except Prometheus and Alertmanager can use a digest alone.
- Registry precedence follows the relevant templates: monitoring images use
  Rancher's `global.cattle.systemDefaultRegistry`, then `global.imageRegistry`,
  then the image registry. Operator/reloader image registries override those
  globals. Grafana uses the Rancher registry or its image registry. Proxy,
  logging-style helpers, adapter, and upgrade jobs use only the Rancher registry.
- Prometheus/Alertmanager/operator/Grafana `sha` fields are prefixed with
  `sha256:`. kube-state-metrics `sha` and node-exporter `digest` are already
  qualified. Node exporter rejects `sha`, matching its template.
- Explicit Docker Hub prefixes are normalized away. Identical image references
  are deduplicated; different tags for the same repository remain separate.

The supplied version produces **16 unique images**. In particular:

- The main upgrade job uses `rancher/kuberlr-kubectl:v7.1.0`.
- The CRD chart uses `rancher/kuberlr-kubectl:v7.1.1`.
- Grafana's persistence init container uses
  `rancher/mirrored-library-busybox:1.31.1`.
- BusyBox `1.37.0` from the main CRD upgrade job is always included, even
  when the job is disabled. Grafana's BusyBox `1.31.1` remains included too.
- The standalone admission-webhook image is also included as an explicit
  exception to disabled-feature exclusion:
  `rancher/mirrored-prometheus-operator-admission-webhook:v0.87.1`.

The manual Harvester `monitoring-images.txt` combines logging and monitoring and
is not an authoritative expected set. Its `rancher/kubectl:v1.35.1` and ingress
nginx certgen entry are not selected by these chart sources. The selected patch
job uses `rancher/mirrored-jkroepke-kube-webhook-certgen:1.7.4`.

No `Verifier` is implemented, so `--strict` and `--no-verify` have no effect on
this subsystem. Missing required files, fields, switches, and metadata mismatches
fail before output is written. The supplied chart version is the tested
baseline; support for a changed upstream schema requires updating collectors
and fixtures. These checks do not detect every possible upstream template change.

See [fixture provenance and template evidence](../pkg/subsystem/ranchercharts/testdata/README.md).
