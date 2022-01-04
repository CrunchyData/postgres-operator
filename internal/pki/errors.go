package pki

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

import "errors"

var (
	// ErrFunctionNotImplemented is returned if a function that should be set on
	// a struct is not set
	ErrFunctionNotImplemented = errors.New("function not implemented")

	// ErrMissingRequired is returned if a required parameter is missing
	ErrMissingRequired = errors.New("missing required parameter")

	// ErrInvalidCertificateAuthority is returned if a certficate authority (CA)
	// has not been properly generated
	ErrInvalidCertificateAuthority = errors.New("invalid certificate authority")

	// ErrInvalidPEM s returned if encoded data is not a valid PEM block
	ErrInvalidPEM = errors.New("invalid pem encoded data")
)
