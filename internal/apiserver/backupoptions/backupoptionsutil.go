package backupoptions

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	msgs "github.com/crunchydata/postgres-operator/pkg/apiservermsgs"
	"github.com/spf13/pflag"
)

type backupOptions interface {
	validate([]string) error
	getDenyListFlags() ([]string, []string)
}

// ValidateBackupOpts validates the backup/restore options that can be provided to the various backup
// and restore utilities supported by pgo (e.g. pg_dump, pg_restore, pgBackRest, etc.)
func ValidateBackupOpts(backupOpts string, request interface{}) error {

	// some quick checks to make sure backup opts string is valid and should be processed and validated
	if strings.TrimSpace(backupOpts) == "" {
		return nil
	} else if !strings.HasPrefix(strings.TrimSpace(backupOpts), "-") &&
		!strings.HasPrefix(strings.TrimSpace(backupOpts), "--") {
		return errors.New("bad flag syntax. Backup options must start with '-' or '--'")
	} else if strings.TrimSpace(strings.Replace(backupOpts, "-", "", -1)) == "" {
		return errors.New("bad flag syntax. No backup options provided")
	}

	// validate backup opts
	backupOptions, setFlagFieldNames, err := convertBackupOptsToStruct(backupOpts, request)
	if err != nil {
		return err
	} else {
		err := backupOptions.validate(setFlagFieldNames)

		if err != nil {
			return err
		}
	}
	return nil
}

func convertBackupOptsToStruct(backupOpts string, request interface{}) (backupOptions, []string, error) {

	parsedBackupOpts := parseBackupOpts(backupOpts)

	optsStruct, utilityName, err := createBackupOptionsStruct(backupOpts, request)
	if err != nil {
		return nil, nil, err
	}

	structValue := reflect.Indirect(reflect.ValueOf(optsStruct))
	structType := structValue.Type()

	commandLine := pflag.NewFlagSet(utilityName+" backup-opts", pflag.ContinueOnError)
	usage := new(bytes.Buffer)
	commandLine.SetOutput(usage)

	for i := 0; i < structValue.NumField(); i++ {
		fieldVal := structValue.Field(i)

		flag, _ := structType.Field(i).Tag.Lookup("flag")
		flagShort, _ := structType.Field(i).Tag.Lookup("flag-short")

		if flag != "" || flagShort != "" {
			switch fieldVal.Kind() {
			case reflect.String:
				commandLine.StringVarP(fieldVal.Addr().Interface().(*string), flag, flagShort, "", "")
			case reflect.Int:
				commandLine.IntVarP(fieldVal.Addr().Interface().(*int), flag, flagShort, 0, "")
			case reflect.Bool:
				commandLine.BoolVarP(fieldVal.Addr().Interface().(*bool), flag, flagShort, false, "")
			case reflect.Slice:
				commandLine.StringArrayVarP(fieldVal.Addr().Interface().(*[]string), flag, flagShort, nil, "")
			}
		}
	}

	err = commandLine.Parse(parsedBackupOpts)
	if err != nil {
		if customErr := handleCustomParseErrors(err, usage, optsStruct); customErr != nil {
			return nil, nil, customErr
		}
	}

	setFlagFieldNames := obtainSetFlagFieldNames(commandLine, structType)

	return optsStruct, setFlagFieldNames, nil
}

func parseBackupOpts(backupOpts string) []string {

	newFields := []string{}
	var newField string
	for i, c := range backupOpts {
		// if another option is found, add current option to newFields array
		if !(c == ' ' && backupOpts[i+1] == '-') {
			newField = newField + string(c)
		}

		// append if at the end of the flag (i.e. if another new flag was found) or if at the end of the string
		if i == len(backupOpts)-1 || c == ' ' && backupOpts[i+1] == '-' {
			if len(strings.Split(newField, " ")) > 1 && !strings.Contains(strings.Split(newField, " ")[0], "=") {
				splitFlagNoEqualsSign := strings.SplitN(newField, " ", 2)
				if (len(splitFlagNoEqualsSign)) > 1 {
					newFields = append(newFields, strings.TrimSpace(splitFlagNoEqualsSign[0]))
					newFields = append(newFields, strings.TrimSpace(splitFlagNoEqualsSign[1]))
				}
			} else {
				newFields = append(newFields, strings.TrimSpace(newField))
			}
			newField = ""
		}
	}

	return newFields
}

