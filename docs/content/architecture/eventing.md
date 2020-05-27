---
title: "Lifecycle Events"
date:
draft: false
weight: 500
---

## Operator Eventing

The Operator creates events from the various life-cycle
events going on within the Operator logic and driven
by pgo users as they interact with the Operator and as
Postgres clusters come and go or get updated.

## Event Watching

There is a pgo CLI command:

    pgo watch alltopic

This command connects to the event stream and listens
on a topic for event real-time.  The command will not
complete until the pgo user enters ctrl-C.

This command will connect to localhost:14150 (default) to reach the
event stream.  If you have the correct priviledges
to connect to the Operator pod, you can port forward
as follows to form a connection to the event stream:

    kubectl port-forward svc/postgres-operator 14150:4150 -n pgo

## Event Topics

The following topics exist that hold the various Operator
generated events:

    alltopic
    clustertopic
    backuptopic
    loadtopic
    postgresusertopic
    policytopic
    pgbouncertopic
    pgotopic
    pgousertopic

## Event Types

The various event types are found in the source code at
https://github.com/CrunchyData/postgres-operator/blob/master/pkg/events/eventtype.go


## Event Deployment

The Operator events are published and subscribed via the NSQ
project software (https://nsq.io/).  NSQ is found in the pgo-event
container which is part of the postgres-operator deployment.

You can see the pgo-event logs by issuing the elog bash function
found in the examples/envs.sh script.

NSQ looks for events currently at port 4150.  The Operator sends
events to the NSQ address as defined in the EVENT_ADDR environment
variable.

If you want to disable eventing when installing with Bash, set the following
environment variable in the Operator Deployment:
    "name": "DISABLE_EVENTING"
    "value": "true"

To disable eventing when installing with Ansible, add the following to
your inventory file:
    pgo_disable_eventing='true'
