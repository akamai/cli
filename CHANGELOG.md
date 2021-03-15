# 1.2.0 (March 16th, 2021)

## Fixes
* Synced logs with terminal output in most commands.
* Fixed module update issue.

## Enhancements
* Code improvements: unit test coverage improvement and project structure refactoring.
* Glide build tool was dropped in favor of go modules.
* Dockerfile has been moved to [akamai-docker repository](https://github.com/akamai/akamai-docker/).
* Logging: all `TRACE` log messages are now written in `DEBUG` level. Besides, all commands are traced in logs with `START`, `FINISH` or eventually `ERROR`.
* Logging: new `AKAMAI_CLI_LOG_PATH` environment variable to redirect logs to a file.