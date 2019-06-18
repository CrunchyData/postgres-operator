package events

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
)

const (
	EventTopicCluster   = "clustertopic"
	EventTopicBackup    = "backuptopic"
	EventTopicUser      = "postgresusertopic"
	EventTopicPolicy    = "policytopic"
	EventTopicPgpool    = "pgpooltopic"
	EventTopicPgbouncer = "pgbouncertopio"
	EventTopicPGO       = "pgotopic"
	EventTopicPGOUser   = "pgousertopic"
)
const (
	EventReloadCluster = iota
	EventCreateCluster
	EventScaleCluster
	EventScaleDownCluster
	EventFailoverCluster
	EventUpgradeCluster
	EventDeleteCluster
	EventTestCluster
	EventCreateLabel
	EventDeleteLabel
	EventLoad
	EventBenchmark
	EventLs
	EventCat

	EventCreateBackup

	EventCreateUser
	EventDeleteUser
	EventUpdateUser

	EventCreatePolicy
	EventApplyPolicy
	EventDeletePolicy

	EventCreatePgpool
	EventDeletePgpool
	EventCreatePgbouncer
	EventDeletePgbouncer

	EventPGOCreateUser
	EventPGOUpdateUser
	EventPGODeleteUser
	EventPGOStart
	EventPGOStop
	EventPGOReload
)

type EventHeader struct {
	EventType   int      `json:eventtype`
	Namespace   string   `json:"namespace"`
	Username    string   `json:"username"`
	Topic       []string `json:"topic"`
	SomeAddress string   `json:"someaddress"`
}

func (lvl EventHeader) String() string {
	msg := fmt.Sprintf("Event %d - ns [%s] - user [%s] topics [%v] tcp-address [%s]", lvl.EventType, lvl.Namespace, lvl.Username, lvl.Topic, lvl.SomeAddress)
	return msg
}

func (lvl EventHeader) Validate() error {
	log.Debugf("Validate called header %s ", lvl.String())
	switch lvl.EventType {
	case EventReloadCluster,
		EventCreateCluster,
		EventScaleCluster,
		EventScaleDownCluster,
		EventFailoverCluster,
		EventUpgradeCluster,
		EventDeleteCluster,
		EventTestCluster,
		EventCreateBackup,
		EventCreateUser,
		EventDeleteUser,
		EventUpdateUser,
		EventCreateLabel,
		EventCreatePolicy,
		EventApplyPolicy,
		EventDeletePolicy,
		EventLoad,
		EventBenchmark,
		EventLs,
		EventCat,
		EventCreatePgpool,
		EventDeletePgpool,
		EventCreatePgbouncer,
		EventDeletePgbouncer:
	default:
		msg := fmt.Sprintf("Event %d - not valid", lvl.EventType)
		return errors.New("could not validate event: invalid event type" + msg)
	}

	incomplete := false
	var errMsg string
	if lvl.Username == "" {
		incomplete = true
		errMsg = "username is not set"
	}
	if lvl.Namespace == "" {
		incomplete = true
		errMsg = "namespace is not set"
	}
	if len(lvl.Topic) == 0 {
		errMsg = "topic is not set"
		incomplete = true
	}
	if lvl.SomeAddress == "" {
		errMsg = "SomeAddress is not set"
		incomplete = true
	}

	if incomplete {
		return errors.New(errMsg + lvl.String())
	}

	return nil
}

type EventInterface interface {
	GetEventType() int
	GetHeader() EventHeader
	String() string
}

