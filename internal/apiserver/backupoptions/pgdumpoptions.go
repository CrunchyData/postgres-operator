package backupoptions

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
	"errors"
	"strings"
)

var pgDumpRestoreOptsDenyList = []string{
	"--binary-upgrade",
	"--dbname",
	"--host",
	"--no-password",
	"--no-reconnect",
	"--password",
	"--port",
	"--username",
	"--version",
}

var pgDumpRestoreOptsDenyListShort = []string{
	"-R",
	"-d",
	"-h",
	"-p",
	"-U",
	"-w",
	"-W",
	"-V",
}

type pgDumpOptions struct {
	DataOnly                   bool     `flag:"data-only" flag-short:"a"`
	Blobs                      bool     `flag:"blobs" flag-short:"b"`
	NoBlobs                    bool     `flag:"no-blobs" flag-short:"B"`
	Clean                      bool     `flag:"clean" flag-short:"c"`
	Create                     bool     `flag:"create" flag-short:"C"`
	Encoding                   string   `flag:"encoding" flag-short:"E"`
	Jobs                       int      `flag:"jobs" flag-short:"j"`
	Format                     string   `flag:"format" flag-short:"F"`
	Schema                     []string `flag:"schema" flag-short:"n"`
	ExcludeSchema              string   `flag:"exclude-schema" flag-short:"N"`
	Oids                       bool     `flag:"oids" flag-short:"o"`
	NoOwner                    bool     `flag:"no-owner" flag-short:"O"`
	SchemaOnly                 bool     `flag:"schema-only" flag-short:"s"`
	SuperUser                  string   `flag:"superuser" flag-short:"S"`
	Table                      []string `flag:"table" flag-short:"t"`
	ExcludeTable               string   `flag:"exclude-table" flag-short:"T"`
	Verbose                    bool     `flag:"verbose" flag-short:"v"`
	NoPrivileges               bool     `flag:"no-privileges" flag-short:"x"`
	NoACL                      bool     `flag:"no-acl"`
	Compress                   int      `flag:"compress" flag-short:"Z"`
	ColumnInserts              bool     `flag:"column-inserts"`
	AttributeInserts           bool     `flag:"attribute-inserts"`
	DisableDollarQuoting       bool     `flag:"disable-dollar-quoting"`
	DisableTriggers            bool     `flag:"disable-triggers"`
	EnableRowSecurity          bool     `flag:"exclude-row-security"`
	ExcludeTableData           string   `flag:"exclude-table-data"`
	IfExists                   bool     `flag:"if-exists"`
	Inserts                    bool     `flag:"inserts"`
	LockWaitTimeout            string   `flag:"lock-wait-timeout"`
	LoadViaPartitionRoot       bool     `flag:"load-via-partition-root"`
	NoComments                 bool     `flag:"no-comments"`
	NoPublications             bool     `flag:"no-publications"` // PG 10+
	NoSecurityLabels           bool     `flag:"no-security-labels"`
	NoSubscriptions            bool     `flag:"no-subscriptions"` // PG 10+
	NoSync                     bool     `flag:"no-sync"`
	NoTableSpaces              bool     `flag:"no-tablespaces"`
	NoUnloggedTableData        bool     `flag:"no-unlogged-table-data"`
	QuoteAllIdentifiers        bool     `flag:"quote-all-identifiers"`
	Section                    []string `flag:"section"`
	SerializableDeferrable     bool     `flag:"serializable-deferrable"`
	Snapshot                   string   `flag:"snapshot"`
	StrictNames                string   `flag:"strict-names"` // PG 9.6+
	UseSetSessionAuthorization bool     `flag:"use-set-session-authorization"`
	Role                       string   `flag:"role"`
}

