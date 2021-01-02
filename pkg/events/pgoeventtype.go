package events

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	EventHeader     `json:"eventheader"`
	CreatedUsername string `json:"createdusername"`
}

func (p EventPGOCreateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOCreateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo create user) %s - created by %s", lvl.EventHeader, lvl.CreatedUsername, lvl.EventHeader.Username)
	return msg
}

//--------
type EventPGOUpdateUserFormat struct {
	EventHeader     `json:"eventheader"`
	UpdatedUsername string `json:"updatedusername"`
}

func (p EventPGOUpdateUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOUpdateUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo update user) %s - updated by %s", lvl.EventHeader, lvl.UpdatedUsername, lvl.EventHeader.Username)
	return msg
}

//--------
type EventPGODeleteUserFormat struct {
	EventHeader     `json:"eventheader"`
	DeletedUsername string `json:"deletedusername"`
}

func (p EventPGODeleteUserFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGODeleteUserFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo delete user) %s - deleted by %s", lvl.EventHeader, lvl.DeletedUsername, lvl.EventHeader.Username)
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

//--------
type EventPGOCreateRoleFormat struct {
	EventHeader     `json:"eventheader"`
	CreatedRolename string `json:"createdrolename"`
}

func (p EventPGOCreateRoleFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOCreateRoleFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo create role) %s - created by %s", lvl.EventHeader, lvl.CreatedRolename, lvl.EventHeader.Username)
	return msg
}

//--------
type EventPGOUpdateRoleFormat struct {
	EventHeader     `json:"eventheader"`
	UpdatedRolename string `json:"updatedrolename"`
}

func (p EventPGOUpdateRoleFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOUpdateRoleFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo update role) %s - updated by %s", lvl.EventHeader, lvl.UpdatedRolename, lvl.EventHeader.Username)
	return msg
}

//--------
type EventPGODeleteRoleFormat struct {
	EventHeader     `json:"eventheader"`
	DeletedRolename string `json:"deletedRolename"`
}

func (p EventPGODeleteRoleFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGODeleteRoleFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo delete role) %s - deleted by %s", lvl.EventHeader, lvl.DeletedRolename, lvl.EventHeader.Username)
	return msg
}

//--------
type EventPGOCreateNamespaceFormat struct {
	EventHeader      `json:"eventheader"`
	CreatedNamespace string `json:"creatednamespace"`
}

func (p EventPGOCreateNamespaceFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGOCreateNamespaceFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo create namespace) %s - created by %s", lvl.EventHeader, lvl.CreatedNamespace, lvl.EventHeader.Username)
	return msg
}

//--------
type EventPGODeleteNamespaceFormat struct {
	EventHeader      `json:"eventheader"`
	DeletedNamespace string `json:"deletednamespace"`
}

func (p EventPGODeleteNamespaceFormat) GetHeader() EventHeader {
	return p.EventHeader
}

func (lvl EventPGODeleteNamespaceFormat) String() string {
	msg := fmt.Sprintf("Event %s - (pgo delete namespace) %s - deleted by %s", lvl.EventHeader, lvl.DeletedNamespace, lvl.EventHeader.Username)
	return msg
}
