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

package pki

import "bytes"

// Equal reports whether c and other have the same value.
func (c Certificate) Equal(other Certificate) bool {
	return bytes.Equal(c.Certificate, other.Certificate)
}

// Equal reports whether k and other have the same value.
func (k PrivateKey) Equal(other PrivateKey) bool {
	if k.PrivateKey == nil || other.PrivateKey == nil {
		return k.PrivateKey == other.PrivateKey
	}
	return k.PrivateKey.Equal(other.PrivateKey)
}
