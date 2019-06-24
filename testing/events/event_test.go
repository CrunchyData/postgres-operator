package eventtest

import (
	"github.com/crunchydata/postgres-operator/events"
	"testing"
)

func TestEventCreate(t *testing.T) {

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
	tryEventChangePasswordUser(t)

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
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventCreateClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			BrokerAddress: EventTCPAddress,
			EventType:     events.EventCreateCluster,
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
	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventCreateClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			BrokerAddress: EventTCPAddress,
			EventType:     events.EventCreateClusterCompleted,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventReloadClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventReloadCluster,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventScaleClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventScaleCluster,
			BrokerAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Replicaname: "somereplicaname",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventScaleDownCluster(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventScaleDownClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventScaleDownCluster,
			BrokerAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Replicaname: "somereplicaname",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventFailoverCluster(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventFailoverClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventFailoverCluster,
			BrokerAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		Target:      "sometarget",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventFailoverClusterCompleted(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventFailoverClusterCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventFailoverClusterCompleted,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventUpgradeClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventUpgradeCluster,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventDeleteClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventDeleteCluster,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventTestClusterFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventTestCluster,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicBackup

	f := events.EventCreateBackupFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreateBackup,
			BrokerAddress: EventTCPAddress,
		},
		Clustername: TestClusterName,
		BackupType:  "pgbackrest",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreateBackupCompleted(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicBackup

	f := events.EventCreateBackupCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreateBackupCompleted,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicUser

	f := events.EventCreateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreateUser,
			BrokerAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
		PostgresPassword: "somepassword",
		Managed:          true,
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventDeleteUser(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicUser

	f := events.EventDeleteUserFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventDeleteUser,
			BrokerAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
		Managed:          true,
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventChangePasswordUser(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicUser

	f := events.EventChangePasswordUserFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventChangePasswordUser,
			BrokerAddress: EventTCPAddress,
		},
		Clustername:      TestClusterName,
		PostgresUsername: TestUsername,
		PostgresPassword: "somepassword",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventCreateLabel(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventCreateLabelFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreateLabel,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventCreatePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreatePolicy,
			BrokerAddress: EventTCPAddress,
		},
		Policyname: "somepolicy",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventApplyPolicy(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventApplyPolicyFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventApplyPolicy,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPolicy

	f := events.EventDeletePolicyFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventDeletePolicy,
			BrokerAddress: EventTCPAddress,
		},
		Policyname: "somepolicy",
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventLoad(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicLoad

	f := events.EventLoadFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventLoad,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicLoad

	f := events.EventLoadCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventLoadCompleted,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventBenchmarkFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventBenchmark,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicCluster

	f := events.EventBenchmarkCompletedFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventBenchmarkCompleted,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPgpool

	f := events.EventCreatePgpoolFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreatePgpool,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPgbouncer

	f := events.EventCreatePgbouncerFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventCreatePgbouncer,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPgbouncer

	f := events.EventDeletePgbouncerFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventDeletePgbouncer,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPgpool

	f := events.EventDeletePgpoolFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventDeletePgpool,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGOCreateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventPGOCreateUser,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGOUpdateUserFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventPGOUpdateUser,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPGOUser

	f := events.EventPGODeleteUserFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventPGODeleteUser,
			BrokerAddress: EventTCPAddress,
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

	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOStartFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventPGOStart,
			BrokerAddress: EventTCPAddress,
		},
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
func tryEventPGOStop(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOStopFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventPGOStop,
			BrokerAddress: EventTCPAddress,
		},
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}

func tryEventPGOUpdateConfig(t *testing.T) {

	topics := make([]string, 1)
	topics[0] = events.EventTopicPGO

	f := events.EventPGOUpdateConfigFormat{
		EventHeader: events.EventHeader{
			Namespace:     Namespace,
			Username:      TestUsername,
			Topic:         topics,
			EventType:     events.EventPGOUpdateConfig,
			BrokerAddress: EventTCPAddress,
		},
	}

	err := events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(f.String())
}
