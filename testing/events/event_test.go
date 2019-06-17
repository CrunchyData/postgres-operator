package eventtest

import (
	"github.com/crunchydata/postgres-operator/events"
	//	log "github.com/sirupsen/logrus"
	"os"
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

	tryEventCreateCluster(t)
	tryEventReloadCluster(t)
	tryEventScaleCluster(t)
	tryEventScaleDownCluster(t)
	tryEventFailoverCluster(t)
	tryEventCreateUser(t)
	tryEventDeleteUser(t)
	tryEventUpdateUser(t)
	tryEventCreateLabel(t)
	tryEventCreatePolicy(t)
	tryEventApplyPolicy(t)
	tryEventDeletePolicy(t)
	tryEventLoad(t)
	tryEventLs(t)
	tryEventCat(t)
	tryEventCreatePgpool(t)
	tryEventDeletePgpool(t)
	tryEventCreatePgbouncer(t)
	tryEventDeletePgbouncer(t)
}

func tryEventCreateCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreateClusterFormat{
		EventHeader: someheader,
		Clustername: "betavalue",
	}

	err := events.NewEventCreateCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventReloadCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventReloadClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventReloadCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventScaleCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventScaleClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventScaleCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventScaleDownCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventScaleDownClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventScaleDownCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventFailoverCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventFailoverClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventFailoverCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventUpgradeCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventUpgradeClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventUpgradeCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventDeleteCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventDeleteClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventDeleteCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventTestCluster(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventTestClusterFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventTestCluster(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCreateBackup(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreateBackupFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventCreateBackup(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCreateUser(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreateUserFormat{
		EventHeader:      someheader,
		Clustername:      TestClusterName,
		PostgresUsername: Username,
	}

	err := events.NewEventCreateUser(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventDeleteUser(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventDeleteUserFormat{
		EventHeader:      someheader,
		Clustername:      TestClusterName,
		PostgresUsername: Username,
	}

	err := events.NewEventDeleteUser(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventUpdateUser(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventUpdateUserFormat{
		EventHeader:      someheader,
		Clustername:      TestClusterName,
		PostgresUsername: Username,
	}

	err := events.NewEventUpdateUser(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCreateLabel(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreateLabelFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
		Label:       "somelabel",
	}

	err := events.NewEventCreateLabel(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCreatePolicy(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreatePolicyFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.NewEventCreatePolicy(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventApplyPolicy(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventApplyPolicyFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.NewEventApplyPolicy(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventDeletePolicy(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventDeletePolicyFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
		Policyname:  "somepolicy",
	}

	err := events.NewEventDeletePolicy(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventLoad(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventLoadFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
		Loadconfig:  "someloadconfig",
	}

	err := events.NewEventLoad(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventBenchmark(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventBenchmarkFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventBenchmark(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventLs(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventLsFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventLs(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCat(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCatFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventCat(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCreatePgpool(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreatePgpoolFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventCreatePgpool(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventCreatePgbouncer(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventCreatePgbouncerFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventCreatePgbouncer(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventDeletePgbouncer(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventDeletePgbouncerFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventDeletePgbouncer(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
func tryEventDeletePgpool(t *testing.T) {
	someheader := events.EventHeader{
		Namespace:   Namespace,
		Username:    Username,
		Topic:       Topics,
		SomeAddress: EventTCPAddress,
	}

	f := events.EventDeletePgpoolFormat{
		EventHeader: someheader,
		Clustername: TestClusterName,
	}

	err := events.NewEventDeletePgpool(&f)
	if err != nil {
		t.Fatal(err.Error())
		os.Exit(2)
	}

	err = events.Publish(f)
	if err != nil {
		t.Fatal(err.Error())
	} else {
		t.Log(f.String())
	}
}
