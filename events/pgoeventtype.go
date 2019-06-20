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
	"fmt"
)

//--------
type EventPGOCreateUserFormat struct {
	EventHeader `json:"eventheader"`
	Username    string `json:"username"`
}

func (p EventPGOCreateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
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

func (p EventPGOUpdateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
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

func (p EventPGODeleteUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGODeleteUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo delete user) %s", lvl.EventHeader, lvl.Username)
	return msg
}

//--------
type EventPGOStartFormat struct {
	EventHeader `json:"eventheader"`
}

func (p EventPGOStartFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOStartFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo start) ", lvl.EventHeader)
	return msg
}

//--------
type EventPGOStopFormat struct {
	EventHeader `json:"eventheader"`
}

func (p EventPGOStopFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOStopFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo stop) ", lvl.EventHeader)
	return msg
}

//--------
type EventPGOUpdateConfigFormat struct {
	EventHeader `json:"eventheader"`
}

func (p EventPGOUpdateConfigFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOUpdateConfigFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo update config) ", lvl.EventHeader)
	return msg
}
