package apiservermsgs

import "errors"

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

// CreateBenchmarkResponse ...
type CreateBenchmarkResponse struct {
	Results []string
	Status
}

// CreateBenchmarkRequest ...
type CreateBenchmarkRequest struct {
	Args          []string
	BenchmarkOpts string
	Clients       int
	ClusterName   string
	Database      string
	InitOpts      string
	Jobs          int
	Namespace     string
	Policy        string
	Scale         int
	Selector      string
	Transactions  int
	User          string
}

type DeleteBenchmarkRequest struct {
	Args        []string
	Namespace   string
	ClusterName string
	Selector    string
}

type ShowBenchmarkRequest struct {
	Args        []string
	Namespace   string
	ClusterName string
	Selector    string
}

type DeleteBenchmarkResponse struct {
	Results []string
	Status
}

type ShowBenchmarkResponse struct {
	Results []string
	Status
}

func (c CreateBenchmarkRequest) Validate() error {
	if c.ClusterName == "" && c.Selector == "" {
		return errors.New("cluster name or selector not set")
	}
	return nil
}

func (s ShowBenchmarkRequest) Validate() error {
	if s.ClusterName == "" && s.Selector == "" {
		return errors.New("cluster name or selector not set")
	}
	return nil
}

func (d DeleteBenchmarkRequest) Validate() error {
	if d.ClusterName == "" && d.Selector == "" {
		return errors.New("cluster name or selector not set")
	}
	return nil
}
