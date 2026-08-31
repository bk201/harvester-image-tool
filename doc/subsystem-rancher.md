# `rancher` subsystem

Sources, per Rancher `<version>` (e.g. `v2.15.1`):

- `build.yaml` and `package/Dockerfile` at `raw.githubusercontent.com/rancher/rancher/<version>/...`. A 404 here means the version doesn't exist.
- `rancher/charts` chart files at `raw.githubusercontent.com/rancher/charts/<charts-branch>/charts/<chart>/<chart-version>/...`, where `<charts-branch>` is `release-v<major>.<minor>` derived from `<version>` (e.g. `v2.15.1` → `release-v2.15`), unless overridden with `--rancher-charts-branch`.
- The official release image list at `github.com/rancher/rancher/releases/download/<version>/rancher-images.txt`, used only to verify (warn on any generated image missing from it).

Each component below is described first in general terms, then with a concrete example using `v2.15.1`.

## Components

### rancher-manager

No file lookup — both images are `<version>` itself:

- `rancher/rancher:<version>`
- `rancher/rancher-agent:<version>`

Example (`v2.15.1`): `rancher/rancher:v2.15.1`, `rancher/rancher-agent:v2.15.1`

### rancher-webhook

`build.yaml: .webhookVersion` → chart `rancher-webhook` at that version → `values.yaml: .image.repository`:`.image.tag`.

Example: `build.yaml` at `v2.15.1` has

```yaml
webhookVersion: 110.0.2+up0.11.1
```

→ `https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-webhook/110.0.2+up0.11.1/values.yaml`:

```yaml
image:
  repository: rancher/rancher-webhook
  tag: v0.11.1
```

→ `rancher/rancher-webhook:v0.11.1`

### fleet

`build.yaml: .fleetVersion` → chart `fleet` at that version → `values.yaml`:

- `.image.repository`:`.image.tag` (controller)
- `.agentImage.repository`:`.agentImage.tag` (agent)

Example: `build.yaml` at `v2.15.1` has

```yaml
fleetVersion: 110.0.1+up0.16.1
```

→ `https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/fleet/110.0.1+up0.16.1/values.yaml`:

```yaml
image:
  repository: rancher/fleet
  tag: v0.16.1

agentImage:
  repository: rancher/fleet-agent
  tag: v0.16.1
```

→ `rancher/fleet:v0.16.1`, `rancher/fleet-agent:v0.16.1`

### shell

`build.yaml: .defaultShellVersion`. This field is already a full image reference (e.g. `rancher/shell:v0.8.1`), used as-is.

Example: `build.yaml` at `v2.15.1` has

```yaml
defaultShellVersion: rancher/shell:v0.8.1
```

→ `rancher/shell:v0.8.1` (used verbatim, no `<version>` appended)

### turtles

`build.yaml: .turtlesVersion` → chart `rancher-turtles` at that version → `values.yaml`:

- `.image.repository`:`.image.tag` (turtles)
- `.shellImage.image.repository`:`.shellImage.image.tag` (kubectl shell image)

Plus, from the same chart version's `templates/core-provider-configmap.yaml` → `.metadata.labels["provider.cluster.x-k8s.io/version"]` → `rancher/cluster-api-controller:<that label's value>`.

Example: `build.yaml` at `v2.15.1` has

```yaml
turtlesVersion: 110.0.1+up0.27.1
```

→ `https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-turtles/110.0.1+up0.27.1/values.yaml`:

```yaml
image:
  repository: rancher/turtles
  tag: v0.27.1

shellImage:
  image:
    repository: rancher/kuberlr-kubectl
    tag: v8.1.1
```

→ `rancher/turtles:v0.27.1`, `rancher/kuberlr-kubectl:v8.1.1`

And `https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/rancher-turtles/110.0.1+up0.27.1/templates/core-provider-configmap.yaml`:

```yaml
metadata:
  labels:
    provider.cluster.x-k8s.io/version: v1.13.3
```

→ `rancher/cluster-api-controller:v1.13.3`

### system-upgrade-controller

`package/Dockerfile: ENV CATTLE_SYSTEM_UPGRADE_CONTROLLER_CHART_VERSION=...` → chart `system-upgrade-controller` at that version → `values.yaml`:

- `.systemUpgradeController.image.repository`:`.systemUpgradeController.image.tag`
- `.kubectl.image.repository`:`.kubectl.image.tag`

Example: `package/Dockerfile` at `v2.15.1` has

```dockerfile
ENV CATTLE_SYSTEM_UPGRADE_CONTROLLER_CHART_VERSION=110.0.0
```

→ `https://raw.githubusercontent.com/rancher/charts/release-v2.15/charts/system-upgrade-controller/110.0.0/values.yaml`:

```yaml
systemUpgradeController:
  image:
    repository: rancher/system-upgrade-controller
    tag: v0.20.1

kubectl:
  image:
    repository: rancher/kuberlr-kubectl
    tag: v8.1.1
```

→ `rancher/system-upgrade-controller:v0.20.1`, `rancher/kuberlr-kubectl:v8.1.1`

### system-agent

`package/Dockerfile: ENV CATTLE_SYSTEM_AGENT_VERSION=<version>` → `rancher/system-agent:<version>-suc`.

Example: `package/Dockerfile` at `v2.15.1` has

```dockerfile
ENV CATTLE_SYSTEM_AGENT_VERSION=v0.15.1
```

→ `rancher/system-agent:v0.15.1-suc`
