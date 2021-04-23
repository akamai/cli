# 1.2.1 (April 28, 2021)

## Fixes
* Fixed `PROXY` flag not working correctly in go 1.16
* Fixed old executable not being removed after upgrading on windows

## Enhancements
* `upgrade` command can now be executed with auto upgrades disabled
* Improved error messages on several commands
* Added upgrade command error message for homebrew installation 

# 1.2.0 (March 16, 2021)

## Fixes
* Synced logs with terminal output in most commands.
* Fixed module update issue.

## Enhancements
* Code improvements: unit test coverage improvement and project structure refactoring.
* Glide build tool was dropped in favor of go modules.
* Dockerfile has been moved to [akamai-docker repository](https://github.com/akamai/akamai-docker/).
* Logging: all `TRACE` log messages are now written in `DEBUG` level. Besides, all commands are traced in logs with `START`, `FINISH` or eventually `ERROR`.
* Logging: new `AKAMAI_CLI_LOG_PATH` environment variable to redirect logs to a file.