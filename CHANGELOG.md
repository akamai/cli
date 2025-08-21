# RELEASE NOTES

## 2.0.2 (Aug 21, 2025)

### Enhancements

* Updated vulnerable dependencies.

## 2.0.1 (Apr 29, 2025)

### Enhancements

* Migrated to Go `1.23.6` and adopted a semver-compliant Go directive.
* Updated the required Go version to `1.23.6` for cli-terraform compilation.
* Increased number of log messages.
* Updated vulnerable dependencies.

## 2.0.0 (Feb 3, 2025)

### Breaking changes

* Removed the deprecated `force` flag from the `install` and `update` commands.
* Removed support for the `glide` package manager.

### Enhancements

* Migrated to Go `1.22`.
* Changed the logger from `apex` to `slog`. The log output has not been changed.
* Updated vulnerable dependencies.

### Fixes

* Fixed a problem with invisible output in the light background by converting all colors to a monochromatic representation ([GH#196](https://github.com/akamai/cli/issues/196)).
* Improved code by resolving issues reported by linter.

## 1.6.1 (Jan 2, 2025)

* Fixed security vulnerability ([GH#198](https://github.com/akamai/cli/pull/198)).

## 1.6.0 (Sep 5, 2024)

### Enhancements

* Added support to show the `Installed Version` of commands during `search`.
* Updated the list of packages in `packages-list.json` ([GH#192](https://github.com/akamai/cli/issues/192)).
* Removed versions of the packages from `package-list.json`.
* Changed package installation order.
    * CLI will first check if new binaries are available. If the package has no binaries or no valid
      binaries can be found, it will build the package locally.
    * The `--force` flag has been deprecated for both the `install` and `update` commands.
* Migrated to Go `1.21`.
* Updated various dependencies.

### Fixes

* Fixed uninstalling of a command when binaries are not found.

## 1.5.6 (Jan 22, 2024)

### Enhancements

* Migrated to Go `1.20`.
* Enhanced `README.md` with information about global flags.
* Updated various dependencies.

## 1.5.5 (Jun 21, 2023)

### Enhancements

* Improved the way the spinner's output is displayed. NOTE: The spinner will not print output if not attached to a tty.
* Updated the versions and descriptions of the dependencies in the `packages-list.json`.

## 1.5.4 (Mar 16, 2023)

### Enhancements

* Updated various dependencies.
* Updated the `cli-diagnostics` version to `v1.1.0`.

## 1.5.3 (Jan 26, 2023)

### Enhancements

* Improved code quality - resolved issues reported by `golangci-lint`.
* Migrated to Go `1.18`.

### Fixes

* Fixed the `akamai search` command error ([GH#166](https://github.com/akamai/cli/issues/166)).
* Fixed the autocompletion for commands ([GH#165](https://github.com/akamai/cli/issues/165)).

## 1.5.2 (Jul 28, 2022)

### Enhancements

* New help option without arguments.

### Fixes

* Fixed the `akamai update` command failures when the directory `~/.akamai-cli/src/cli-xyz` is in a git detached state.
* Show the correct version for CLI modules which version is set via ldflags.
* Fixed execution of Python submodules on Windows ([GH#159](https://github.com/akamai/cli/issues/159)).
* Fine print update warnings for homebrew installations.
* Fixed failing unit tests on Windows.

## 1.5.1 (Jun 8, 2022)

### Fixes

* The `update` command does not work for some packages, including `cli-terraform`.

## 1.5.0 (May 26, 2022)

### Enhancements

* Added support for a new Apple M1(Darwin ARM64) build ([GH#127](https://github.com/akamai/cli/issues/127)). NOTE: Not all CLI packages currently support Apple M1.

## 1.4.2 (May 11, 2022)

### Fixes

* Handled recent Python versions ([GH#148](https://github.com/akamai/cli/issues/148)).
* Handled the `yes` command input ([GH#136](https://github.com/akamai/cli/issues/136)).
* Purged directories on unit test error.

## 1.4.1 (Mar 24, 2022)

### Fixes

* Refactored a CLI error to debug a statement when virtual environment deactivation fails.
* Refactored CLI documentation by removing usage examples of `cli-property` (decommissioned).

## 1.4.0 (Mar 14, 2022)

### Enhancements

* [IMPORTANT] Refactored Python support, making use of virtual environments to isolate dependencies for each Python package.
  * Refer to `README.md` for new system dependencies.

## 1.3.1 (Dec 8, 2021)

### Enhancements

* Improved a message for updating a CLI version.

## 1.3.0 (Oct 6, 2021)

### Fixes

* Removed an old binary in a PowerShell terminal ([#125](https://github.com/akamai/cli/issues/125)).
* Documented CLI exit codes.
* Reviewed the exit code when trying to install an already installed command ([#83](https://github.com/akamai/cli/issues/83)).

### Enhancements
* Updated a list of installable CLI commands.
* Documented the `--version` flag ([#94](https://github.com/akamai/cli/issues/94)).
* Added an alias with a package prefix to all installed commands to work around possible command name collisions ([#60](https://github.com/akamai/cli/issues/60)).
* Made the `.edgerc` file location configurable ([#81](https://github.com/akamai/cli/issues/81)).

## 1.2.1 (Apr 28, 2021)

### Fixes
* Fixed the `PROXY` flag not working correctly in Go `1.16`.
* Fixed an old executable not being removed after upgrading on Windows.

### Enhancements
* The `upgrade` command can now be executed with the auto-upgrades disabled.
* Improved error messages for several commands.
* Added an upgrade command error message for Homebrew installation. 

## 1.2.0 (Mar 16, 2021)

### Fixes
* Synced logs with terminal output in most commands.
* Fixed module update issue ([#113](https://github.com/akamai/cli/issues/113)).
* Fixed panic when attempting to write on an empty writer ([#116](https://github.com/akamai/cli/issues/116)).

### Enhancements
* Added code improvements: unit test coverage improvement and project structure refactoring.
* Removed the Glide build tool in favor of the Go modules.
* Moved Dockerfile to the [akamai-docker](https://github.com/akamai/akamai-docker/) repository.
* Logging: all `TRACE` log messages are now written in the `DEBUG` level. Besides, all commands are traced in logs with `START`, `FINISH`, or `ERROR`.
* Logging: added a new `AKAMAI_CLI_LOG_PATH` environment variable to redirect logs to a file.
