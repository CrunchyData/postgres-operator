package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("/tmp/runpsql.sh", "SmS7IHQOPb", "10.110.126.221")
	cmd.Stdin = strings.NewReader("select now();")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("output [%s]\n", out.String())
}
