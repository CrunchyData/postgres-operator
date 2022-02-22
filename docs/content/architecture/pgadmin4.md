---
title: "pgAdmin 4"
date:
draft: false
weight: 900
---

![pgAdmin 4 Query](/images/pgadmin4-query.png)

[pgAdmin 4](https://www.pgadmin.org/) is a popular graphical user interface that
makes it easy to work with PostgreSQL databases from a web-based client. With
its ability to manage and orchestrate changes for PostgreSQL users, the PostgreSQL
Operator is a natural partner to keep a pgAdmin 4 environment synchronized with
a PostgreSQL environment.

The PostgreSQL Operator lets you deploy a pgAdmin 4 environment alongside a
PostgreSQL cluster and keeps users' database credentials synchronized. You can
simply log into pgAdmin 4 with your PostgreSQL username and password and
immediately have access to your databases.

## Deploying pgAdmin 4

If you've done the [quickstart]({{< relref "quickstart/_index.md" >}}), add the
following fields to the spec and reapply; if you don't have any Postgres clusters
running, add the fields to a spec, and apply.

```yaml
  userInterface:
    pgAdmin:
      image: {{< param imageCrunchyPGAdmin >}}
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
```

This creates a pgAdmin 4 deployment unique to this PostgreSQL cluster and synchronizes
the PostgreSQL user information. To access pgAdmin 4, you can set up a port-forward
to the Service, which follows the pattern `<clusterName>-pgadmin`, to port `5050`:

```
kubectl port-forward svc/hippo-pgadmin 5050:5050
```

Point your browser at `http://localhost:5050` and use your database username and
password to log in. Access your username and password by getting the values from
your user secret. In our case, the secret will be `hippo-pguser-hippo`:

```
PG_CLUSTER_USER_SECRET_NAME=hippo-pguser-hippo

PGPASSWORD=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.password | base64decode}}')
PGUSER=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.user | base64decode}}')
```

Though the prompt says "email address", using your PostgreSQL username will work.

![pgAdmin 4 Login Page](/images/pgadmin4-login.png)

{{% notice tip %}}
If your password does not appear to work, you can retry setting up the user by
rotating the user password. Do this by deleting the `password` data field from
the user secret (e.g. `hippo-pguser-hippo`).

Optionally, you can also set a [custom password]({{< relref "architecture/user-management.md" >}}).
{{% /notice %}}

## User Synchronization

The operator will synchronize users defined in the spec (e.g., in `spec.users`) with the pgAdmin 4
deployment. Any user created in the database without being defined in the spec will not be
synchronized.

## Custom Configuration

You can adjust some pgAdmin settings through the
[`userInterface.pgAdmin.config`]({{< relref "/references/crd#postgresclusterspecuserinterfacepgadminconfig" >}})
field. For example, set `SHOW_GRAVATAR_IMAGE` to `False` to disable automatic profile pictures:

```yaml
  userInterface:
    pgAdmin:
      config:
        settings:
          SHOW_GRAVATAR_IMAGE: False
```

You can also add a Secret containing the pgAdmin `LDAP_BIND_PASSWORD` through the
[`userInterface.pgAdmin.config.ldapBindPassword`]
({{< relref "/references/crd#postgresclusterspecuserinterfacepgadminconfigldapbindpassword" >}})
field. This is one of the configuration settings needed to enable LDAP authentication
for pgAdmin and is handled separately from the other pgAdmin settings to allow for
proper storage of the sensitive value in a Secret rather than a ConfigMap.

```yaml
  userInterface:
    pgAdmin:
      config:
        ldapBindPassword:
          name: ldappass
          key: mypw
```

Lastly, you can also use Secrets and ConfigMaps to mount required files to your
pgAdmin container through the
[`userInterface.pgAdmin.config.files`]
({{< relref "/references/crd#postgresclusterspecuserinterfacepgadminconfigfilesindex" >}})
field. The contents of the Secrets and ConfigMaps defined here are mounted at
`/etc/pgadmin/conf.d` and can be referenced from various pgAdmin configuration
settings as needed.

```yaml
  userInterface:
    pgAdmin:
      config:
        files:
          - secret:
              name: mysecret
          - configMap:
              name: myconfigmap
              optional: false
```

## Deleting pgAdmin 4

You can remove the pgAdmin 4 deployment by removing the `userInterface` field from the spec.
