# Quake

A task runner and build tool with a simple, readable syntax.

> **Note**: Quake is an internal tool built by the [Miren](https://miren.dev) team. We built it to give us something a little bit nicer than `make`, and also for fun. See [CONTRIBUTING.md](CONTRIBUTING.md) for more context on the project's scope.

## Installation

```bash
go install miren.dev/quake@latest
```

## Quick Start

Create a `Quakefile` in your project:

```
task default => build

task build {
    go build -o myapp .
}

task test {
    go test ./...
}

task clean {
    rm -rf build/
}
```

Run tasks:

```bash
quake          # runs default task
quake build    # runs build task
quake -l       # list all tasks
```

## Features

- Simple, readable task syntax
- Task dependencies (`task build => test`)
- Task arguments (`task deploy(env)`)
- Variables and expressions
- Namespaces for organization
- Go task integration (write tasks in Go)
- Claude AI integration for generating tasks (`quake -g`)

## Usage

```
quake [options] [task] [args...]

Options:
  -l, --list      List all tasks
  -v              Verbose output (show source locations with -l)
  -g, --generate  Generate a new task using Claude AI
  --init          Initialize a new Quakefile using Claude AI
  -f, --file      Path to Quakefile
```

## Syntax

See the included [Quakefile](Quakefile) for a full example.

### Editor Support

A Neovim plugin with syntax highlighting is included in the [nvim/](nvim/) directory.

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
