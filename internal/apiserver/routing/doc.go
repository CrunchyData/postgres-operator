/*
Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

/* Package routing temporarily breaks circular dependencies within the
structure of the apiserver package

The apiserver package contains a mix of package content (used by external
code) and refactored functionality from the *service folders. The
refactored functionality of the *service folders causes import dependencies
on the apiserver package.

Strictly speaking, the *service folders are an organizational element and
their dependencies could be resolved via dot-import. Idiomatic Go
guidelines point out that using a dot-import outside of testing scenarios is
a sign that package structure needs to be reconsidered and should not be
used outside of the *_test.go scenarios.

Creating this package is preferable to pushing all service-common code into
a 'junk-drawer' package to resolve the circular dependency.

*/
package routing
