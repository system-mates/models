# Mate models

This repository contains reusable system models, example models, a container
image that combines them with the Mate runtime, and acceptance tests for the
system-model behavior.

## System Mates repositories

- [Runtime image](https://github.com/orgs/system-mates/packages/container/package/runtime) —
  published Mate executable and runtime engine
- [Models](https://github.com/system-mates/models) — system and example models
- [Demos](https://github.com/system-mates/demos) — runnable compositions
- [Documentation](https://github.com/system-mates/docs) — public user
  documentation

For public development, clone Models, Demos, and Documentation as siblings as described
in the [organization overview](https://github.com/system-mates).

## Repository layout

```text
models/
├── system/models/     reusable system models
├── example/models/    example and demonstration models
├── tests/acceptance/  HTTP acceptance tests against a running Mate
├── tests/models/      small models used only by acceptance scenarios
└── Dockerfile         combined mate/models image
```

System models provide general Mate capabilities such as Portal, Users, Logs,
Data Editor, and Arch. Example models demonstrate application behavior and may
be used by the demos. Test-only models exist solely to exercise a system model
and are not part of the distributed model set.

See the [Model Reference](https://github.com/system-mates/docs/blob/main/reference/model/README.md)
and [Mate Developer Guide](https://github.com/system-mates/docs/blob/main/guides/mate-developer/README.md)
when writing or assembling models.

## Use or build the models image

Pull the published image containing the Mate runtime plus all distributed
models:

```sh
docker pull ghcr.io/system-mates/models:latest
```

To build it from this repository, the Dockerfile uses the published Runtime
image as its base:

```sh
docker build -t mate/models .
```

The resulting image contains both `system/models` and `example/models` beneath
`/mate/models`.

## Run acceptance tests

The acceptance suite requires a Mate executable compatible with the host. Set
`MATE_BINARY` when testing a specific installed runtime:

```sh
MATE_BINARY=/path/to/mate go test ./tests/acceptance -v
```

Runtime maintainers may instead keep the private `runtime` repository beside
`models`; without `MATE_BINARY`, the suite builds `../runtime`. Every scenario
starts real Mate processes with isolated temporary source and data directories.

See [tests/README.md](tests/README.md) for test ownership, coverage, and rules
for test-focused models.

## Run the demos

The Demos repository expects this repository at `../models`. With the three
public repositories in the recommended sibling layout, run:

```sh
docker compose -f demos/compose.yaml up
```
