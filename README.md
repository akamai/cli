# Akamai CLI

Akamai CLI is an ever-growing CLI toolkit for working with Akamai's API from the command line.

## Goals

- Simplicity
- Feature-full
- Consistent UX

## Installation

Akamai CLI is itself a Go application, but may rely on sub-commands that can be written using any language.

The easiest way to install Akamai CLI is to download a [Release](https://github,com/akamai-open/akamai-cli/releases) for your platform.

If you want to compile it from source, you will need Go 1.8 or later installed:

1. Clone this repository:  
  `git clone https://github.com/akamai-open/akamai-cli.git`
2. Change to the clone directory:  
  `cd akamai-cli`
3. Compile the binary and move `akamai` to your `PATH`:  
  `go build akamai.go`
4. **OR** Install automatically using Go:  
  `GOBIN=PATH/TO/bin go install akamai.go`

## Usage

All commands start with the `akamai` binary, followed by a `sub-command` which correlates directly to another binary in your path starting with `akamai-` or `akamaiTitlecase` (in the case of node.js binaries).

### Built-in commands

#### Help

Calling `akamai help` will show basic usage info, and available commands. To learn more about a specific sub-command, use `akamai help <command> [sub-command]`.

#### List

Calling `akamai list` will show you a list of available sub-commands. If a command is not shown, ensure that the binary is executable, and in your `PATH`.

#### Get

The `get` command allows you to easily install new sub-commands from a git repository.

Calling `akamai get <repo>` will download and install the command repository to the `$HOME/.akamai-cli` directory.

#### Update

To update a sub-command installed with `akamai get`, you call `akamai update <command>`.

Calling `akamai update` with no arguments will update _all_ commands installed using `akamai get`

#### Sub-commands

To call a sub-command, use `akamai <sub-command> [args]`, e.g.

```sh
akamai property create example.org
```

### Custom commands

Akamai CLI also provides a framework for writing custom CLI commands. There are a few rules:

1. The binary must be named `akamai-<command>` or `akamai<Command>`
2. Help must be visible when you run: `akamai-command help` and ideally, should allow for `akamai-command help <sub-command>`
3. If the action fails to complete, it should return a non-zero status code (however, `akamai` will only return `0` on success or `1` on failure)

You can use _any_ language to build commands, so long as the result is executable — this includes PHP, Python, Ruby, Perl, Java, Golang, JavaScript, and C#.