type pgDumpAllOptions struct {
	DataOnly                   bool   `flag:"data-only" flag-short:"a"`
	Clean                      bool   `flag:"clean" flag-short:"c"`
	Encoding                   string `flag:"encoding" flag-short:"E"`
	GlobalsOnly                bool   `flag:"globals-only" flag-short:"g"`
	Oids                       bool   `flag:"oids" flag-short:"o"`
	NoOwner                    bool   `flag:"no-owner" flag-short:"O"`
	RolesOnly                  bool   `flag:"roles-only" flag-short:"r"`
	SchemaOnly                 bool   `flag:"schema-only" flag-short:"s"`
	SuperUser                  string `flag:"superuser" flag-short:"S"`
	TablespacesOnly            bool   `flag:"tablespaces-only" flag-short:"t"`
	Verbose                    bool   `flag:"verbose" flag-short:"v"`
	NoPrivileges               bool   `flag:"no-privileges" flag-short:"x"`
	NoACL                      bool   `flag:"no-acl"`
	ColumnInserts              bool   `flag:"column-inserts"`
	AttributeInserts           bool   `flag:"attribute-inserts"`
	DisableDollarQuoting       bool   `flag:"disable-dollar-quoting"`
	DisableTriggers            bool   `flag:"disable-triggers"`
	IfExists                   bool   `flag:"if-exists"`
	Inserts                    bool   `flag:"inserts"`
	LockWaitTimeout            string `flag:"lock-wait-timeout"`
	LoadViaPartitionRoot       bool   `flag:"load-via-partition-root"`
	NoComments                 bool   `flag:"no-comments"`
	NoPublications             bool   `flag:"no-publications"` // PG 10+
	NoRolePasswords            bool   `flag:"no-role-passwords"`
	NoSecurityLabels           bool   `flag:"no-security-labels"`
	NoSubscriptions            bool   `flag:"no-subscriptions"` // PG 10+
	NoSync                     bool   `flag:"no-sync"`
	NoTableSpaces              bool   `flag:"no-tablespaces"`
	NoUnloggedTableData        bool   `flag:"no-unlogged-table-data"`
	QuoteAllIdentifiers        bool   `flag:"quote-all-identifiers"`
	UseSetSessionAuthorization bool   `flag:"use-set-session-authorization"`
	Role                       string `flag:"role"`
	DumpAll                    bool   `flag:"dump-all"` // custom pgo backup opt used for pg_dumpall
}

type pgRestoreOptions struct {
	DataOnly                   bool     `flag:"data-only" flag-short:"a"`
	Clean                      bool     `flag:"clean" flag-short:"c"`
	Create                     bool     `flag:"create" flag-short:"C"`
	ExitOnError                bool     `flag:"exit-on-error" flag-short:"e"`
	Filename                   string   `flag:"filename" flag-short:"f"`
	Format                     string   `flag:"format" flag-short:"F"`
	Index                      []string `flag:"index" flag-short:"I"`
	Jobs                       int      `flag:"jobs" flag-short:"j"`
	List                       bool     `flag:"list" flag-short:"l"`
	UseList                    bool     `flag:"useList" flag-short:"L"`
	Schema                     string   `flag:"schema" flag-short:"n"`
	ExcludeSchema              string   `flag:"exclude-schema" flag-short:"N"`
	NoOwner                    bool     `flag:"no-owner" flag-short:"O"`
	Function                   []string `flag:"function" flag-short:"P"`
	SchemaOnly                 bool     `flag:"schema-only" flag-short:"s"`
	SuperUser                  string   `flag:"superuser" flag-short:"S"`
	Table                      string   `flag:"table" flag-short:"t"`
	Trigger                    []string `flag:"trigger" flag-short:"T"`
	Verbose                    bool     `flag:"verbose" flag-short:"v"`
	NoPrivileges               bool     `flag:"no-privileges" flag-short:"x"`
	NoACL                      bool     `flag:"no-acl"`
	SingleTransaction          bool     `flag:"single-transaction" flag-short:"1"`
	DisableTriggers            bool     `flag:"disable-triggers"`
	EnableRowSecurity          bool     `flag:"enable-row-security"`
	IfExists                   bool     `flag:"if-exists"`
	NoComments                 bool     `flag:"no-comments"`
	NoDataForFailedTables      bool     `flag:"no-data-for-failed-tables"`
	NoPublications             bool     `flag:"no-publications"` // PG 10+
	NoSecurityLabels           bool     `flag:"no-security-labels"`
	NoSubscriptions            bool     `flag:"no-subscriptions"` // PG 10+
	NoTableSpaces              bool     `flag:"no-tablespaces"`
	Section                    []string `flag:"section"`
	StrictNames                string   `flag:"strict-names"` // PG 9.6+
	UseSetSessionAuthorization bool     `flag:"use-set-session-authorization"`
	Role                       string   `flag:"role"`
}

