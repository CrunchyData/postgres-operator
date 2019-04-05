package backupoptions

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
	"strconv"
	"strings"
)

var pgBasebackupOptsBlacklist = []string{"--pgdata", "--slot", "--wal-method", "--label", "--dbname", "--host", "--port",
	"--username", "--no-password", "--password", "--version"}

var pgBasebackupOptsBlacklistShort = []string{"-D", "-S", "-X", "-l", "-d", "-h", "-p", "-U", "-w", "-W", "-V"}

type pgBaseBackupOptions struct {
	Format            string `flag:"format" flag-short:"F"`
	MaxRate           string `flag:"max-rate" flag-short:"r"`
	WriteRecover      bool   `flag:"write-recovery" flag-short:"R"`
	NoSlot            bool   `flag:"no-slot"`
	TablespaceMapping string `flag:"tablespace-mapping" flag-short:"T"`
	WalDir            string `flag:"waldir"`
	Gzip              bool   `flag:"gzip" flag-short:"z"`
	Compress          int    `flag:"compress" flag-short:"Z"`
	Checkpoint        string `flag:"checkpoint" flag-short:"c"`
	NoClean           bool   `flag:"no-clean" flag-short:"n"`
	Progress          bool   `flag:"progress" flag-short:"P"`
	NoSync            bool   `flag:"no-sync"`
	Verbose           bool   `flag:"verbose" flag-short:"v"`
	NoVerifyChecksums bool   `flag:"no-verify-checksums"`
	StatusInterval    int    `flag:"status-interval" flag-short:"s"`
}

func (baseBackupOpts pgBaseBackupOptions) validate(setFlagFieldNames []string) error {

	var errstrings []string

	for _, setFlag := range setFlagFieldNames {

		switch setFlag {
		case "Format":
			if !isValidValue([]string{"p", "plain", "t", "tar"}, baseBackupOpts.Format) {
				err := errors.New("Invalid format provided for pg_basebackup backup")
				errstrings = append(errstrings, err.Error())
			}
		case "MaxRate":
			err := validateMaxRate(baseBackupOpts.MaxRate)
			if err != nil {
				errstrings = append(errstrings, err.Error())
			}
		case "Compress":
			if !isValidCompressLevel(baseBackupOpts.Compress) {
				err := errors.New("Invalid compress level for pg_basebackup backup")
				errstrings = append(errstrings, err.Error())
			} else if baseBackupOpts.Format != "tar" {
				err := errors.New("Compress level is only supported when using the tar format for a pg_basebackup backup")
				errstrings = append(errstrings, err.Error())
			}
		case "Checkpoint":
			if !isValidValue([]string{"fast", "spread"}, baseBackupOpts.Checkpoint) {
				err := errors.New("Invalid checkpoint provided for pg_basebackup backup")
				errstrings = append(errstrings, err.Error())
			}
		}
	}

	if len(errstrings) > 0 {
		return errors.New(strings.Join(errstrings, "\n"))
	}

	return nil
}

func validateMaxRate(maxRate string) error {
	var maxRateVal int
	var convErr error
	if strings.HasSuffix(maxRate, "k") {
		maxRateVal, convErr = strconv.Atoi(strings.TrimSuffix(maxRate, "k"))
	} else if strings.HasSuffix(maxRate, "M") {
		maxRateVal, convErr = strconv.Atoi(strings.TrimSuffix(maxRate, "M"))
		maxRateVal = maxRateVal * 1000 // convert MB to KB
	} else {
		maxRateVal, convErr = strconv.Atoi(maxRate)
	}

	if convErr != nil {
		return errors.New("Invalid format for max rate value provided for pg_basebackup backup")
	} else {
		if maxRateVal < 32 || maxRateVal > 1024000 {
			return errors.New("The max rate value provided for pg_basebackup backup is outside of the " +
				"range allowed (between 32KB and 1024MB)")
		}
	}

	return nil
}

func (baseBackupOpts pgBaseBackupOptions) getBlacklistFlags() ([]string, []string) {
	return pgBasebackupOptsBlacklist, pgBasebackupOptsBlacklistShort
}
