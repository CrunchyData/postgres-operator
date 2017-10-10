package util

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"os/exec"
	"strings"
)

func RunPsql(password string, hostip string, sqlstring string) {

	log.Infoln("RunPsql password [" + password + "] hostip=[" + hostip + "] sql=[" + sqlstring + "]")
	cmd := exec.Command("runpsql.sh", password, hostip)

	cmd.Stdin = strings.NewReader(sqlstring)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Error("error in run cmd " + err.Error())
		log.Error(out.String())
		log.Error(stderr.String())
		return
	}

	log.Debugf("runpsql output [%s]\n", out.String())
}
