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

/**
EventPGOCreateUser
EventPGOUpdateUser
EventPGODeleteUser
EventPGOStart
EventPGOStop
EventPGOReload
*/

//--------
type EventPGOCreateUserFormat struct {
	EventHeader `json:"eventheader"`
	Username    string `json:"username"`
}

func (EventPGOCreateUserFormat) GetEventType() int {
	return EventPGOCreateUser
}
func (p EventPGOCreateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventPGOCreateUser(p *EventPGOCreateUserFormat) error {
	if p.Username == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventPGOCreateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo create user) %s", lvl.EventHeader, lvl.Username)
	return msg
}

//--------
type EventPGOUpdateUserFormat struct {
	EventHeader `json:"eventheader"`
	Username    string `json:"username"`
}

func (EventPGOUpdateUserFormat) GetEventType() int {
	return EventPGOUpdateUser
}
func (p EventPGOUpdateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventPGOUpdateUser(p *EventPGOUpdateUserFormat) error {
	if p.Username == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventPGOUpdateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo update user) %s", lvl.EventHeader, lvl.Username)
	return msg
}

//--------
type EventPGODeleteUserFormat struct {
	EventHeader `json:"eventheader"`
	Username    string `json:"username"`
}

func (EventPGODeleteUserFormat) GetEventType() int {
	return EventPGODeleteUser
}
func (p EventPGODeleteUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventPGODeleteUser(p *EventPGODeleteUserFormat) error {
	if p.Username == "" {
		return errors.New("required fields missing")
	}
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventPGODeleteUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo delete user) %s", lvl.EventHeader, lvl.Username)
	return msg
}

//--------
type EventPGOStartFormat struct {
	EventHeader `json:"eventheader"`
}

func (EventPGOStartFormat) GetEventType() int {
	return EventPGOStart
}
func (p EventPGOStartFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventPGOStart(p *EventPGOStartFormat) error {
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventPGOStartFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo start) ", lvl.EventHeader)
	return msg
}

//--------
type EventPGOStopFormat struct {
	EventHeader `json:"eventheader"`
}

func (EventPGOStopFormat) GetEventType() int {
	return EventPGOStop
}
func (p EventPGOStopFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventPGOStop(p *EventPGOStopFormat) error {
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventPGOStopFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo stop) ", lvl.EventHeader)
	return msg
}

//--------
type EventPGOReloadFormat struct {
	EventHeader `json:"eventheader"`
}

func (EventPGOReloadFormat) GetEventType() int {
	return EventPGOReload
}
func (p EventPGOStopFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func NewEventPGOStop(p *EventPGOStopFormat) error {
	p.EventHeader.EventType = p.GetEventType()
	return p.EventHeader.Validate()
}

func (lvl EventPGOStopFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo stop) ", lvl.EventHeader)
	return msg
}
