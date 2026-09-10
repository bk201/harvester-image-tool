# Fixture provenance

The YAML files are verbatim extracted source files fetched from `rancher/charts`
branch `release-v2.15` on 2026-09-07. [SOURCES.txt](SOURCES.txt) records every exact
URL. All 15 runtime source files were re-fetched from `release-v2.15` and
compared byte-for-byte with the original `release-v2.14` fixtures on 2026-09-07;
all were identical. Only source URLs and the tested branch changed.

A separate archive-content comparison also found all 694 regular files
identical across the four charts (monitoring, monitoring-crd, logging, and
logging-crd), including templates and bundled dependencies. Archives were used
only for this verification; runtime fetching remains individual source files.

Tests use these committed bytes without network access. When refreshing,
review template behavior as well as values; do not regenerate expected images
from the implementation itself.

Tested versions:

- Monitoring and monitoring-crd: `109.0.3+up80.9.1-rancher.14`.
- Logging and logging-crd: `109.0.0+up4.10.0-rancher.23`.

Archives were inspected during implementation only to audit templates and
bundled dependencies. All fixture source URLs were separately fetched from the
extracted chart directories. Runtime collection uses only those extracted-file
URLs and fetches no templates for this baseline, because the required images
are fully represented in values and metadata.

## Template evidence

Paths below are relative to the corresponding chart directory at the branch
and version above. They can be appended to the same raw GitHub URL base in
SOURCES.txt for review.

Monitoring:

- `templates/prometheus/prometheus.yaml` and
  `templates/alertmanager/alertmanager.yaml`: image repository, tag and optional
  SHA in operator CRs, including SHA-only references.
- `templates/prometheus-operator/deployment.yaml`: operator image and
  `--prometheus-config-reloader` argument; image registry overrides the
  monitoring registry; empty tags fall back to main `appVersion`.
- `templates/_helpers.tpl`: `monitoring_registry` prefers Rancher's registry
  over `global.imageRegistry`; `system_default_registry` uses Rancher's registry.
- `templates/prometheus-operator/admission-webhooks/job-patch/job-createSecret.yaml`
  and `job-patchWebhook.yaml`: patch image used only with operator, webhooks,
  patch enabled and cert-manager disabled.
- `templates/prometheus-operator/admission-webhooks/deployment/deployment.yaml`:
  separate deployment is disabled by the supplied values, but its image is
  explicitly included in the inventory. Registry selection uses
  `global.imageRegistry`, falling back to the image registry; the tag falls back
  to main `appVersion`, and SHA syntax is `@sha256:<sha>`.
- `templates/rancher-monitoring/upgrade/job.yaml`: `upgrade.enabled` controls the
  pre-upgrade/pre-rollback job and `.upgrade.image`.
- `charts/grafana/templates/_pod.tpl`: main image falls back to child appVersion;
  init-chown image is required when persistence and initChownData are enabled;
  dashboards/datasources use sidecar images. SHA syntax is `@sha256:<sha>`.
- Main `values.yaml` supplies nginx proxy containers and their image mappings
  for Grafana and Prometheus.
- `charts/kube-state-metrics/templates/_helpers.tpl`: Rancher registry, then
  global registry, then image registry; tag fallback is `v<appVersion>` and
  `sha` is already a qualified digest.
- `charts/prometheus-node-exporter/templates/_helpers.tpl`: same registry/tag
  fallback, with an already qualified `digest`; `sha` is forbidden.
- `charts/prometheus-adapter/templates/deployment.yaml`: Rancher registry,
  `.image.repository`, and tag fallback to child appVersion.

Monitoring CRD:

- `templates/jobs.yaml`: installation/upgrade jobs use `.image`; this is
  independent of the main chart's upgrade image/version.

Logging:

- `templates/deployment.yaml`: `.image`, Rancher registry, appVersion fallback.
- `templates/_generic_logging.yaml`: fluentd and config-reloader mappings.
- `templates/_generic_fluentbitagent.yaml` and `templates/_helpers.tpl`: normal
  Linux Fluent Bit mapping selected with debug off.
- `templates/loggings/rke2/daemonset.yaml`: reuses the same Fluent Bit image.
- The logging CRD archive contains CRD definitions, no runtime image workloads.

Harvester profile was checked against `harvester/addons`
`pkg/templates/rancherd-22-addons.yaml`: Grafana PVC persistence enabled,
`rancherMonitoring.enabled: false`, RKE2/audit logging sources enabled. Its
Harvester-injected event-router is deliberately outside these collectors.
