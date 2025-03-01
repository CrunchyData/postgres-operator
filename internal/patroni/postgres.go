// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	"encoding/json"
	"fmt"

	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// PostgresHBAs returns the HBA rules in spec, if any.
func PostgresHBAs(spec *v1beta1.PatroniSpec) []string {
	var result []string

	if spec != nil {
		// DynamicConfiguration lacks an OpenAPI schema, so it may contain any type
		// at any depth. Navigate the object and skip HBA values that aren't string.
		//
		// Patroni expects a list of strings:
		// https://github.com/patroni/patroni/blob/v4.0.0/patroni/validator.py#L1170
		//
		if root := spec.DynamicConfiguration; root != nil {
			if postgresql, ok := root["postgresql"].(map[string]any); ok {
				if section, ok := postgresql["pg_hba"].([]any); ok {
					for i := range section {
						if value, ok := section[i].(string); ok {
							result = append(result, value)
						}
					}
				}
			}
		}
	}

	return result
}

// PostgresParameters returns the Postgres parameters in spec, if any.
func PostgresParameters(spec *v1beta1.PatroniSpec) *postgres.ParameterSet {
	result := postgres.NewParameterSet()

	if spec != nil {
		// DynamicConfiguration lacks an OpenAPI schema, so it may contain any type
		// at any depth. Navigate the object and convert parameter values to string.
		//
		// Patroni accepts booleans, integers, and strings but also parses
		// string values into the types it expects:
		// https://github.com/patroni/patroni/blob/v4.0.0/patroni/postgresql/validator.py
		//
		// Patroni passes JSON arrays and objects through Python str() which looks
		// similar to YAML in simple cases:
		// https://github.com/patroni/patroni/blob/v4.0.0/patroni/postgresql/config.py#L254-L259
		//
		//	>>> str(list((1, 2.3, True, "asdf")))
		//	"[1, 2.3, True, 'asdf']"
		//
		//	>>> str(dict(a = 1, b = True))
		//	"{'a': 1, 'b': True}"
		//
		if root := spec.DynamicConfiguration; root != nil {
			if postgresql, ok := root["postgresql"].(map[string]any); ok {
				if section, ok := postgresql["parameters"].(map[string]any); ok {
					for k, v := range section {
						switch v.(type) {
						case []any, map[string]any:
							if b, err := json.Marshal(v); err == nil {
								result.Add(k, string(b))
							}
						default:
							result.Add(k, fmt.Sprint(v))
						}
					}
				}
			}
		}
	}

	return result
}
