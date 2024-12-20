// Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var userAgent string
var versionString string

func initVersion() {
	command := "unknown"
	if len(os.Args) > 0 && len(os.Args[0]) > 0 {
		command = filepath.Base(os.Args[0])
	}
	if len(versionString) > 0 {
		command += "/" + versionString
	}
	userAgent = fmt.Sprintf("%s (%s/%s)", command, runtime.GOOS, runtime.GOARCH)
}
