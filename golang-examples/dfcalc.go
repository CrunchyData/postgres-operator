/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

func main() {

	diskSizeString := "5Gi"
	pgSize := float64(2500000000)

	diskSize := resource.MustParse(diskSizeString)
	diskSizeInt64, _ := diskSize.AsInt64()
	diskSizeFloat := float64(diskSizeInt64)
	fmt.Printf("%f pgsize \n", pgSize)
	fmt.Printf("%d diskSize \n", diskSizeInt64)
	fmt.Printf("%f pct used \n", (pgSize/diskSizeFloat)*100.0)
	fmt.Printf("%d pct used \n", int64((pgSize/diskSizeFloat)*100.0))
}
