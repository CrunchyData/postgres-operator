package postgrescluster

import (
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func emitAdvanceWarning(cluster *v1beta1.PostgresCluster, r *Reconciler) {
	advanceWarning := "Crunchy Postgres for Kubernetes now requires registration for " +
		"operator upgrades. Register now to be ready for your next upgrade. See " +
		r.RegistrationURL + " for details."
	r.Recorder.Event(cluster, corev1.EventTypeWarning, "Register Soon", advanceWarning)
}

func emitEncumbranceWarning(cluster *v1beta1.PostgresCluster, r *Reconciler) {
	encumbranceWarning := "Registration required for Crunchy Postgres for Kubernetes to modify " +
		cluster.Name + ". See " + r.RegistrationURL + " for details."
	r.Recorder.Event(cluster, corev1.EventTypeWarning, "Registration Required", encumbranceWarning)
	addTokenRequiredCondition(cluster)
}

func registrationRequiredStatusFound(cluster *v1beta1.PostgresCluster) bool {
	return cluster.Status.RegistrationRequired != nil
}

func tokenRequiredConditionFound(cluster *v1beta1.PostgresCluster) bool {
	for _, c := range cluster.Status.Conditions {
		if c.Type == v1beta1.TokenRequired {
			return true
		}
	}

	return false
}

func addTokenRequiredCondition(cluster *v1beta1.PostgresCluster) {
	meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
		Type:               v1beta1.TokenRequired,
		Status:             metav1.ConditionTrue,
		Reason:             "TokenRequired",
		Message:            "Reconciliation suspended",
		ObservedGeneration: cluster.GetGeneration(),
	})
}

func addRegistrationRequiredStatus(cluster *v1beta1.PostgresCluster, pgoVersion string) {
	cluster.Status.RegistrationRequired = &v1beta1.RegistrationRequirementStatus{
		PGOVersion: pgoVersion,
	}
}

func shouldEncumberReconciliation(validToken bool, cluster *v1beta1.PostgresCluster, pgoVersion string) bool {
	if validToken {
		return false
	}

	// Get the CPK version that first imposed RegistrationRequired status on this cluster.
	trialStartedWith := config.RegistrationRequiredBy(cluster)
	currentPGOVersion := pgoVersion
	startedLessThanCurrent := semver.Compare(trialStartedWith, currentPGOVersion) == -1

	return startedLessThanCurrent
}
