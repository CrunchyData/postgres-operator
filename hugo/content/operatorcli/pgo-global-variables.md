## pgo Global Environment Variables

*pgo* will pick up these settings if set in your environment:

| Name | Description | NOTES |
|------|-------------|-------|
|PGOUSERNAME |The username (role) used for auth on the operator apiserver. | Requires that PGOUSERPASS be set. |
|PGOUSERPASS |The password for used for auth on the operator apiserver. | Requires that PGOUSERNAME be set. |
|PGOUSER |The path the the pgorole file. | Will be ignored if either PGOUSERNAME or PGOUSERPASS are set. |
