package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/crunchydata/postgres-operator/backupservice"
	"github.com/crunchydata/postgres-operator/cloneservice"
	"github.com/crunchydata/postgres-operator/clusterservice"
	"github.com/crunchydata/postgres-operator/policyservice"
	"github.com/crunchydata/postgres-operator/upgradeservice"
	//"github.com/crunchydata/postgres-operator/tpr"
	"log"
	"net/http"
	//"net/url"
	//"strings"
)

func main() {
	//phone := "14158586273"
	// QueryEscape escapes the phone string so
	// it can be safely placed inside a URL query
	//safePhone := url.QueryEscape(phone)
	//testShowCluster(false)
	//testTestGet()
	//testPost()
	//testShowCluster(true)
	//testShowBackup(true)
	//testShowPolicy(true)
	//testCreatePolicy()
	//testCreateClone()
	//testScale()
	testShowUpgrade(true)
	testCreateUpgrade()

}

func testPost() {
	url := "http://localhost:8080/clusters"

	cl := new(clusterservice.CreateClusterRequest)
	cl.Name = "newcluster"
	jsonValue, _ := json.Marshal(cl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// For control over HTTP client headers,
	// redirect policy, and other settings,
	// create a Client
	// A Client is an HTTP client
	client := &http.Client{}

	// Send the request via a client
	// Do sends an HTTP request and
	// returns an HTTP response
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	fmt.Printf("%v\n", resp)

}

func testShowCluster(deleteFlag bool) {
	url := "http://localhost:8080/clusters/somename?showsecrets=true&other=thing"

	action := "GET"
	if deleteFlag {
		action = "DELETE"
		fmt.Println("doing delete")
	}
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	var response clusterservice.ShowClusterResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Println("Name = ", response.Items[0].Name)

}

func testTestGet() {
	url := "http://localhost:8080/clusters/test/somename"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	// For control over HTTP client headers,
	// redirect policy, and other settings,
	// create a Client
	// A Client is an HTTP client
	client := &http.Client{}

	// Send the request via a client
	// Do sends an HTTP request and
	// returns an HTTP response
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	// Callers should close resp.Body
	// when done reading from it
	// Defer the closing of the body
	defer resp.Body.Close()

	// Fill the record with the data from the JSON
	var response clusterservice.TestResults

	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Printf("test results %v\n", response.Results)

}
func testShowBackup(deleteFlag bool) {
	url := "http://localhost:8080/backups/somename?showsecrets=true&other=thing"

	action := "GET"
	if deleteFlag {
		action = "DELETE"
		fmt.Println("doing delete")
	}
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	var response backupservice.ShowBackupResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Println("Name = ", response.Items[0].Name)

}
func testShowPolicy(deleteFlag bool) {
	url := "http://localhost:8080/policies/somename?showsecrets=true&other=thing"

	action := "GET"
	if deleteFlag {
		action = "DELETE"
		fmt.Println("doing delete")
	}
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	var response policyservice.ShowPolicyResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Println("Name = ", response.Items[0].Name)

}
func testCreatePolicy() {
	url := "http://localhost:8080/policies"

	cl := new(policyservice.CreatePolicyRequest)
	cl.Name = "newpolicy"
	jsonValue, _ := json.Marshal(cl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// For control over HTTP client headers,
	// redirect policy, and other settings,
	// create a Client
	// A Client is an HTTP client
	client := &http.Client{}

	// Send the request via a client
	// Do sends an HTTP request and
	// returns an HTTP response
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	fmt.Printf("%v\n", resp)

}
func testCreateClone() {
	url := "http://localhost:8080/clones"

	cl := new(cloneservice.CreateCloneRequest)
	cl.Name = "newclone"
	jsonValue, _ := json.Marshal(cl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	fmt.Printf("%v\n", resp)

}

func testScale() {
	url := "http://localhost:8080/clusters/scale/somename"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	defer resp.Body.Close()

	var response clusterservice.ScaleResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Printf("test results %s\n", response.Results)

}
func testShowUpgrade(deleteFlag bool) {
	url := "http://localhost:8080/upgrades/somename?showsecrets=true&other=thing"

	action := "GET"
	if deleteFlag {
		action = "DELETE"
		fmt.Println("doing delete")
	}
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}

	defer resp.Body.Close()

	var response upgradeservice.ShowUpgradeResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Println(err)
	}

	fmt.Println("Name = ", response.Items[0].Name)

}
func testCreateUpgrade() {
	url := "http://localhost:8080/upgrades"

	cl := new(upgradeservice.CreateUpgradeRequest)
	cl.Name = "newupgrae"
	jsonValue, _ := json.Marshal(cl)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal("NewRequest: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Do: ", err)
		return
	}
	fmt.Printf("%v\n", resp)

}
