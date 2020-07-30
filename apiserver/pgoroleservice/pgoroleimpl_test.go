package pgoroleservice

import (
	"fmt"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/apiserver"
)

func TestValidPermissions(t *testing.T) {
	apiserver.PermMap = map[string]string{
		apiserver.CREATE_CLUSTER_PERM:   "yes",
		apiserver.CREATE_PGBOUNCER_PERM: "yes",
	}

	t.Run("with valid permission", func(t *testing.T) {
		perms := apiserver.CREATE_CLUSTER_PERM

		if err := validPermissions(perms); err != nil {
			t.Errorf("%q should be a valid permission", perms)
		}
	})

	t.Run("with multiple valid permissions", func(t *testing.T) {
		perms := fmt.Sprintf("%s,%s", apiserver.CREATE_CLUSTER_PERM, apiserver.CREATE_PGBOUNCER_PERM)

		if err := validPermissions(perms); err != nil {
			t.Errorf("%v should be a valid permission", perms)
		}
	})

	t.Run("with an invalid permission", func(t *testing.T) {
		perms := "bogus"

		if err := validPermissions(perms); err == nil {
			t.Errorf("%q should raise an error", perms)
		}
	})

	t.Run("with a mix of valid and invalid permissions", func(t *testing.T) {
		perms := fmt.Sprintf("%s,%s", apiserver.CREATE_CLUSTER_PERM, "bogus")

		if err := validPermissions(perms); err == nil {
			t.Errorf("%q should raise an error", perms)
		}
	})

	t.Run("with *", func(t *testing.T) {
		perms := "*"

		if err := validPermissions(perms); err != nil {
			t.Errorf("%q should be a valid permission", perms)
		}
	})
}
