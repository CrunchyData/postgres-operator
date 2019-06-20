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
			EventType:   events.EventCreateCluster,
		},
		Clustername: "betavalue",
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreateClusterCompleted,
		},
		Clustername: "betavalue",
	}

	err := events.Publish(f)
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
			EventType:   events.EventReloadCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventScaleCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventScaleDownCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventFailoverCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventFailoverClusterCompleted,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventUpgradeCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventDeleteCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventTestCluster,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreateBackup,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreateBackupCompleted,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreateUser,
			SomeAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
	}

	err := events.Publish(f)
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
			EventType:   events.EventDeleteUser,
			SomeAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
	}

	err := events.Publish(f)
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
			EventType:   events.EventUpdateUser,
			SomeAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreateLabel,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Label:       "somelabel",
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreatePolicy,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.Publish(f)
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
			EventType:   events.EventApplyPolicy,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.Publish(f)
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
			EventType:   events.EventDeletePolicy,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.Publish(f)
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
			EventType:   events.EventLoad,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Loadconfig:  "someloadconfig",
	}

	err := events.Publish(f)
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
			EventType:   events.EventLoadCompleted,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Loadconfig:  "someloadconfig",
	}

	err := events.Publish(f)
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
			EventType:   events.EventBenchmark,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventBenchmarkCompleted,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreatePgpool,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventCreatePgbouncer,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventDeletePgbouncer,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventDeletePgpool,
			SomeAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
	}

	err := events.Publish(f)
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
			EventType:   events.EventPGOCreateUser,
			SomeAddress: EventTCPAddress,
		},
		Username: TestUsername,
	}

	err := events.Publish(f)
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
			EventType:   events.EventPGOUpdateUser,
			SomeAddress: EventTCPAddress,
		},
		Username: TestUsername,
	}

	err := events.Publish(f)
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
			EventType:   events.EventPGODeleteUser,
			SomeAddress: EventTCPAddress,
		},
		Username: TestUsername,
	}

	err := events.Publish(f)
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
			EventType:   events.EventPGOStart,
			SomeAddress: EventTCPAddress,
		},
	}

	err := events.Publish(f)
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
			EventType:   events.EventPGOStop,
			SomeAddress: EventTCPAddress,
		},
	}

	err := events.Publish(f)
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
			EventType:   events.EventPGOUpdateConfig,
			SomeAddress: EventTCPAddress,
		},
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
