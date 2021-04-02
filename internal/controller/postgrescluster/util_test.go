/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"errors"
	"io"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSafeHash32(t *testing.T) {
	expected := errors.New("whomp")

	_, err := safeHash32(func(io.Writer) error { return expected })
	assert.Equal(t, err, expected)

	stuff, err := safeHash32(func(w io.Writer) error {
		_, _ = w.Write([]byte(`some stuff`))
		return nil
	})
	assert.NilError(t, err)
	assert.Equal(t, stuff, "574b4c7d87", "expected alphanumeric")

	same, err := safeHash32(func(w io.Writer) error {
		_, _ = w.Write([]byte(`some stuff`))
		return nil
	})
	assert.NilError(t, err)
	assert.Equal(t, same, stuff, "expected deterministic hash")
}