//--------
type EventReloadClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventReloadClusterFormat) GetEventType() int {
	return EventReloadCluster
}
func (p EventReloadClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventReloadCluster(p *EventReloadClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventReloadClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s - (reload) %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreateClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventCreateClusterFormat) GetEventType() int {
	return EventCreateCluster
}
func NewEventCreateCluster(p *EventCreateClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (p EventCreateClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func (lvl EventCreateClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s - (create cluster) clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventScaleClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventScaleClusterFormat) GetEventType() int {
	return EventScaleCluster
}
func (p EventScaleClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventScaleCluster(p *EventScaleClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventScaleClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (scale) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventScaleDownClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventScaleDownClusterFormat) GetEventType() int {
	return EventScaleDownCluster
}
func (p EventScaleDownClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventScaleDownCluster(p *EventScaleDownClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventScaleDownClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (scaledown) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventFailoverClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventFailoverClusterFormat) GetEventType() int {
	return EventFailoverCluster
}
func (p EventFailoverClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventFailoverCluster(p *EventFailoverClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventFailoverClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (failover) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventUpgradeClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventUpgradeClusterFormat) GetEventType() int {
	return EventUpgradeCluster
}
func NewEventUpgradeCluster(p *EventUpgradeClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}
func (p EventUpgradeClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventUpgradeClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (upgrade) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventDeleteClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventDeleteClusterFormat) GetEventType() int {
	return EventDeleteCluster
}
func NewEventDeleteCluster(p *EventDeleteClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}
func (p EventDeleteClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeleteClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventTestClusterFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventTestClusterFormat) GetEventType() int {
	return EventTestCluster
}
func NewEventTestCluster(p *EventTestClusterFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (p EventTestClusterFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func (lvl EventTestClusterFormat) String() string {
	msg := fmt.Sprintf("Event %s (test) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreateBackupFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventCreateBackupFormat) GetEventType() int {
	return EventCreateBackup
}
func NewEventCreateBackup(p *EventCreateBackupFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}
func (p EventCreateBackupFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateBackupFormat) String() string {
	msg := fmt.Sprintf("Event %s (create backup) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreateUserFormat struct {
	EventHeader      `json:"eventheader"`
	Clustername      string `json:"clustername"`
	PostgresUsername string `json:"postgresusername"`
}

func (EventCreateUserFormat) GetEventType() int {
	return EventCreateUser
}
func NewEventCreateUser(p *EventCreateUserFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername required fields missing")
	}
	if p.PostgresUsername == "" {
		return errors.New("PostgresUsername required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (p EventCreateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s (create user) - clustername %s - postgres user [%s]", lvl.EventHeader, lvl.Clustername, lvl.PostgresUsername)
	return msg
}

//----------------------------
type EventDeleteUserFormat struct {
	EventHeader      `json:"eventheader"`
	Clustername      string `json:"clustername"`
	PostgresUsername string `json:"postgresusername"`
}

func (EventDeleteUserFormat) GetEventType() int {
	return EventDeleteUser
}
func (p EventDeleteUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventDeleteUser(p *EventDeleteUserFormat) error {
	p.EventHeader.EventType = p.GetEventType()
	if p.Clustername == "" {
		return errors.New("Clustername required fields missing")
	}
	if p.PostgresUsername == "" {
		return errors.New("PostgresUsername required fields missing")
	}
	return p.EventHeader.Validate()
}

func (lvl EventDeleteUserFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete user) - clustername %s - postgres user [%s]", lvl.EventHeader, lvl.Clustername, lvl.PostgresUsername)
	return msg
}

//----------------------------
type EventUpdateUserFormat struct {
	EventHeader      `json:"eventheader"`
	Clustername      string `json:"clustername"`
	PostgresUsername string `json:"postgresusername"`
}

func (EventUpdateUserFormat) GetEventType() int {
	return EventUpdateUser
}
func (p EventUpdateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventUpdateUser(p *EventUpdateUserFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	if p.PostgresUsername == "" {
		return errors.New("PostgresUsername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventUpdateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s (update user) - clustername %s - postgres user [%s]", lvl.EventHeader, lvl.Clustername, lvl.PostgresUsername)
	return msg
}

//----------------------------
type EventCreateLabelFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Label       string `json:"label"`
}

func (EventCreateLabelFormat) GetEventType() int {
	return EventCreateLabel
}
func NewEventCreateLabel(p *EventCreateLabelFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	if p.Label == "" {
		return errors.New("Label fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}
func (p EventCreateLabelFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreateLabelFormat) String() string {
	msg := fmt.Sprintf("Event %s (create label) - clustername %s - label [%s]", lvl.EventHeader, lvl.Clustername, lvl.Label)
	return msg
}

//----------------------------
type EventCreatePolicyFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Policyname  string `json:"policyname"`
}

func (EventCreatePolicyFormat) GetEventType() int {
	return EventCreatePolicy
}
func NewEventCreatePolicy(p *EventCreatePolicyFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	if p.Policyname == "" {
		return errors.New("Policyname fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}
func (p EventCreatePolicyFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventCreatePolicyFormat) String() string {
	msg := fmt.Sprintf("Event %s (create policy) - clustername %s - policy [%s]", lvl.EventHeader, lvl.Clustername, lvl.Policyname)
	return msg
}

//----------------------------
type EventDeletePolicyFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Policyname  string `json:"policyname"`
}

func (EventDeletePolicyFormat) GetEventType() int {
	return EventDeletePolicy
}
func NewEventDeletePolicy(p *EventDeletePolicyFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	if p.Policyname == "" {
		return errors.New("Policyname fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}
func (p EventDeletePolicyFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventDeletePolicyFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete policy) - clustername %s - policy [%s]", lvl.EventHeader, lvl.Clustername, lvl.Policyname)
	return msg
}

//----------------------------
type EventApplyPolicyFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Policyname  string `json:"policyname"`
}

func (EventApplyPolicyFormat) GetEventType() int {
	return EventApplyPolicy
}
func (p EventApplyPolicyFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventApplyPolicy(p *EventApplyPolicyFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	if p.Policyname == "" {
		return errors.New("Policyname fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventApplyPolicyFormat) String() string {
	msg := fmt.Sprintf("Event %s (apply policy) - clustername %s - policy [%s]", lvl.EventHeader, lvl.Clustername, lvl.Policyname)
	return msg
}

//----------------------------
type EventLoadFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
	Loadconfig  string `json:"loadconfig"`
}

func (EventLoadFormat) GetEventType() int {
	return EventLoad
}
func (p EventLoadFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventLoad(p *EventLoadFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	if p.Loadconfig == "" {
		return errors.New("Loadconfig fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventLoadFormat) String() string {
	msg := fmt.Sprintf("Event %s (load) - clustername %s - load config [%s]", lvl.EventHeader, lvl.Clustername, lvl.Loadconfig)
	return msg
}

//----------------------------
type EventBenchmarkFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventBenchmarkFormat) GetEventType() int {
	return EventBenchmark
}
func (p EventBenchmarkFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventBenchmark(p *EventBenchmarkFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventBenchmarkFormat) String() string {
	msg := fmt.Sprintf("Event %s (benchmark) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventLsFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventLsFormat) GetEventType() int {
	return EventLs
}
func (p EventLsFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventLs(p *EventLsFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventLsFormat) String() string {
	msg := fmt.Sprintf("Event %s (ls) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCatFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventCatFormat) GetEventType() int {
	return EventLs
}
func (p EventCatFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventCat(p *EventCatFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventCatFormat) String() string {
	msg := fmt.Sprintf("Event %s (cat) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreatePgpoolFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventCreatePgpoolFormat) GetEventType() int {
	return EventCreatePgpool
}
func (p EventCreatePgpoolFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventCreatePgpool(p *EventCreatePgpoolFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventCreatePgpoolFormat) String() string {
	msg := fmt.Sprintf("Event %s (create pgpool) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventDeletePgpoolFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventDeletePgpoolFormat) GetEventType() int {
	return EventDeletePgpool
}
func (p EventDeletePgpoolFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventDeletePgpool(p *EventDeletePgpoolFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventDeletePgpoolFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete pgpool) - clustername %s ", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventCreatePgbouncerFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventCreatePgbouncerFormat) GetEventType() int {
	return EventCreatePgbouncer
}
func (p EventCreatePgbouncerFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventCreatePgbouncer(p *EventCreatePgbouncerFormat) error {
	if p.Clustername == "" {
		return errors.New("Clustername fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventCreatePgbouncerFormat) String() string {
	msg := fmt.Sprintf("Event %s (create pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}

//----------------------------
type EventDeletePgbouncerFormat struct {
	EventHeader `json:"eventheader"`
	Clustername string `json:"clustername"`
}

func (EventDeletePgbouncerFormat) GetEventType() int {
	return EventDeletePgbouncer
}
func (p EventDeletePgbouncerFormat) GetHeader() EventHeader {
	return p.EventHeader
}
func NewEventDeletePgbouncer(p *EventDeletePgbouncerFormat) error {
	if p.Clustername == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventDeletePgbouncerFormat) String() string {
	msg := fmt.Sprintf("Event %s (delete pgbouncer) - clustername %s", lvl.EventHeader, lvl.Clustername)
	return msg
}
