package events

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
)

const (
	EventReloadCluster    = 3
	EventCreateCluster    = 4
	EventScaleCluster     = 5
	EventScaleDownCluster = 6
	EventFailoverCluster  = 7
	EventUpgradeCluster   = 8
	EventDeleteCluster    = 9
	EventTestCluster      = 10
	EventCreateBackup     = 11
	EventCreateUser       = 12
	EventDeleteUser       = 13
	EventUpdateUser       = 14
	EventCreateLabel      = 15
	EventCreatePolicy     = 20
	EventApplyPolicy      = 21
	EventDeletePolicy     = 22
	EventLoad             = 30
	EventBenchmark        = 40
	EventLs               = 50
	EventCat              = 60
	EventCreatePgpool     = 70
	EventDeletePgpool     = 71
	EventCreatePgbouncer  = 80
	EventDeletePgbouncer  = 81
)

type EventHeader struct {
	EventType int    `json:eventtype`
	Namespace string `json:"namespace"`
	Username  string `json:"username"`
}

func (lvl EventHeader) String() string {
	msg := fmt.Sprintf("Event %d - ns [%s] - user [%s]", lvl.EventType, lvl.Namespace, lvl.Username)
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
		EventDeleteLabel,
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

	if lvl.Username == "" {
		msg := fmt.Sprintf("Event %d - not valid %s username is not set", lvl.EventType, lvl.Username)
		return errors.New("could not validate event: invalid event type" + msg)
	}
	if lvl.Namespace == "" {
		msg := fmt.Sprintf("Event %d - not valid %s namespace is not set", lvl.EventType, lvl.Namespace)
		return errors.New("could not validate event: invalid event type" + msg)
	}
	return nil
}

type EventInterface interface {
	GetEventType() int
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

func NewEventReloadCluster(p *EventReloadClusterFormat) error {
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
func NewEventScaleCluster(p *EventScaleClusterFormat) error {
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
func NewEventScaleDownCluster(p *EventScaleDownClusterFormat) error {
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
func NewEventFailoverCluster(p *EventFailoverClusterFormat) error {
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
func NewEventDeleteUser(p *EventDeleteUserFormat) error {
	p.EventHeader.EventType = p.GetEventType()
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
func NewEventUpdateUser(p *EventUpdateUserFormat) error {
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
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
func NewEventApplyPolicy(p *EventApplyPolicyFormat) error {
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
func NewEventLoad(p *EventLoadFormat) error {
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventLoadFormat) String() string {
	msg := fmt.Sprintf("Event %s (load) - clustername %s - load config [%s]", lvl.EventHeader, lvl.Clustername, lvl.Loadconfig)
	return msg
}
