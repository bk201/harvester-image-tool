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
```

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
