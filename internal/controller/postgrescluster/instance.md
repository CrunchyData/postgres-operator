<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

## Shutdown and Startup Logic Detail

The Shutdown/Startup process used by the `postgresclusters` is somewhat nuanced
and may be a bit difficult to understand by just reviewing the code and 
associated comments. To help clarify, here is a brief explanation of the logic
being used.

### Startup Instance Value

The first code block to consider is found in the `observeInstances` function:

```
// Go through the observed instances and check if a primary has been determined.
// If the cluster is being shutdown and this instance is the primary, store
// the instance name as the startup instance. If the primary can be determined
// from the instance and the cluster is not being shutdown, clear any stored
// startup instance values.
for _, instance := range observed.forCluster {
	if primary, known := instance.IsPrimary(); primary && known {
		if cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown {
			cluster.Status.StartupInstance = instance.Name
		} else {
			cluster.Status.StartupInstance = ""
		}
	}
}
```

This sets the `StartupInstance` status value, which stores the primary/leader
instance name during a `postgrescluster` shutdown. When the cluster is restarted,
this value is cleared so that it only appears in the `postgrescluster` status
while the  cluster is shutdown.

### Other Key Values

Besides the stored `StartupInstance` name, the two other values used to set
the replica count are the `Shutdown` value from the `postgrescluster` spec
and the current pod count per cluster. With these values, the solution used 
in the code can be represented by:

`Replicas = (SSI match & ~Single Pod) | (SSI match & ~Shutdown) | (SSI blank)`

where 
`Replicas` is the number of replica pods to be created, either zero or one

`SSI` refers to the status value for `StartupInstance`, either matching the
instance name or set to blank ("")

 `Single Pod` refers to whether the cluster has a single pod left running, i.e.
 the primary/leader

 `Shutdown` is whether the cluster is configured to be shutdown

### Logic Map

With this, the grid below shows the expected replica count value, depending on
the values. Below, the letters represent the following:

M = StartupInstance matches the instance name

E = StartupInstance is empty

S = cluster is configured to Shutdown

P = a single pod exists

When the letter is capitalized, that indicates the statement is `true`
if lowercase, the statement is `false`.

|    | em | eM | EM | Em |
|----|---|----|----|----|
| sp | 0 | 1 | 1 | 1 |
| sP | 0 | 1 | 1 | 1 |
| SP | 0 | 0 | 1 | 1 |
| Sp | 0 | 1 | 1 | 1 |


### Implementation

Following this, we have the `if/else` block as found in the 
`generateInstanceStatefulSetIntent` function:

```
if cluster.Status.StartupInstance == "" {
		// there is no designated startup instance; all instances should run.
		sts.Spec.Replicas = initialize.Int32(1)
	} else if cluster.Status.StartupInstance != sts.Name {
		// there is a startup instance defined, but not this instance; do not run.
		sts.Spec.Replicas = initialize.Int32(0)
	} else if cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown &&
		numInstancePods <= 1 {
		// this is the last instance of the shutdown sequence; do not run.
		sts.Spec.Replicas = initialize.Int32(0)
	} else {
		// this is the designated instance, but
		// - others are still running during shutdown, or
		// - it is time to startup.
		sts.Spec.Replicas = initialize.Int32(1)
	}
```

Which allows the correct replica count to be set during both startup and
shutdown. During a shutdown, all pods other than the primary will begin
termination first, followed by the primary. On startup, a reversed process
will be followed. In cases where the `StartupInstance` value is not set, all
pods will be allowed to start at the same time.
