package api

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

import (
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// StatusCheck ...
func StatusCheck(resp *http.Response) error {
	log.Debugf("http status code is %d", resp.StatusCode)
	if resp.StatusCode == 401 {
		return errors.New(fmt.Sprintf("Authentication Failed: %d\n", resp.StatusCode))
	} else if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Invalid Status Code: %d\n", resp.StatusCode))
	}
	return nil
}
