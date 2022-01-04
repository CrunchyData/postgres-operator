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

package initialize

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

// ByteMap initializes m when it points to nil.
func ByteMap(m *map[string][]byte) {
	if m != nil && *m == nil {
		*m = make(map[string][]byte)
	}
}

// Int32 returns a pointer to v.
func Int32(v int32) *int32 { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }

// StringMap initializes m when it points to nil.
func StringMap(m *map[string]string) {
	if m != nil && *m == nil {
		*m = make(map[string]string)
	}
}
