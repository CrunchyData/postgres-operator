package apiservermsgs

import (
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

type CreatePolicyRequest struct {
	Name      string
	URL       string
	SQL       string
	Namespace string
}
type ApplyResults struct {
	Results []string
}
type ShowPolicyResponse struct {
	PolicyList crv1.PgpolicyList
}