func (dumpOpts pgDumpOptions) validate(setFlagFieldNames []string) error {
	var errstrings []string

	for _, setFlag := range setFlagFieldNames {
		switch setFlag {
		case "Format":
			if !isValidValue([]string{"p", "plain", "c", "custom", "t", "tar"}, dumpOpts.Format) {
				err := errors.New("Invalid format provided for pg_dump backup")
				errstrings = append(errstrings, err.Error())
			}
		case "SuperUser":
			if !dumpOpts.DisableTriggers {
				err := errors.New("The --superuser option is only applicable for a pg_dump backup if the " +
					"--disable-triggers option has also been specified")
				errstrings = append(errstrings, err.Error())
			}
		case "Compress":
			if !isValidCompressLevel(dumpOpts.Compress) {
				err := errors.New("Invalid compress level for pg_dump backup")
				errstrings = append(errstrings, err.Error())
			} else if dumpOpts.Format == "tar" {
				err := errors.New("Compress level is not supported when using the tar format for a pg_dump backup")
				errstrings = append(errstrings, err.Error())
			}
		case "IfExists":
			if !dumpOpts.Clean {
				err := errors.New("The --if-exists option is only valid for a pg_dump backup if the --clean option is " +
					"also specified")
				errstrings = append(errstrings, err.Error())
			}
		case "Section":
			for _, currSection := range dumpOpts.Section {
				if !isValidValue([]string{"pre-data", "data", "post-data"}, currSection) {
					err := errors.New("Invalid section provided for pg_dump backup")
					errstrings = append(errstrings, err.Error())
				}
			}
		}
	}

	if len(errstrings) > 0 {
		return errors.New(strings.Join(errstrings, "\n"))
	}

	return nil
}

func (dumpAllOpts pgDumpAllOptions) validate(setFlagFieldNames []string) error {
	var errstrings []string

	for _, setFlag := range setFlagFieldNames {
		switch setFlag {
		case "SuperUser":
			if !dumpAllOpts.DisableTriggers {
				err := errors.New("The --superuser option is only applicable for a pg_dumpall backup if the " +
					"--disable-triggers option has also been specified")
				errstrings = append(errstrings, err.Error())
			}
		case "IfExists":
			if !dumpAllOpts.Clean {
				err := errors.New("The --if-exists option is only valid for a pg_dumpall backup if the --clean option is " +
					"also specified")
				errstrings = append(errstrings, err.Error())
			}
		}
	}

	if len(errstrings) > 0 {
		return errors.New(strings.Join(errstrings, "\n"))
	}

	return nil
}

func (restoreOpts pgRestoreOptions) validate(setFlagFieldNames []string) error {
	var errstrings []string

	for _, setFlag := range setFlagFieldNames {
		switch setFlag {
		case "Format":
			if !isValidValue([]string{"p", "plain", "c", "custom", "t", "tar"}, restoreOpts.Format) {
				err := errors.New("Invalid format provided for pg_restore restore")
				errstrings = append(errstrings, err.Error())
			}
		case "SuperUser":
			if !restoreOpts.DisableTriggers {
				err := errors.New("The --superuser option is only applicable for a pg_restore restore if the " +
					"--disable-triggers option has also been specified")
				errstrings = append(errstrings, err.Error())
			}
		case "IfExists":
			if !restoreOpts.Clean {
				err := errors.New("The --if-exists option is only valid for a pg_restore restore if the --clean option is " +
					"also specified")
				errstrings = append(errstrings, err.Error())
			}
		case "Section":
			for _, currSection := range restoreOpts.Section {
				if !isValidValue([]string{"pre-data", "data", "post-data"}, currSection) {
					err := errors.New("Invalid section provided for pg_restore restore")
					errstrings = append(errstrings, err.Error())
				}
			}
		}
	}

	if len(errstrings) > 0 {
		return errors.New(strings.Join(errstrings, "\n"))
	}

	return nil
}

func (dumpOpts pgDumpOptions) getDenyListFlags() ([]string, []string) {
	return pgDumpRestoreOptsDenyList, pgDumpRestoreOptsDenyListShort
}

func (dumpAllOpts pgDumpAllOptions) getDenyListFlags() ([]string, []string) {
	return pgDumpRestoreOptsDenyList, pgDumpRestoreOptsDenyListShort
}

func (restoreOpts pgRestoreOptions) getDenyListFlags() ([]string, []string) {
	return pgDumpRestoreOptsDenyList, pgDumpRestoreOptsDenyListShort
}
