---
title: TODO - Content that needs a home
draft: false
weight: 200
---

This content needs to find a permanent home.

## Add pgBackRest Backup Schedules

Scheduled pgBackRest `full`, `differential` and `incremental` backups can be added for each defined pgBackRest
repo. This is done by adding, under the `repos` section, a `schedules` section with the designated CronJob
schedule defined for each backup type desired. For example, for `repo1`, we defined the following:
```
  archive:
    pgbackrest:
      repoHost:
        dedicated: {}
        image: gcr.io/crunchy-dev-test/crunchy-pgbackrest:centos8-12.6-multi.dev2
      repos:
      - name: repo1
        schedules:
          full: "* */1 * * *"
          differential: "*/10 * * * *"
          incremental: "*/5 * * * *"
```
For any type not listed, no CronJob will be created. For more information on CronJobs and the necessary scheduling
syntax, please see https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax.
