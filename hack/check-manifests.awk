# Copyright 2025 - 2026 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

## TODO: Exit successfully only when there are no errors.
#/^ERROR:/ { rc = 1 }
#END { exit rc }

# Shorten these frequent messages about validation rules.
/The maximum allowable value is 10000000[.]/ {
	sub(/ The maximum allowable value is 10000000./, "")
	sub(/ allowed budget/, "&, 10M")
}

# These are informational, but "MustNot" sounds like something is wrong.
/^info: "MustNotExceedCostBudget"/ {
	sub(/"MustNotExceedCostBudget"/, "\"CostBudget\"")
}

# Color errors and warnings when attached to a terminal.
ENVIRON["MAKE_TERMOUT"] != "" {
	sub(/^ERROR:/, "\033[0;31m&\033[0m")
	sub(/^Warning:/, "\033[1;33m&\033[0m")
}

{ print }
