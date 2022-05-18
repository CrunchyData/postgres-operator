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

package initialize_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/initialize"
)

func TestRestrictedPodSecurityContext(t *testing.T) {
	psc := initialize.RestrictedPodSecurityContext()

	// Kubernetes describes recommended security profiles:
	// - https://docs.k8s.io/concepts/security/pod-security-standards/

	// > The Baseline policy is aimed at ease of adoption for common
	// > containerized workloads while preventing known privilege escalations.
	// > This policy is targeted at application operators and developers of
	// > non-critical applications.
	t.Run("Baseline", func(t *testing.T) {
		assert.Assert(t, psc.SELinuxOptions == nil,
			`Setting custom SELinux options should be disallowed.`)

		assert.Assert(t, psc.Sysctls == nil,
			`Sysctls can disable security mechanisms or affect all containers on a host, and should be disallowed except for an allowed "safe" subset.`)
	})

	// > The Restricted policy is aimed at enforcing current Pod hardening best
	// > practices, at the expense of some compatibility. It is targeted at
	// > operators and developers of security-critical applications, as well as
	// > lower-trust users.
	t.Run("Restricted", func(t *testing.T) {
		if assert.Check(t, psc.RunAsNonRoot != nil) {
			assert.Assert(t, *psc.RunAsNonRoot == true,
				"Containers must be required to run as non-root users.")
		}

		assert.Assert(t, psc.SeccompProfile == nil,
			"The RuntimeDefault seccomp profile must be required, or allow specific additional profiles.")
	})
}

func TestRestrictedSecurityContext(t *testing.T) {
	sc := initialize.RestrictedSecurityContext()

	// Kubernetes describes recommended security profiles:
	// - https://docs.k8s.io/concepts/security/pod-security-standards/

	// > The Baseline policy is aimed at ease of adoption for common
	// > containerized workloads while preventing known privilege escalations.
	// > This policy is targeted at application operators and developers of
	// > non-critical applications.
	t.Run("Baseline", func(t *testing.T) {
		if assert.Check(t, sc.Privileged != nil) {
			assert.Assert(t, *sc.Privileged == false,
				"Privileged Pods disable most security mechanisms and must be disallowed.")
		}

		assert.Assert(t, sc.Capabilities == nil,
			"Adding additional capabilities beyond the default set must be disallowed.")

		assert.Assert(t, sc.SELinuxOptions == nil,
			"Setting custom SELinux options should be disallowed.")

		assert.Assert(t, sc.ProcMount == nil,
			"The default /proc masks are set up to reduce attack surface, and should be required.")
	})

	// > The Restricted policy is aimed at enforcing current Pod hardening best
	// > practices, at the expense of some compatibility. It is targeted at
	// > operators and developers of security-critical applications, as well as
	// > lower-trust users.
	t.Run("Restricted", func(t *testing.T) {
		if assert.Check(t, sc.AllowPrivilegeEscalation != nil) {
			assert.Assert(t, *sc.AllowPrivilegeEscalation == false,
				"Privilege escalation (such as via set-user-ID or set-group-ID file mode) should not be allowed.")
		}

		if assert.Check(t, sc.RunAsNonRoot != nil) {
			assert.Assert(t, *sc.RunAsNonRoot == true,
				"Containers must be required to run as non-root users.")
		}

		assert.Assert(t, sc.SeccompProfile == nil,
			"The RuntimeDefault seccomp profile must be required, or allow specific additional profiles.")
	})

	if assert.Check(t, sc.ReadOnlyRootFilesystem != nil) {
		assert.Assert(t, *sc.ReadOnlyRootFilesystem == true)
	}
}
