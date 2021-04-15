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

var pgBackRestOptsDenyList = []string{
	"--cmd-ssh",
	"--config",
	"--config-include-path",
	"--config-path",
	"--lock-path",
	"--log-timestamp",
	"--neutral-umask",
	"--no-neutral-umask",
	"--no-online",
	"--online",
	"--pg-host",
	"--pg-host-cmd",
	"--pg-host-config",
	"--pg-host-config-include-path",
	"--pg-host-config-path",
	"--pg-host-port",
	"--pg-host-user",
	"--pg-path",
	"--pg-port",
	"--repo-gcs-bucket",
	"--repo-gcs-endpoint",
	"--repo-gcs-key",
	"--repo-gcs-key-type",
	"--repo-host",
	"--repo-host-cmd",
	"--repo-host-config",
	"--repo-host-config-include-path",
	"--repo-host-config-path",
	"--repo-host-port",
	"--repo-host-user",
	"--repo-path",
	"--repo-s3-bucket",
	"--repo-s3-endpoint",
	"--repo-s3-host",
	"--repo-s3-region",
	"--no-repo-s3-verify-tls",
	"--repo-s3-uri-style",
	"--stanza",
	"--tablespace-map",
	"--tablespace-map-all",
}

type pgBackRestBackupOptions struct {
	ArchiveCopy              bool   `flag:"archive-copy"`
	NoArchiveCopy            bool   `flag:"no-archive-copy"`
	ArchiveTimeout           int    `flag:"archive-timeout"`
	BackupStandby            bool   `flag:"backup-standby"`
	NoBackupStandby          bool   `flag:"no-backup-standby"`
	ChecksumPage             bool   `flag:"checksum-page"`
	NoChecksumPage           bool   `flag:"no-checksum-page"`
	Exclude                  string `flag:"exclude"`
	Force                    bool   `flag:"force"`
	ManifestSaveThreshold    string `flag:"manifest-save-threshold"`
	Resume                   bool   `flag:"resume"`
	NoResume                 bool   `flag:"no-resume"`
	StartFast                bool   `flag:"start-fast"`
	NoStartFast              bool   `flag:"no-start-fast"`
	StopAuto                 bool   `flag:"stop-auto"`
	NoStopAuto               bool   `flag:"no-stop-auto"`
	BackupType               string `flag:"type"`
	BufferSize               string `flag:"buffer-size"`
	CompressLevel            int    `flag:"compress-level"`
	CompressLevelNetwork     int    `flag:"compress-level-network"`
	CompressType             string `flag:"compress-type"`
	DBTimeout                int    `flag:"db-timeout"`
	Delta                    bool   `flag:"no-delta"`
	ProcessMax               int    `flag:"process-max"`
	ProtocolTimeout          int    `flag:"protocol-timeout"`
	LogLevelConsole          string `flag:"log-level-console"`
	LogLevelFile             string `flag:"log-level-file"`
	LogLevelStderr           string `flag:"log-level-stderr"`
	LogSubprocess            bool   `flag:"log-subprocess"`
	RepoRetentionFull        int    `flag:"repo1-retention-full"`
	RepoRetentionDiff        int    `flag:"repo1-retention-diff"`
	RepoRetentionArchive     int    `flag:"repo1-retention-archive"`
	RepoRetentionArchiveType string `flag:"repo1-retention-archive-type"`
}

type pgBackRestRestoreOptions struct {
	DBInclude            string `flag:"db-include"`
	Force                bool   `flag:"force"`
	RecoveryOption       string `flag:"recovery-option"`
	Set                  string `flag:"set"`
	Target               string `flag:"target"`
	TargetAction         string `flag:"target-action"`
	TargetExclusive      bool   `flag:"target-exclusive"`
	NoTargetExclusive    bool   `flag:"no-target-exclusive"`
	TargetTimeline       int    `flag:"target-timeline"`
	RestoreType          string `flag:"type"`
	BufferSize           string `flag:"buffer-size"`
	CompressLevelNetwork int    `flag:"compress-level-network"`
	DBTimeout            int    `flag:"db-timeout"`
	Delta                bool   `flag:"no-delta"`
	ProcessMax           int    `flag:"process-max"`
	ProtocolTimeout      int    `flag:"protocol-timeout"`
	LogLevelConsole      string `flag:"log-level-console"`
	LogLevelFile         string `flag:"log-level-file"`
	LogLevelStderr       string `flag:"log-level-stderr"`
	LogSubprocess        bool   `flag:"log-subprocess"`
}

