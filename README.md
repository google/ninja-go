# ninja-go

ninja-go is a reimplementation of the [Ninja](https://ninja-build.org/) build
system in Go. It reads standard `build.ninja` files, including the extensions
used by Android's Ninja fork (phony and symlink outputs, validations, the
Soong frontend status protocol, and so on). The binary is called `ninjago`.

Parsing, graph construction, scheduling and execution all call into pluggable
backends. Because of this, the same engine can run with different policies for
deciding when a step is up to date and how to run its command:

*   The **classic** backend (`backends/classic`) works like upstream Ninja. It
    uses `.ninja_log` and `.ninja_deps` and runs every command locally.
*   Another Build File Service (ABFS) has its own backend. It replaces
    timestamp checks with **content hashes**: for each step, it hashes the
    command, its inputs and the relevant environment, and looks the result up
    in a remote function cache. If there is a hit, ABFS materializes the cached
    outputs and skips the command. If there is a miss, ABFS runs the command,
    traces the files it reads and writes, and records the result for next time.

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

## Package layout

| Package | Purpose |
| :--- | :--- |
| `ninja` | Data model for the Ninja language: rules, builds, pools, scopes and variable expansion. |
| `parse` | Parallel parser for Ninja files and depfiles. |
| `graph` | Builds the dependency graph from parsed files. |
| `build` | Scheduler and execution engine. |
| `backends` | Backend interfaces and the `classic` implementation. |
| `ninjafiles` | Readers and writers for `.ninja_log` and `.ninja_deps`. |
| `subcommands` | The `-t` tools. |
| `runtime` | Flags, logging, status line, and other shared runtime helpers. |
| `third_party/siso/frontend` | Writer for the Soong frontend status protocol, derived from Siso. |

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
