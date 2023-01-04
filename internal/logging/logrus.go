/*
Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Logrus creates a sink that writes to out using a logrus format. Log entries
// are emitted when their level is at or below verbosity. (Only the most
// important entries are emitted when verbosity is zero.) Error entries get a
// logrus.ErrorLevel, Info entries with verbosity less than debug get a
// logrus.InfoLevel, and Info entries with verbosity of debug or more get a
// logrus.DebugLevel.
func Logrus(out io.Writer, version string, debug, verbosity int) logr.LogSink {
	root := logrus.New()

	root.SetLevel(logrus.TraceLevel)
	root.SetOutput(out)

	root.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	_, module, _, _ := runtime.Caller(0)
	module = strings.TrimSuffix(module, "internal/logging/logrus.go")

	return &sink{
		verbosity: verbosity,

		fnError: func(err error, message string, kv ...interface{}) {
			entry := root.WithField("version", version)
			entry = logrusFields(entry, kv...)

			if v, ok := entry.Data[logrus.ErrorKey]; ok {
				entry.Data["fields."+logrus.ErrorKey] = v
			}
			entry = entry.WithError(err)

			var t interface{ StackTrace() errors.StackTrace }
			if errors.As(err, &t) {
				if st := t.StackTrace(); len(st) > 0 {
					frame, _ := runtime.CallersFrames([]uintptr{uintptr(st[0])}).Next()
					logrusFrame(entry, frame, module)
				}
			}
			entry.Log(logrus.ErrorLevel, message)
		},

		fnInfo: func(level int, message string, kv ...interface{}) {
			entry := root.WithField("version", version)
			entry = logrusFields(entry, kv...)

			if level >= debug {
				entry.Log(logrus.DebugLevel, message)
			} else {
				entry.Log(logrus.InfoLevel, message)
			}
		},
	}
}

// logrusFields structures and adds the key/value interface to the logrus.Entry;
// for instance, if a key is not a string, this formats the key as a string.
func logrusFields(entry *logrus.Entry, kv ...interface{}) *logrus.Entry {
	if len(kv) == 0 {
		return entry
	}
	if len(kv)%2 == 1 {
		kv = append(kv, nil)
	}

	m := make(map[string]interface{}, len(kv)/2)

	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprintf("!(%#v)", kv[i])
		}
		m[key] = kv[i+1]
	}

	return entry.WithFields(m)
}

// logrusFrame adds the file and func to the logrus.Entry,
// for use in logging errors
func logrusFrame(entry *logrus.Entry, frame runtime.Frame, module string) {
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
}
