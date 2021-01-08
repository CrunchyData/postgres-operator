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

package patroni

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

type API interface {
	// ReplaceConfiguration replaces Patroni's entire dynamic configuration.
	ReplaceConfiguration(ctx context.Context, configuration map[string]interface{}) error
}

// Executor implements API by calling "patronictl".
type Executor func(
	ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// Executor implements API.
var _ API = Executor(nil)

// ReplaceConfiguration replaces Patroni's entire dynamic configuration by
// calling "patronictl".
func (exec Executor) ReplaceConfiguration(
	ctx context.Context, configuration map[string]interface{},
) error {
	var stdin, stdout, stderr bytes.Buffer

	err := json.NewEncoder(&stdin).Encode(configuration)
	if err == nil {
		err = exec(ctx, &stdin, &stdout, &stderr,
			"patronictl", "edit-config", "--replace=-", "--force")

		log := logging.FromContext(ctx)
		log.V(1).Info("replaced configuration",
			"stdout", stdout.String(),
			"stderr", stderr.String(),
		)
	}

	return err
}
