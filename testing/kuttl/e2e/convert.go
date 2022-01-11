/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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
	"os"
	"path/filepath"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	yamljsonparser "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	// for _, filename := range os.Args[1:] {
	filename := os.Args[1]
	fmt.Printf("converting %v …\n", filename)

	if err := Convert(filename); err != nil {
		fmt.Fprintf(os.Stderr, "unable to convert %v: %+v\n", filename, err)
		os.Exit(1)
	}

	fmt.Println("\tok")
	// }
}

type FileParse struct {
	Name        string `json:"name,omitempty"`
	Source      string `json:"source,omitempty"`
	Alterations string `json:"alterations,omitempty"`
}

// Convert prepares the KUTTL test files by
//	prepping a test dir,
//	reading the outline YAML file, and
//	creating the KUTTL test files registered in the outline YAML
//		with the JSON-patches applied to the base files
func Convert(filename string) error {

	dir := strings.TrimSuffix(filename, filepath.Ext(filename))
	writer := Writer{Directory: dir}
	if err := os.RemoveAll(dir); err == nil {
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			return err
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := yaml.NewYAMLToJSONDecoder(file)

	var documents int
	for {
		testFile, name, err := parseAndPatchTestFile(decoder)
		if err == nil {
			fmt.Printf("err %v …\n", err)
			err = writer.Write(documents, name, testFile)
			if err != nil {
				break
			}
			documents++
		} else {
			break
		}
	}

	return err
}

func parseAndPatchTestFile(decoder *yaml.YAMLToJSONDecoder) (string, string, error) {
	var fp FileParse
	err := decoder.Decode(&fp)

	if err == nil {
		var original, jsonified, modified, yamlified []byte
		var patch jsonpatch.Patch
		var expanded string

		original, err := os.ReadFile("testing/kuttl/e2e/" + fp.Source)
		if err == nil {
			jsonified, err = yamljsonparser.YAMLToJSON(original)
		}
		if err == nil {
			patch, err = jsonpatch.DecodePatch([]byte(fp.Alterations))
		}
		if err == nil {
			modified, err = patch.Apply(jsonified)
		}
		if err == nil {
			yamlified, err = yamljsonparser.JSONToYAML(modified)
		}
		if err == nil {
			expanded = os.ExpandEnv(string(yamlified))
		}
		return expanded, fp.Name, err
	}
	return "", "", err
}

type Writer struct {
	Directory string
}

func (w Writer) Write(index int, name, fileContent string) error {
	filename := fmt.Sprintf("%02d-%s", index, name)
	return os.WriteFile(filepath.Join(w.Directory, filename), []byte(fileContent), 0o644) //nolint:gosec
}
