package util

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"os/exec"
	"strings"
)

func RunPsql(password string, hostip string, sqlstring string) {

	cmd := exec.Command("runpsql.sh", password, hostip)

	cmd.Stdin = strings.NewReader(sqlstring)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Error("error in run cmd " + err.Error())
		log.Error(err)
		return
	}

	log.Debugf("runpsql output [%s]\n", out.String())
}
