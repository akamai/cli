# Akamai CLI
![Build Status](https://github.com/akamai/cli/actions/workflows/checks.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/akamai/cli)](https://goreportcard.com/report/github.com/akamai/cli)
![GitHub release](https://img.shields.io/github/v/release/akamai/cli)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc](https://godoc.org/github.com/akamai/cli?status.svg)](https://pkg.go.dev/github.com/akamai/cli)


Use the Akamai command-line interface (CLI) to configure Akamai's platform and products directly from the command line. You can install ready-to-use product packages or build your own custom solutions to manage from CLI.

## Benefits

- Simple and task-oriented interface.
- Consistent user experience across all Akamai products.
- Wide range of supported packages and capabilities.
- Extend or build your own CLI packages with supported programming languages such as Go, Python, and JavaScript.

## Install base Akamai CLI package

Akamai CLI doesn't have any dependencies and is quick to install. However, you may need an additional runtime for packages depending on the programming language they are based on.

If you're using a Python-based CLI package, install these additional dependencies:

- Python 3.3 or above
- [Python 3 `pip` package installer](https://pip.pypa.io/en/stable/installation)
- [Python 3 `venv` module](https://docs.python.org/3/library/venv.html)
- Up-to-date common CA certificates for your operating system (PEM files)

### Install from binaries

Download a [release binary](https://github.com/akamai/cli/releases) compatible with your operating system.

- **Linux and macOS.** Once you download the appropriate binary for your system, make it executable and move it to a directory you have write access to. Optionally, you can add the directory to your `$PATH` environment variable. Run these commands:

    ```sh
    $ chmod +x ~/Downloads/akamai-<VERSION>-<PLATFORM>
    $ mv ~/Downloads/akamai-<VERSION>-<PLATFORM> /usr/local/bin/akamai
    ```

- **Windows.** Once you download the appropriate binary for your system, you can execute the binary from the command line. For example:

    ```sh
    $ akamai.exe help
    ```

### Install with Homebrew

You can also install Akamai CLI using the Homebrew package manager. If you haven’t used it before, check [Homebrew documentation](https://docs.brew.sh/Installation) for system requirements and read the installation guide.

Once set up, run this command:

```sh
$ brew install akamai
```

This command compiles and globally installs the binary with all necessary dependencies.

### Install with Docker

A container with Akamai CLI and pre-installed public packages is also available in [Docker](http://docker.com).
All images are built using Docker files from the [akamai-docker](https://github.com/akamai/akamai-docker) repository.
You can find all Akamai builds on [Docker Hub](https://hub.docker.com/u/akamai).

To start, create and run the container with Akamai Development Environment:

```sh
$ docker run -it -v $HOME/.edgerc:/root/.edgerc:ro akamai/shell
```

> **Note:** This mounts your local `$HOME/.edgerc` into the container. To change the local path, modify the `-v` argument.

The `akamai` command and basic packages are already installed. See the [akamai-docker](https://github.com/akamai/akamai-docker) repository for more details.

If you want to open Akamai Development Environment when calling the `akamai` command, add this line to your `.bashrc`, `.bash_profile`, or `.zshrc` file:

```sh
alias akamai='docker run -it -v $HOME/.edgerc:/root/.edgerc:ro akamai/shell'
```

If you want to use a local `.akamai-cli` directory to configure and manage your installed packages, modify the `-v` argument:

```sh
$ docker run -it -v $HOME/.akamai-cli:/cli/.akamai-cli akamai/shell
```

This command installs the CLI and persists the configuration and packages in `$HOME/.akamai-docker` directory.

### Compile from source

To compile Akamai CLI from source, you need [Go](https://golang.org/) 1.25.7 or later installed.

1. Change the working directory.

    ```sh
    $ cd $GOPATH
    ```

2. Fetch the package.

    ```sh
    $ git clone github.com/akamai/cli
    ```

3.  Go to the package directory.

    ```sh
    $ cd cli
    ```

4. Compile the binary.

   ```sh
   # Linux, macOS, other Unix-based systems
   go build -o akamai cli/main.go

   # Windows
   go build -o akamai.exe cli/main.go
   ```

5. Move the `akamai` or `akamai.exe` binary so that it's available in your `$PATH`.

> **Tip:** Once you've installed the base CLI, you can expand the functionality by installing [CLI packages](https://github.com/akamai/?q=cli&type=&language=&sort=).

## Authenticate

Akamai-branded packages use a `.edgerc` file for standard EdgeGrid authentication. By default, CLI looks for credentials in your `$HOME` directory.

You can override both the file location or the credentials section by passing the `--edgerc` or `--section` flags to each command.

To set up your `.edgerc` file:

1. [Create authentication credentials](https://techdocs.akamai.com/developer/docs/edgegrid).

2. Place your credentials in an EdgeGrid resource file, `.edgerc`, under a heading of `[default]` at your local home directory.

   ```
    [default]
    client_secret = C113nt53KR3TN6N90yVuAgICxIRwsObLi0E67/N8eRN=
    host = akab-h05tnam3wl42son7nktnlnnx-kbob3i3v.luna.akamaiapis.net
    access_token = akab-acc35t0k3nodujqunph3w7hzp7-gtm6ij
    client_token = akab-c113ntt0k3n4qtari252bfxxbsl-yvsdj
    ```

## Upgrade

Unless you installed Akamai CLI with [Homebrew](#install-with-homebrew), you can enable automatic check for updates when you run Akamai CLI v0.3.0 or later for the first time.

When run for the first time, the CLI asks if you want to enable automatic upgrades. If you don't agree, `last-upgrade-check=ignore` is set in the `.akamai-cli/config` file (this option will still allow you to perform a manual upgrade). Otherwise, if a new version is available, the CLI prompts you to download it. Akamai CLI automatically checks the new version's `SHA256` signature to verify it isn't corrupt. After the update, your original command executes using the new version.

For information on manual upgrade and the supported Homebrew command, see `akamai upgrade` in [Built-in commands](#built-in-commands).

## Use

All CLI commands start with the `akamai` binary, followed by a command, and optionally an action or other arguments to further define the output.

You can optionally provide the path to your `.edgerc` file and credentials section header. If you pass the command without the `--edgerc` and `--section` global flags, the command, by default, will point to the local home directory of your `.edgerc` file and the `default` credentials section header of that file.

If you manage multiple accounts, pass your account switch key using the `--accountkey` global flag.

To use an installed command from the package you installed, run:

```sh
akamai [global flags] [command] [action] [arguments...]
```

Example:

```sh
akamai --edgerc "~/.edgerc" --section "default" --accountkey "A-CCT1234:A-CCT5432" property-manager new-property -p example.org -g grp_12345 -c ctr_C0NT7ACT -d prd_Web_App_Accel
```

### Global flags

Use the global flags to modify the Akamai CLI behavior or get additional information.

| Flag | Description |
| ------ | --------- |
| `--edgerc` (string) | Alias `-e`. The location of your credentials file. The default is `$HOME/.edgerc`. |
| `--section` (string) | Alias `-s`. A credential set's section name. The default is `default`. |
| `--accountkey` (string) | Alias `--account-key`. An account switch key. |
| `--help` (boolean) | Outputs basic usage info and available commands. |
| `--bash` (boolean) | Outputs help on using auto-complete with bash. |
| `--zsh` (boolean) | Outputs help on using auto-complete with zsh. |
| `--proxy` (string) | Sets a proxy to use. |
| `--version` (boolean) | Outputs a version number of currently installed Akamai CLI. |

### Built-in commands

Use the built-in commands to manage packages and the toolkit.

<table>
    <thead>
        <tr>
            <th>Command</th>
            <th>Description</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td><code>help</code></td>
            <td><code>akamai help</code> outputs basic usage info and available commands. To learn more about a specific command, run <code>akamai help <command> [sub-command]</code>.</td>
        </tr>
        <tr>
            <td><code>list</code></td>
            <td><code>akamai list</code> outputs a list of available commands. If a command doesn't display, ensure the binary is executable and in your <code>$PATH</code>.</td>
        </tr>
        <tr>
            <td><code>install</code></td>
            <td>This installs new packages from a git repository.<br/><br/> <code>akamai install {package name or repository URL}</code> downloads and installs the command repository to the <code>$HOME/.akamai-cli</code> directory.<br/><br/> For Github repositories, specify <code>user/repo</code> or <code>organization/repo</code>. For official Akamai packages, you can omit the <code>akamai/cli-</code> prefix. For example, to install <code>akamai/cli-property-manager</code>, run <code>property-manager</code>.<br/><br/> These examples install Akamai CLI for Property Manager from GitHub using various aliases:<br/><br/>
<pre lang="sh">
    akamai install property-manager
    akamai install akamai/cli-property-manager
    akamai install https://github.com/akamai/cli-property-manager.git
</pre>
            </br>The <code>install</code> command accepts more than one argument, so you can install many packages at once using any of these types of syntax.</td>
        </tr>
        <tr>
            <td><code>uninstall</code></td>
            <td>To remove all the package files you installed with <code>akamai install</code>, run <code>akamai uninstall {command}</command></code>, where <code>{command}</code> is any command within that package.<br/><br/> The <code>uninstall</code> command accepts more than one argument, so you can uninstall many packages at once.</td>
        </tr>
        <tr>
            <td><code>update</code></td>
            <td>To update a package you installed with <code>akamai install</code>, run <code>akamai update {command}</command></code>, where <code>{command}</code> is any command within that package.<br/><br/> You can specify multiple packages to update at once. If you don't specify additional arguments, <code>akamai update</code> updates <i>all</i> packages installed with <code>akamai install</code>.</td>
        </tr>
        <tr>
            <td><code>upgrade</code></td>
            <td>Manually upgrade Akamai CLI to the latest version. If you installed Akamai CLI with Homebrew, run this command instead: <code>brew upgrade akamai</code>.</td>
        </tr>
        <tr>
            <td><code>search</code></td>
            <td>Search all the packages published on <a href="https://github.com/akamai/?q=cli&type=&language=&sort=">Akamai GitHub</a> for the submitter string. Searches apply to the package name, alias, and description. Search results appear in the console output.</td>
        </tr>
        <tr>
            <td><code>config</code></td>
            <td>View or modify the configuration settings that drive the common CLI behavior. Akamai CLI maintains a local configuration file in its root directory. The <code>config</code> command supports these sub-commands:
                <ul>
                    <li><code>get</code></li>
                    <li><code>set</code></li>
                    <li><code>set</code></li>
                    <li><code>unset</code> or <code>rm</code></li>
                </ul>
            </td>
        </tr>
    </tbody>
</table>

### Custom commands

Akamai CLI provides a framework for writing custom CLI commands.

Before you start to build your own commands, make sure you meet these prerequisites:

1. The package is available through a Git repository that supports standard SSH public key authentication.
2. The executable is named `akamai-<command>` using dashed-lowercase, or `akamai<Command>` using camelCase.
3. Verify that `akamai-command help` works for you. Ideally, CLI should allow for `akamai-command help <sub-command>`.
4. If you're using Akamai APIs, the executable must support the `.edgerc` format, and must support both `--edgerc` and `--section` flags.
5. If an action fails to complete, the executable exits with a non-zero status code.

As long as the result is executable, you can use any of the supported languages to build your commands, including Python, Go, and JavaScript.

### Logging

To see additional log information, prepend `AKAMAI_LOG=<logging-level>` to any CLI command. You can specify one of these logging levels:

- `fatal`
- `error`
- `warn`
- `info`
- `debug`

For example, to see extra debug information while updating the property-manager package, run:

```sh
AKAMAI_LOG=debug akamai update property-manager
```

Each level is a progressive superset of all previous tiers. The output for `debug` also includes `fatal`, `error`, `warn`, and `info` logs.

If you want to redirect logs to a file, use the `AKAMAI_CLI_LOG_PATH` environmental variable.

```sh
AKAMAI_LOG=debug AKAMAI_CLI_LOG_PATH=akamai.log akamai update property-manager
```

## Dependencies

Akamai CLI supports these package managers that help you automatically install package dependencies:

- Python: `pip` (using `requirements.txt`)
- Go: `go modules`
- JavaScript: `npm` and `yarn`

If you want to use other languages or package managers, make sure you include all dependencies in the package repository.

## Command package metadata

The package you install needs a `cli.json` file. This is where you specify the command language runtime version and define all commands included in package.

### Format

| Parameter | Description|
| ---------- | ---------- |
| `requirements` | Specifies the runtime requirements. You may specify a minimum version number or use the `*` wildcard for any version. Possible requirements are:<ul><li><code>go</code></li><li><code>node</code></li><li><code>python</code></li></ul>|
| `commands` | Lists commands included in the package. Contains:<ul><li><code>name</code>. The command name, used as the executable name.</li><li><code>aliases</code>. An array of aliases that invoke the same command.</li><li><code>version</code>. The command version.</li><li><code>description</code>. A short description for the command.</li><li><code>bin</code>. A URL to fetch a binary package from if it can't be installed from source. It may contain these placeholders:<ul><li><code>{{.Version}}</code>. The command version.</li><li><code>{{.Name}}</code>. The command name.</li><li><code>{{.OS}}</code>. The current operating system, either <code>windows</code>, <code>mac</code>, or <code>linux</code>.</li><li><code>{{.Arch}}</code>. The current OS architecture, either <code>386</code>, <code>amd64</code>, or <code>arm64</code>.</li><li><code>{{.BinSuffix}}</code>. The binary suffix for the current OS: <code>.exe</code> for <code>windows</code>.</li></ul></li></ul> |

### Example

```json
{
  "requirements": {
    "go": "1.8.0"
  },
  "commands": [
    {
      "name": "purge",
      "version": "0.1.0",
      "description": "Purge content from the Edge",
      "bin": "https://github.com/akamai/cli-purge/releases/download/{{.Version}}/akamai-{{.Name}}-{{.OS}}{{.Arch}}{{.BinSuffix}}"
    }
  ]
}
```

## Akamai CLI exit codes

When you complete an operation, Akamai CLI generates one of these exit codes.

| Exit code | Description |
| ----------- | ----------- |
| `0` (Success) | Indicates that the latest command or script executed successfully. |
| `1` (Configuration error) | Indicates an error while loading `AKAMAI_CLI_VERSION` or `AKAMAI_CLI`. |
| `2` (Configuration error) | Indicates an error while creating the `cache directory`. |
| `3` (Configuration error) | Indicates an error while saving the `cache-path`. |
| `5` (Application error) | Indicates an error with the initial setup. Occurs when you run Akamai CLI for the first time.|
| `6` (Syntax error) | Indicates that the latest command or script can't be processed. |
| `7` (Syntax error) | Indicates that the commands in your installed packages have conflicting names. To fix this, add a prefix to the commands that have the same name. |

## Reporting issues

To report an issue or make a suggestion, create a new [GitHub issue](https://github.com/akamai/cli/issues).

## License

Copyright 2026 Akamai Technologies, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use these files except in compliance with the License. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.