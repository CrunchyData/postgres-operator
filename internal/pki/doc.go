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

// Package pki provides types and functions to support the public key
// infrastructure of the Postgres Operator. It enforces a two layer system
// of certificate authorities and certificates.
//
// NewRootCertificateAuthority() creates a new root CA.
// GenerateLeafCertificate() creates a new leaf certificate.
//
// Certificate and PrivateKey are primitives that can be marshaled.
package pki
