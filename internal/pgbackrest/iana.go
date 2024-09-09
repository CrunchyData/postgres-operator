// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

// The protocol used by pgBackRest is registered with the Internet Assigned
// Numbers Authority (IANA).
// - https://www.iana.org/assignments/service-names-port-numbers
const (
	// IANAPortNumber is the port assigned to pgBackRest at the IANA.
	IANAPortNumber = 8432

	// IANAServiceName is the name of the pgBackRest protocol at the IANA.
	IANAServiceName = "pgbackrest"
)
