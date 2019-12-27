package main

import (
	"fmt"
	"github.com/crunchydata/postgres-operator/pgo/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra/doc"
	"path"
	"path/filepath"
	"strings"
)

const fmTemplate = `---
title: "%s"
---
`

func main() {

	fmt.Println("generate CLI markdown")

	filePrepender := func(filename string) string {
		// now := time.Now().Format(time.RFC3339)
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		fmt.Println(base)
		// url := "/commands/" + strings.ToLower(base) + "/"
		return fmt.Sprintf(fmTemplate, strings.ReplaceAll(base, "_", " "))
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/pgo-cli/reference/" + strings.ToLower(base) + "/"
	}

	err := doc.GenMarkdownTreeCustom(cmd.RootCmd, "./", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}
