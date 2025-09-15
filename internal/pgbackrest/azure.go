// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"os"
)

// setAzureCredentials populates the provided map with references to the Azure
// credentials for use with pgBackRest
func setAzureCredentials(configMapKeyData map[string]string, clusterName, namespace string) {
	configMapKeyData["AZURE_CONTAINER"] = "$(PGBACKREST_REPO1_AZURE_CONTAINER)"
	configMapKeyData["AZURE_ACCOUNT"] = "$(PGBACKREST_REPO1_AZURE_ACCOUNT)"

	// When using Azure AD Pod Identity, we don't need to set the AZURE_KEY
	if os.Getenv("PGBACKREST_AZURE_USE_AAD") != "true" {
		configMapKeyData["AZURE_KEY"] = "$(PGBACKREST_REPO1_AZURE_KEY)"
	}
}
