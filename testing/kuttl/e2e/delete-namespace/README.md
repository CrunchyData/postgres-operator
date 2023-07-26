### Delete namespace test

* Create a namespace
* Start a regular cluster in that namespace
* Delete the namespace
* Check that nothing remains.

Note: KUTTL provides a `$NAMESPACE` var that can be used in scripts/commands,
but which cannot be used in object definition yamls (like `01--cluster.yaml`).
Therefore, we use a given, non-random namespace that is defined in the makefile
and generated with `generate-kuttl`.
