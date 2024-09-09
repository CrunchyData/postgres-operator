// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

// The protocol used by PostgreSQL is registered with the Internet Assigned
// Numbers Authority (IANA).
// - https://www.iana.org/assignments/service-names-port-numbers
const (
	// IANAPortNumber is the port assigned to PostgreSQL at the IANA.
	IANAPortNumber = 5432

	// IANAServiceName is the name of the PostgreSQL protocol at the IANA.
	IANAServiceName = "postgresql"
)
