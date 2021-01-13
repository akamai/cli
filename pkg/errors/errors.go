// Copyright 2018. Akamai Technologies, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors

import (
	"fmt"
)

const (
	ERR_NOT_FOUND                        = "%s Not Found."
	ERR_RUNTIME_NOT_FOUND                = "Unable to locate %s runtime."
	ERR_RUNTIME_NO_VERSION_FOUND         = "%s %s is required to install this command, unable to determine installed version."
	ERR_RUNTIME_MINIMUM_VERSION_REQUIRED = "%s %s is required to install this command, you have %s."
	ERR_PACKAGE_MANAGER_NOT_FOUND        = "Unable to locate package manager (%s) in PATH"
	ERR_PACKAGE_MANAGER_EXEC             = "Unable to execute package manager (%s)."
	ERR_PACKAGE_NEEDS_REINSTALL          = "You must reinstall this package to continue."
	ERR_PACKAGE_COMPILE_FAILURE          = "Unable to build binary (%s)."
)

type ExitError struct {
	exitCode int
	message  interface{}
}

// Error returns the string message, fulfilling the interface required by
// `error`
func (ee *ExitError) Error() string {
	return fmt.Sprintf("%v", ee.message)
}

// ExitCode returns the exit code, fulfilling the interface required by
// `ExitCoder`
func (ee *ExitError) ExitCode() int {
	return ee.exitCode
}

// NewExitError makes a new *ExitError
func NewExitErrorf(exitCode int, format string, a ...interface{}) *ExitError {
	return &ExitError{
		exitCode: exitCode,
		message:  fmt.Sprintf(format, a...),
	}
}
