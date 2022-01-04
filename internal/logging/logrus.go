/*
Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logging

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wojas/genericr"
)

// Logrus creates a function that writes genericr.Entry to out using a logrus
// format. The resulting logrus.Level depends on Entry.Error and Entry.Level:
//	- Entry.Error ≠ nil   → logrus.ErrorLevel
//	- Entry.Level < debug → logrus.InfoLevel
//	- Entry.Level ≥ debug → logrus.DebugLevel
func Logrus(out io.Writer, version string, debug int) genericr.LogFunc {
	root := logrus.New()

	root.SetLevel(logrus.TraceLevel)
	root.SetOutput(out)

	root.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	_, module, _, _ := runtime.Caller(0)
	module = strings.TrimSuffix(module, "internal/logging/logrus.go")

	return func(input genericr.Entry) {
		entry := root.WithField("version", version)
		frame := input.Caller
		level := logrus.InfoLevel

		if input.Level >= debug {
			level = logrus.DebugLevel
		}
		if len(input.Fields) != 0 {
			entry = entry.WithFields(input.FieldsMap())
		}
		if input.Error != nil {
			if v, ok := entry.Data[logrus.ErrorKey]; ok {
				entry.Data["fields."+logrus.ErrorKey] = v
			}
			entry = entry.WithError(input.Error)
			level = logrus.ErrorLevel

			var t interface{ StackTrace() errors.StackTrace }
			if errors.As(input.Error, &t) {
				if st := t.StackTrace(); len(st) > 0 {
					frame, _ = runtime.CallersFrames([]uintptr{uintptr(st[0])}).Next()
				}
			}
		}
		if frame.File != "" {
			filename := strings.TrimPrefix(frame.File, module)
			fileline := fmt.Sprintf("%s:%d", filename, frame.Line)
			if v, ok := entry.Data["file"]; ok {
				entry.Data["fields.file"] = v
			}
			entry.Data["file"] = fileline
		}
		if frame.Function != "" {
			_, function := filepath.Split(frame.Function)
			if v, ok := entry.Data["func"]; ok {
				entry.Data["fields.func"] = v
			}
			entry.Data["func"] = function
		}

		entry.Log(level, input.Message)
	}
}
