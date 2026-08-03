# Mate models

This repository contains reusable system models, example models, a container
image that combines them with the Mate runtime, and acceptance tests for the
system-model behavior.

## System Mates repositories

- [Runtime](https://github.com/system-mates/runtime) — executable and runtime
  engine
- [Models](https://github.com/system-mates/models) — system and example models
- [Demos](https://github.com/system-mates/demos) — runnable compositions
- [Documentation](https://github.com/system-mates/docs) — user and architecture
  documentation

For development across repositories, clone all four as siblings as described
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

See the [Mate YAML language](https://github.com/system-mates/docs/blob/main/users/yaml-language.md)
and [composition guide](https://github.com/system-mates/docs/blob/main/users/application-configuration.md)
when writing or assembling models.

## Build the models image

The combined image is based on the locally named `mate/runtime` image. From the
parent directory of sibling checkouts:

```sh
docker build -t mate/runtime ./runtime
docker build -t mate/models ./models
```

The resulting `mate/models` image contains both `system/models` and
`example/models` beneath `/mate/models`.

## Run acceptance tests

Keep `models` and `runtime` beside each other:

```text
system-mates/
├── runtime/
└── models/
```

Then run:

```sh
go test ./tests/acceptance -v
```

The suite builds `../runtime` and starts real Mate processes with isolated
temporary source and data directories. To test an existing runtime binary
instead, set `MATE_BINARY`:

```sh
MATE_BINARY=/path/to/mate go test ./tests/acceptance -v
```

See [tests/README.md](tests/README.md) for test ownership, coverage, and rules
for test-focused models.

## Run the demos

The Demos repository expects this repository at `../models`. With all four
repositories in the recommended sibling layout and both images built, run:

```sh
docker compose -f demos/compose.yaml up
```
