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

package postgres

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestHostBasedAuthentication(t *testing.T) {
	assert.Equal(t, `local all "postgres" peer`,
		NewHBA().Local().User("postgres").Method("peer").String())

	assert.Equal(t, `host all all "::1/128" trust`,
		NewHBA().TCP().Network("::1/128").Method("trust").String())

	assert.Equal(t, `host replication "KD6-3.7" samenet scram-sha-256`,
		NewHBA().TCP().SameNetwork().Replication().
			User("KD6-3.7").Method("scram-sha-256").
			String())

	assert.Equal(t, `hostssl "data" "+admin" all md5  clientcert="verify-ca"`,
		NewHBA().SSL().Database("data").Role("admin").
			Method("md5").Options(map[string]string{"clientcert": "verify-ca"}).
			String())

	assert.Equal(t, `hostnossl all all all reject`,
		NewHBA().NoSSL().Method("reject").String())
}
