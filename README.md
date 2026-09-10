# image-tool

A CLI for generating container image lists for subsystems, derived mechanically from their upstream sources.

```
image-tool create-list <subsystem> <version> [-o output-file]
```

See [doc/create-list.md](doc/create-list.md) for the subsystem/component concepts, and [doc/subsystem-rancher.md](doc/subsystem-rancher.md) for how the `rancher` subsystem calculates its image list.

## Usage

```sh
image-tool create-list rancher v2.15.1
image-tool create-list rancher v2.15.1 -o rancher-images.txt
image-tool create-list rancher-monitoring '109.0.3+up80.9.1-rancher.14' --chart-branch release-v2.15
image-tool create-list rancher-logging '109.0.0+up4.10.0-rancher.23' --chart-branch release-v2.15
```

See [monitoring discovery](doc/subsystem-rancher-monitoring.md) and [logging discovery](doc/subsystem-rancher-logging.md) for the built-in Harvester component selection. These two subsystems take chart versions, require `--chart-branch`, and do not implement release-list verification.

By default, generated images are cross-checked against the subsystem's official release image list; discrepancies are printed as warnings on stderr. Use `--no-verify` to skip the check, or `--strict` to make missing images a hard failure.

## Adding a subsystem

Implement `subsystem.Subsystem` (see [pkg/subsystem/subsystem.go](pkg/subsystem/subsystem.go)) and add an explicit `subsystem.Register(...)` call for it in `registerSubsystems` in [main.go](main.go). Look at [pkg/subsystem/rancher](pkg/subsystem/rancher) for a full example, including its own `components` list ([component.go](pkg/subsystem/rancher/component.go)) for discovering images per component — new components are added there the same explicit way.

## Development

```sh
make build   # build ./bin/image-tool
make test    # go test ./...
make validate # golangci-lint + gofmt, if golangci-lint is installed
make ci      # build + test + validate
```
