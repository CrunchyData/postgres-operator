package api

/*
 Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// StatusCheck ...
func StatusCheck(resp *http.Response) error {
	log.Debugf("http status code is %d", resp.StatusCode)
	if resp.StatusCode == 401 {
		return fmt.Errorf("Authentication Failed: %d\n", resp.StatusCode)
	} else if resp.StatusCode == 405 {
		return fmt.Errorf("Method %s for URL %s is not allowed in the current Operator "+
			"install: %d", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode)
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("Invalid Status Code: %d\n", resp.StatusCode)
	}
	return nil
}