func createBackupOptionsStruct(backupOpts string, request interface{}) (backupOptions, string, error) {

	switch request.(type) {
	case *msgs.CreateBackrestBackupRequest:
		return &pgBackRestBackupOptions{}, "pgBackRest", nil
	case *msgs.RestoreRequest, *msgs.CreateClusterRequest:
		return &pgBackRestRestoreOptions{}, "pgBackRest", nil
	case *msgs.CreatepgDumpBackupRequest:
		if strings.Contains(backupOpts, "--dump-all") {
			return &pgDumpAllOptions{}, "pg_dumpall", nil
		} else {
			return &pgDumpOptions{}, "pg_dump", nil
		}
	case *msgs.PgRestoreRequest:
		return &pgRestoreOptions{}, "pg_restore", nil
	}
	return nil, "", errors.New("Request type not recognized. Unable to create struct for backup opts")
}

func isValidCompressLevel(compressLevel int) bool {
	if compressLevel >= 0 && compressLevel <= 9 {
		return true
	} else {
		return false
	}
}

// isValidRetentionRange validates that pgBackrest Full, Diff or Archive
// retention option value is set within the allowable range.
// allowed: 1-9999999
func isValidRetentionRange(retentionRange int) bool {
	return (retentionRange >= 1 && retentionRange <= 9999999)
}

func isValidValue(vals []string, val string) bool {
	isValid := false
	for _, currVal := range vals {
		if val == currVal {
			isValid = true
			return isValid
		}
	}
	return isValid
}

// this function checks unknown options from the backup-opts flag to validate that they are not denied
// if the option is in the deny list and error is returned, otherwise the flag is unknown to the operator
// and can be passed to pgBackRest for validation.
func handleCustomParseErrors(err error, usage *bytes.Buffer, optsStruct backupOptions) error {
	denyListFlags, denyListFlagsShort := optsStruct.getDenyListFlags()
	if err.Error() == "pflag: help requested" {
		pflag.Usage()
		return errors.New(usage.String())
	} else if strings.Contains(err.Error(), "unknown flag") {
		for _, denyListFlag := range denyListFlags {
			flagMatch, err := regexp.MatchString("\\B"+denyListFlag+"$", err.Error())
			if err != nil {
				return err
			} else if flagMatch {
				return fmt.Errorf("Flag %s is not supported for use with PGO", denyListFlag)
			}
		}
	} else if strings.Contains(err.Error(), "unknown shorthand flag") {
		for _, denyListFlagShort := range denyListFlagsShort {
			denyListFlagQuotes := "'" + strings.TrimPrefix(denyListFlagShort, "-") + "'"
			if strings.Contains(err.Error(), denyListFlagQuotes) {
				return fmt.Errorf("Shorthand flag %s is not supported for use with PGO", denyListFlagShort)
			}
		}
	}
	return nil
}

func obtainSetFlagFieldNames(commandLine *pflag.FlagSet, structType reflect.Type) []string {
	var setFlagFieldNames []string
	var visitBackupOptFlags = func(flag *pflag.Flag) {
		for i := 0; i < structType.NumField(); i++ {
			field := structType.Field(i)
			flagName, _ := field.Tag.Lookup("flag")
			flagNameShort, _ := field.Tag.Lookup("flag-short")
			if flag.Name == flagName || flag.Name == flagNameShort {
				setFlagFieldNames = append(setFlagFieldNames, field.Name)
			}
		}
	}
	commandLine.Visit(visitBackupOptFlags)
	return setFlagFieldNames
}
