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

// RootCAIsBad checks that at least one root CA has been generated and that
// all returned certs are CAs and not expired
//
// TODO(tjmoore4): Currently this will return 'true' if any of the parsed certs
// fail a given check. For scenarios where multiple certs may be returned, such
// as in a BYOC/BYOCA, this will need to be handled so we only generate a new
// certificate for our cert if it is the one that fails.
func RootCAIsBad(root *RootCertificateAuthority) bool {
	return !RootIsValid(root)
}
