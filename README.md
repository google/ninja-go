# ninja-go

Ninja-go was built as part of the [ABFS (Another Build File Service)](https://github.com/google/abfs) ecosystem. This repo alone is a golang reimplementation of the ninja build system (https://ninja-build.org/), which is used by Android.

Parsing, graph construction, scheduling and execution all call into pluggable
backends. Because of this, the same engine can run with different policies for
deciding when a step is up to date and how to run its command:

*   The **classic** backend (`backends/classic`) works like upstream Ninja. It
    uses `.ninja_log` and `.ninja_deps` and runs every command locally.
*   [ABFS](https://github.com/google/abfs) has its own backend. It replaces
    timestamp checks with **content hashes**: for each step, it hashes the
    command, its inputs and the relevant environment, and looks the result up
    in a remote function cache. If there is a hit, ABFS materializes the cached
    outputs and skips the command. If there is a miss, ABFS runs the command,
    traces the files it reads and writes, and records the result for next run.

> [!NOTE]
> This is not an officially supported Google product.

## Building

You need Go 1.26 or later.

```sh
go build -o ninjago ./cmd
```

## Usage

`ninjago` accepts the usual Ninja flags:

```sh
ninjago -C out -j 64 all
```

Tools are available through `-t`, as in Ninja. Run `ninjago -t list` to see
them all. They include the familiar `commands`, `inputs`, `path`, `query` and
`graph`, plus ninja-go extras such as `dump`, `ninja_files` and `rule-query`.
On Linux, the `--psi-cpu`, `--psi-io` and `--psi-mem` flags pause scheduling
of new jobs when pressure stall information (PSI) exceeds the given threshold.

## Testing

```sh
go test ./...
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

Eligibility for the [Google Open Source Software Vulnerability Rewards
Program](https://bughunters.google.com/open-source-security) is determined by
the [Google Open Source Software Vulnerability Reward Program
Rules](https://bughunters.google.com/about/rules/open-source/google-open-source-software-vulnerability-reward-program-rules).

## License

Apache 2.0; see [LICENSE](LICENSE). Code under `third_party/` comes from other
projects and keeps its original license; see
[third_party/siso/frontend/README.md](third_party/siso/frontend/README.md).
