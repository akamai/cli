# 1.2.0 (March 16th, 2021)

## Fixes
* Synced logs with terminal output in most commands.
* Fixed module update issue ([#113](https://github.com/akamai/cli/issues/113)).
* Fix panic when attempting to write on an empty writer ([#116](https://github.com/akamai/cli/issues/116))

## Enhancements
* Code improvements: unit test coverage improvement and project structure refactoring.
* Glide build tool was dropped in favor of go modules.
* Dockerfile has been moved to [akamai-docker](https://github.com/akamai/akamai-docker/) repository.
* Logging: all `TRACE` log messages are now written in `DEBUG` level. Besides, all commands are traced in logs with `START`, `FINISH` or eventually `ERROR`.
* Logging: new `AKAMAI_CLI_LOG_PATH` environment variable to redirect logs to a file.