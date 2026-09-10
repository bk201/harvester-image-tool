# Concepts

`image-tool create-list <subsystem> <version>` produces a deduped, sorted list of container images for one version of a **subsystem**. This page explains the two core concepts the tool is built around: subsystems and components.

## Subsystem

A subsystem is a piece of software whose released container images `image-tool` knows how to derive from upstream sources (e.g. `rancher`).

A subsystem implements `subsystem.Subsystem` ([pkg/subsystem/subsystem.go](../pkg/subsystem/subsystem.go)):

- `Name() string` — the identifier used on the command line (`create-list <name> <version>`).
- `Collect(ctx, Options) ([]ComponentResult, error)` — given a version, discover every image the subsystem pulls in, split out by component.

A subsystem may optionally implement:

- `FlagRegistrar` — to expose its own CLI flags (e.g. an override for which branch of a source repo to read).
- `Verifier` — to cross-check the images it discovered against an authoritative upstream source (e.g. an official release image list). A failed or negative verification is treated as a warning, not a fatal error — it flags a likely bug in the subsystem's discovery logic without blocking the tool from producing output.

Subsystems are registered explicitly — see [main.go](../main.go)'s `registerSubsystems` — there is no implicit, `init()`-based self-registration. Adding a new subsystem means writing a package that implements `Subsystem` and adding one line to `registerSubsystems`.

## Component

Most subsystems are made up of multiple independently-versioned pieces — for example, Rancher ships its own server image alongside separately versioned charts for Fleet, its webhook, etc. A **component** is one such piece.

A subsystem's `Collect` method typically loops over its own list of components, asking each to discover its images:

```go
type Component interface {
    Name() string
    Images(ctx context.Context, rc *Context) ([]string, error)
}
```

(`Context` here is subsystem-specific — see [pkg/subsystem/rancher/context.go](../pkg/subsystem/rancher/context.go) for an example — it carries whatever the subsystem's components need to fetch and parse upstream files.)

Each component is responsible for its own discovery strategy: reading a value out of a manifest file, parsing a chart's `values.yaml`, extracting an `ENV` from a Dockerfile, and so on. Components, like subsystems, are listed explicitly rather than self-registering — see [pkg/subsystem/rancher/component.go](../pkg/subsystem/rancher/component.go).

## Putting it together

```
create-list <subsystem> <version>
  -> subsystem.Collect(version)
       -> component₁.Images()  ─┐
       -> component₂.Images()  ─┼─► every discovered image ref
       -> ...                  ─┘
  -> dedupe + sort (pkg/imagelist.Normalize)
  -> subsystem.Verify(images)   (warns on any that aren't in the official list)
  -> write to stdout or -o file
```

See [subsystem-rancher.md](subsystem-rancher.md) for how the `rancher` subsystem's components calculate their images.

## Standalone Rancher charts

The [`rancher-monitoring`](subsystem-rancher-monitoring.md) and
[`rancher-logging`](subsystem-rancher-logging.md) subsystems take exact chart
versions and require the shared `--chart-branch` flag. Each includes its
corresponding CRD chart at the same version. They read specified extracted
GitHub files and use a built-in Harvester component selection.

Shared command flags are registered once by `create-list` and passed through
`subsystem.Options` (`ChartBranch` for `--chart-branch`). Subsystem-owned flags
retain the `--<subsystem>-<flag>` naming convention. The existing `rancher`
subsystem continues using `--rancher-charts-branch`; it ignores `ChartBranch`.
