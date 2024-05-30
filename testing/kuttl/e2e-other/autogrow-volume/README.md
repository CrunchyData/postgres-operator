### AutoGrow Volume

* 00: Assert the storage class allows volume expansion
* 01: Create and verify PostgresCluster and PVC
* 02: Add data to trigger growth and verify Job completes
* 03: Verify annotation on the instance Pod
* 04: Verify the PVC request has been set and the PVC has grown
* 05: Verify the expansion request Event has been created
      Note: This Event should be created between steps 03 and 04 but is checked at the end for timing purposes.
