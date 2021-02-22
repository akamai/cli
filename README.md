<h1 align="center">
  <br>
      <img src="assets/screen-1.png">
  <br>
</h1>

# Akamai CLI
[![Go Report Card](https://goreportcard.com/badge/github.com/akamai/cli)](https://goreportcard.com/report/github.com/akamai/cli) [![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fakamai%2Fcli.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fakamai%2Fcli?ref=badge_shield)


Akamai CLI is an ever-growing CLI toolkit that lets you configure Akamai platform and products directly from the command line. You can install ready-to-use product packages or build your own custom solutions to manage from the CLI.

#### Benefits

- Simple and task-oriented interface
- Consistent user experience across all Akamai products
- Wide range of supported packages and capabilities
- Ability to extend or build your own CLI packages with nine supported programming languages, including Golang, Python, Javascript, PHP, and Ruby

#### Available Packages

Browse the list of available packages [here](https://developer.akamai.com/cli).

## Install Akamai CLI

Akamai CLI doesn't have any dependencies and is quick to install. However, you may need additional runtime for the packages as they can be based on different programming languages.

Install Akamai CLI by downloading an applicable [release binary](https://github.com/akamai/cli/releases). Follow the detailed instructions for various operating systems below.

You can also use [Docker](#UsingDocker) or compile the CLI from [source](#compile).

#### Linux and macOS

Once you download the appropriate binary for your system, make it executable, and optionally move it to `$PATH`. In your Linux command line or in macOS terminal, run the following commands:

```sh
$ chmod +x ~/Downloads/akamai-<VERSION>-<PLATFORM>
$ mv ~/Downloads/akamai-<VERSION>-<PLATFORM> /usr/local/bin/akamai
```

You can also install Akamai CLI using [Homebrew](https://brew.sh) package manager:

```sh
$ brew install akamai
```

This command compiles and globally installs the binary with all necessary dependencies.

#### Windows

Once you download the appropriate binary for your system, simply execute the binary from the command line.


#### Install with Docker

A container with Akamai CLI and pre-installed public packages is also available in [Docker](http://docker.com). To start, run the following command:

```sh
$ docker run -ti -v $HOME/.edgerc:/root/.edgerc akamaiopen/cli [arguments]
```

> **Note:** This mounts your local `$HOME/.edgerc`, and `$HOME/.akamai-cli-docker` into the container. To change the local path, modify the `-v` arguments.

If you want to transparently use docker when calling the `akamai` command, add the following code block to your `.bashrc`, `.bash_profile`, or `.zshrc` files:

```bash
function akamai {
    if [[ `docker ps | grep akamai-cli$ | wc -l` -eq 1 ]]; then
        docker exec -it akamai-cli akamai $@;
    elif docker start akamai-cli > /dev/null 2>&1 && sleep 3 && docker exec -it akamai-cli akamai $@; then
        return 0;
    else
        echo "Creating new docker container"
        mkdir -p $HOME/.akamai-cli-docker
        docker create -it -v $HOME/.edgerc:/root/.edgerc -v $HOME/.akamai-cli-docker:/cli --name akamai-cli akamai/cli > /dev/null 2>&1 && akamai $@;
    fi;
}
```

You can then run `akamai [arguments]` command and it automatically creates or re-uses a "persistent" container.

Docker containers are ephemeral and run for as long as the command (PID 1) inside them stays running. To let you re-use the same container, Akamai uses `akamai --daemon` command that runs indefinitely inside the container.

To restart the container created by the function above, you can safely run `docker stop akamai-cli` followed by `docker start akamai-cli`.

The script above persists your Akamai CLI installation with configuration and packages in the `$HOME/.akamai-cli-docker` directory.

#### Compile from Source

**Prerequisite:** Make sure you install Go 1.7 or later, and the [Glide](https://glide.sh) package manager.

To compile Akamai CLI from source:

1. Fetch the package:  
  `go get github.com/akamai/cli`
2. Go to the package directory:  
  `cd $GOPATH/src/github.com/akamai/cli`
3. Install dependencies using Glide:  
  `glide install`
4. Compile the binary:  
  - For Linux/macOS/*nix, run: `go build -o akamai`
  - For Windows, run: `go build -o akamai.exe`
5. Move the binary (`akamai` or `akamai.exe`) in to your `$PATH`

#### Credentials File

Akamai CLI uses the standard EdgeGrid credentials file, `.edgerc`, to authenticate to the Akamai platform.

To set up your credential file, see [Create an API Client](https://developer.akamai.com/api/getting-started#createanapiclient) and [Add credential to .edgerc file](https://developer.akamai.com/api/getting-started#addcred).

The default location for the credentials file is your `HOME` directory. To override both the default credentials file location and the section with your client tokens, pass the the `--edgerc` or `--section` flags to each command.

## Upgrade to a newer version

You can enable automatic check for updates when you run Akamai CLI v0.3.0 or later for the first time. Akamai CLI also checks the `SHA256` signature of the new version to verify it's valid.

If a new version is available, CLI prompts you to download it. After the update, your original command is executed using the _new_ version.

To perform manual upgrade, see `akamai upgrade`

## How to use Akamai CLI

All CLI commands start with the `akamai` binary, followed by a `command`, and optionally an action or other arguments.

```
akamai [command] [action] [arguments]
```

#### Built-in commands

Use the following commands to manage the packages and the toolkit itself:

- `help`

    `akamai help` shows basic usage info, and available commands. To learn more about a specific command, run `akamai help <command> [sub-command]`

- `list`

    `akamai list` shows you a list of available commands. If a command doesn't display, ensure the binary is executable, and in your `PATH`.

- `install`

    The `install` command lets you install new packages from a git repository.

    `akamai install <package name or repository URL>` downloads and installs the command repository to the `$HOME/.akamai-cli` directory.

    For Github repositories, specify `user/repo` or `organization/repo`. For official Akamai packages, you can omit the `akamai/cli-` prefix, so to install `akamai/cli-property` it's enough to run `property`.

    The following example installs Akamai CLI for Property Manager from Github using various aliases:

    ```
    akamai install property
    akamai install akamai/cli-property
    akamai install https://github.com/akamai/cli-property.git
    ```

    You can specify _multiple_ packages to install at once.

- `uninstall`

    To uninstall a package you installed with `akamai install`, run `akamai uninstall <command>`, where `<command>` is any command within that package.

    You can specify _multiple_ packages to uninstall at once.

- `update`

    To update a package you installed with `akamai install`, run `akamai update <command>`, where `<command>` is any command within that package.

    You can specify _multiple_ packages to update at once.

    If you don't specify additional arguments, `akamai update` updates _all_ packages installed with `akamai install`

- `upgrade`

  Manually upgrade Akamai CLI to the latest version.

#### Installed commands

This type of commands depends on the packages you installed. To use an installed command, run `akamai <command> [arguments]`, e.g.

```sh
akamai property create example.org
```
For the list of supported commands, see the [documentation](https://developer.akamai.com/cli-packages) for your package.

#### Custom commands

Akamai CLI provides a framework for writing custom CLI commands. These commands are contained in packages, which may include one or more commands.

Before you start building your own commands, make sure you meet the following prerequisites:

1. The package must be available through a Git repository that supports standard SSH public key authentication.
2. The executable must be named `akamai-<command>` or `akamai<Command>`
3. Verify that `akamai-command help` works for you. Ideally, CLI should allow for `akamai-command help <sub-command>`
4. If you're using OPEN APIs, the executable must support the `.edgerc` format, and must support both `--edgerc` and `--section` flags
5. If an action fails to complete, the executable should exit with a non-zero status code (however, `akamai` will only return `0` on success or `1` on failure)

As long as the result is executable, you can use _any_ of the supported languages to build your commands, including **PHP, Python, Ruby, Perl, Java, Golang, JavaScript,** and **C#**.

#### Debugging

To see additional debug information, prepend `AKAMAI_LOG=<debug-level>` to any CLI command. You can specify one of the following debug levels:

- panic
- fatal
- error
- warn
- info
- debug
- trace (for full logging)

For example, to see extra debug information while updating the property package, run:
```sh
AKAMAI_LOG=trace akamai update property
```

## Dependencies

Akamai CLI supports the following package managers that help you automatically install package dependencies:

- PHP: composer
- Python: pip (using requirements.txt)
- Ruby: bundler
- Golang: Glide
- JavaScript: npm and yarn

If you want to use other languages or package managers, make sure you include all dependencies in the package repository.

## Command package metadata

The package you install *must* include a `cli.json` file. This is where you specify the command language runtime version and define all commands included in package.

##### Example

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

##### Format

- `requirements`: Specifies runtime requirements. You may specify a minimum version number or use `*` for any version. Possible requirements are:
  - `go`
  - `php`
  - `ruby`
  - `node`
  - `python`
- `commands`: Lists commands included in the package.
  - `name`: The command name (used as the executable name).
  - `aliases`: An array of aliases that can be used to invoke the command.
  - `version`: The command version.
  - `description`: A short description for the command.
  - `bin`: A url to fetch a binary package from if it cannot be installed from source.

    The `bin` URL may contain the following placeholders:

    - `{{.Version}}`: The command version.
    - `{{.Name}}`: The command name
    - `{{.OS}}`: The current operating system, either `windows`, `mac`, or `linux`
    - `{{.Arch}}`: The current OS architecture, either `386` or `amd64`
    - `{{.BinSuffix}}`: The binary suffix for the current OS: `.exe` for `windows`


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fakamai%2Fcli.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fakamai%2Fcli?ref=badge_large)
