package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"path"
	"strings"
	"github.com/spf13/cobra/doc"
	"github.com/crunchydata/postgres-operator/pgo/cmd"
	
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
		fmt.Println(base);
		// url := "/commands/" + strings.ToLower(base) + "/"
		return fmt.Sprintf(fmTemplate, base)
	}

	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/operatorcli/cli/" + strings.ToLower(base) + "/"
	}

	err := doc.GenMarkdownTreeCustom(cmd.RootCmd, "./", filePrepender, linkHandler)
	if err != nil {
		log.Fatal(err)
	}
}