// validate method runs validation checks against any pgBackrest backup options given when executing
// a pgBackrest backup. As it iterates through the options array, it will call the appropriate
// function to ensure an allowed value has been set, otherwise it will produce an appropriate error
func (backRestBackupOpts pgBackRestBackupOptions) validate(setFlagFieldNames []string) error {
	var errstrings []string

	for _, setFlag := range setFlagFieldNames {
		switch setFlag {
		case "BackupType":
			if !isValidValue([]string{"full", "diff", "incr"}, backRestBackupOpts.BackupType) {
				err := errors.New("Invalid type provided for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "CompressLevel":
			if !isValidCompressLevel(backRestBackupOpts.CompressLevel) {
				err := errors.New("Invalid compress level for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "CompressLevelNetwork":
			if !isValidCompressLevel(backRestBackupOpts.CompressLevelNetwork) {
				err := errors.New("Invalid network compress level for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "CompressType":
			if !isValidCompressType(backRestBackupOpts.CompressType) {
				err := errors.New("Invalid compress type for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "LogLevelConsole":
			if !isValidBackrestLogLevel(backRestBackupOpts.LogLevelConsole) {
				err := errors.New("Invalid log level for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "LogLevelFile":
			if !isValidBackrestLogLevel(backRestBackupOpts.LogLevelFile) {
				err := errors.New("Invalid log level for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "LogLevelStdErr":
			if !isValidBackrestLogLevel(backRestBackupOpts.LogLevelStderr) {
				err := errors.New("Invalid log level for pgBackRest backup")
				errstrings = append(errstrings, err.Error())
			}
		case "RepoRetentionFull":
			if !isValidRetentionRange(backRestBackupOpts.RepoRetentionFull) {
				err := errors.New("Invalid value for pgBackRest full backup retention. Allowed: 1-9999999")
				errstrings = append(errstrings, err.Error())
			}
		case "RepoRetentionDiff":
			if !isValidRetentionRange(backRestBackupOpts.RepoRetentionDiff) {
				err := errors.New("Invalid value for pgBackRest diff backup retention. Allowed: 1-9999999")
				errstrings = append(errstrings, err.Error())
			}
		case "RepoRetentionArchive":
			if !isValidRetentionRange(backRestBackupOpts.RepoRetentionArchive) {
				err := errors.New("Invalid value for pgBackRest archive retention. Allowed: 1-9999999")
				errstrings = append(errstrings, err.Error())
			}
		case "RepoRetentionArchiveType":
			if !isValidValue([]string{"full", "diff", "incr"}, backRestBackupOpts.RepoRetentionArchiveType) {
				err := errors.New("Invalid backup type for pgBackRest WAL retention. Allowed: \"full\", \"diff\", \"incr\"")
				errstrings = append(errstrings, err.Error())
			}
		}
	}

	if len(errstrings) > 0 {
		return errors.New(strings.Join(errstrings, "\n"))
	}

	return nil
}

func (backRestRestoreOpts pgBackRestRestoreOptions) validate(setFlagFieldNames []string) error {
	var errstrings []string

	for _, setFlag := range setFlagFieldNames {
		switch setFlag {
		case "TargetAction":
			if !isValidValue([]string{"pause", "promote", "shutdown"}, backRestRestoreOpts.TargetAction) {
				err := errors.New("Invalid target action provided for pgBackRest restore")
				errstrings = append(errstrings, err.Error())
			}
		case "TargetExclusive":
			if backRestRestoreOpts.RestoreType != "time" && backRestRestoreOpts.RestoreType != "xid" {
				err := errors.New("The target exclusive option is only applicable for a pgBackRest restore " +
					"when type is 'time' or 'xid' ")
				errstrings = append(errstrings, err.Error())
			}
		case "RestoreType":
			validRestoreTypes := []string{"default", "immediate", "name", "xid", "time", "preserve", "none"}
			if !isValidValue(validRestoreTypes, backRestRestoreOpts.RestoreType) {
				err := errors.New("Invalid type provided for pgBackRest restore")
				errstrings = append(errstrings, err.Error())
			}
		case "CompressLevelNetwork":
			if !isValidCompressLevel(backRestRestoreOpts.CompressLevelNetwork) {
				err := errors.New("Invalid network compress level for pgBackRest restore")
				errstrings = append(errstrings, err.Error())
			}
		case "LogLevelConsole":
			if !isValidBackrestLogLevel(backRestRestoreOpts.LogLevelConsole) {
				err := errors.New("Invalid log level for pgBackRest restore")
				errstrings = append(errstrings, err.Error())
			}
		case "LogLevelFile":
			if !isValidBackrestLogLevel(backRestRestoreOpts.LogLevelFile) {
				err := errors.New("Invalid log level for pgBackRest restore")
				errstrings = append(errstrings, err.Error())
			}
		case "LogLevelStdErr":
			if !isValidBackrestLogLevel(backRestRestoreOpts.LogLevelStderr) {
				err := errors.New("Invalid log level for pgBackRest restore")
				errstrings = append(errstrings, err.Error())
			}
		}
	}

	if len(errstrings) > 0 {
		return errors.New(strings.Join(errstrings, "\n"))
	}

	return nil
}

func isValidBackrestLogLevel(logLevel string) bool {
	logLevels := []string{"off", "error", "warn", "info", "detail", "debug", "trace"}
	return isValidValue(logLevels, logLevel)
}

func (backRestBackupOpts pgBackRestBackupOptions) getDenyListFlags() ([]string, []string) {
	return pgBackRestOptsDenyList, nil
}

func (backRestRestoreOpts pgBackRestRestoreOptions) getDenyListFlags() ([]string, []string) {
	return pgBackRestOptsDenyList, nil
}
