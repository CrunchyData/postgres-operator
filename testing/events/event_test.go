package eventtest

import (
	"github.com/crunchydata/postgres-operator/events"
	//	log "github.com/sirupsen/logrus"
	"testing"
)

//import "bytes"
//import "encoding/json"

func TestEventCreate(t *testing.T) {

	//var clientset *kubernetes.Clientset
	//var restclient *rest.RESTClient
	// t.Fatal("not implemented")
	t.Run("setup", func(t *testing.T) {
		t.Log("some setup code")
		_, _ = SetupKube()

	})

	t.Log("starting")

	tryEventReloadCluster(t)
	tryEventCreateCluster(t)
	tryEventCreateClusterCompleted(t)
	tryEventScaleCluster(t)
	tryEventScaleDownCluster(t)
	tryEventFailoverCluster(t)
	tryEventFailoverClusterCompleted(t)
	tryEventDeleteCluster(t)
	tryEventTestCluster(t)
	tryEventCreateLabel(t)
	tryEventLoad(t)
	tryEventLoadCompleted(t)
	tryEventBenchmark(t)
	tryEventBenchmarkCompleted(t)
	tryEventLs(t)
	tryEventCat(t)

	tryEventCreateBackup(t)
	tryEventCreateBackupCompleted(t)

	tryEventCreateUser(t)
	tryEventDeleteUser(t)
	tryEventUpdateUser(t)

	tryEventCreatePolicy(t)
	tryEventApplyPolicy(t)
	tryEventDeletePolicy(t)

	tryEventCreatePgpool(t)
	tryEventDeletePgpool(t)
	tryEventCreatePgbouncer(t)
	tryEventDeletePgbouncer(t)

	tryEventPGOCreateUser(t)
	tryEventPGOUpdateUser(t)
	tryEventPGODeleteUser(t)
	tryEventPGOStart(t)
	tryEventPGOStop(t)
	tryEventPGOUpdateConfig(t)
}

func tryEventCreateCluster(t *testing.T) {

	f := events.EventCreateClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: "betavalue",
	}

	err := events.NewEventCreateCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreateClusterCompleted(t *testing.T) {
	f := events.EventCreateClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: "betavalue",
	}

	err := events.NewEventCreateClusterCompleted(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventReloadCluster(t *testing.T) {

	f := events.EventReloadClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventReloadCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventScaleCluster(t *testing.T) {

	f := events.EventScaleClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventScaleCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventScaleDownCluster(t *testing.T) {

	f := events.EventScaleDownClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventScaleDownCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventFailoverCluster(t *testing.T) {

	f := events.EventFailoverClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventFailoverCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventFailoverClusterCompleted(t *testing.T) {

	f := events.EventFailoverClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventFailoverClusterCompleted(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventUpgradeCluster(t *testing.T) {

	f := events.EventUpgradeClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventUpgradeCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventDeleteCluster(t *testing.T) {

	f := events.EventDeleteClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventDeleteCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventTestCluster(t *testing.T) {

	f := events.EventTestClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventTestCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreateBackup(t *testing.T) {

	f := events.EventCreateBackupFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventCreateBackup(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreateBackupCompleted(t *testing.T) {

	f := events.EventCreateBackupCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventCreateBackupCompleted(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventCreateUser(t *testing.T) {

	f := events.EventCreateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
	}

	err := events.NewEventCreateUser(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventDeleteUser(t *testing.T) {

	f := events.EventDeleteUserFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
	}

	err := events.NewEventDeleteUser(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventUpdateUser(t *testing.T) {

	f := events.EventUpdateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
	}

	err := events.NewEventUpdateUser(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreateLabel(t *testing.T) {

	f := events.EventCreateLabelFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Label:       "somelabel",
	}

	err := events.NewEventCreateLabel(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventCreatePolicy(t *testing.T) {

	f := events.EventCreatePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.NewEventCreatePolicy(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventApplyPolicy(t *testing.T) {

	f := events.EventApplyPolicyFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.NewEventApplyPolicy(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventDeletePolicy(t *testing.T) {

	f := events.EventDeletePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.NewEventDeletePolicy(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventLoad(t *testing.T) {

	f := events.EventLoadFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Loadconfig:  "someloadconfig",
	}

	err := events.NewEventLoad(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventLoadCompleted(t *testing.T) {

	f := events.EventLoadCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Loadconfig:  "someloadconfig",
	}

	err := events.NewEventLoadCompleted(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventBenchmark(t *testing.T) {

	f := events.EventBenchmarkFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventBenchmark(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventBenchmarkCompleted(t *testing.T) {

	f := events.EventBenchmarkCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventBenchmarkCompleted(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventLs(t *testing.T) {

	f := events.EventLsFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventLs(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCat(t *testing.T) {

	f := events.EventCatFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventCat(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreatePgpool(t *testing.T) {

	f := events.EventCreatePgpoolFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventCreatePgpool(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreatePgbouncer(t *testing.T) {

	f := events.EventCreatePgbouncerFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventCreatePgbouncer(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventDeletePgbouncer(t *testing.T) {

	f := events.EventDeletePgbouncerFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventDeletePgbouncer(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventDeletePgpool(t *testing.T) {

	f := events.EventDeletePgpoolFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.NewEventDeletePgpool(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventPGOCreateUser(t *testing.T) {

	f := events.EventPGOCreateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Username: TestUsername,
	}

	err := events.NewEventPGOCreateUser(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventPGOUpdateUser(t *testing.T) {

	f := events.EventPGOUpdateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Username: TestUsername,
	}

	err := events.NewEventPGOUpdateUser(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventPGODeleteUser(t *testing.T) {

	f := events.EventPGODeleteUserFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
		Username: TestUsername,
	}

	err := events.NewEventPGODeleteUser(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventPGOStart(t *testing.T) {

	f := events.EventPGOStartFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
	}

	err := events.NewEventPGOStart(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventPGOStop(t *testing.T) {

	f := events.EventPGOStopFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
	}

	err := events.NewEventPGOStop(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventPGOUpdateConfig(t *testing.T) {

	f := events.EventPGOUpdateConfigFormat{
		EventHeader: events.EventHeader{
			Namespace:   Namespace,
			Username:    TestUsername,
			Topic:       Topics,
			SomeAddress: EventTCPAddress,
		},
	}

	err := events.NewEventPGOUpdateConfig(&f)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
