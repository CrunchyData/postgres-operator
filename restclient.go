package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/crunchydata/postgres-operator/clusterservice"
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
	//testShowCluster()
	//testTestGet()
	testPost()

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

func testShowCluster() {
	url := "http://localhost:8080/clusters/somename?showsecrets=true&other=thing"

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
