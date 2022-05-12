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

Point your browser at `http://localhost:5050` and you will be prompted to log in.
Use your database username with `@pgo` appended and your database password.
In our case, the pgAdmin username is `hippo@pgo` and the password is found in the
user secret, `hippo-pguser-hippo`:

```
PG_CLUSTER_USER_SECRET_NAME=hippo-pguser-hippo

PGPASSWORD=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.password | base64decode}}')
PGUSER=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.user | base64decode}}')
```

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

You can also mount files to `/etc/pgadmin/conf.d` inside the pgAdmin container using
[projected volumes](https://kubernetes.io/docs/concepts/storage/projected-volumes/).
The following mounts `useful.txt` of Secret `mysecret` to `/etc/pgadmin/conf.d/useful.txt`:

```yaml
  userInterface:
    pgAdmin:
      config:
        files:
        - secret:
            name: mysecret
            items:
            - key: useful.txt
        - configMap:
            name: myconfigmap
            optional: false
```

### Kerberos Configuration

You can configure pgAdmin to [authenticate its users using Kerberos](https://www.pgadmin.org/docs/pgadmin4/latest/kerberos.html)
SPNEGO. In addition to setting `AUTHENTICATION_SOURCES` and `KRB_APP_HOST_NAME`, you need to
enable `KERBEROS_AUTO_CREATE_USER` and mount a `krb5.conf` and a keytab file:

```yaml
  userInterface:
    pgAdmin:
      config:
        settings:
          AUTHENTICATION_SOURCES: ['kerberos']
          KERBEROS_AUTO_CREATE_USER: True
          KRB_APP_HOST_NAME: my.service.principal.name.local # without HTTP class
          KRB_KTNAME: /etc/pgadmin/conf.d/krb5.keytab
        files:
        - secret:
            name: mysecret
            items:
            - key: krb5.conf
            - key: krb5.keytab
```

### LDAP Configuration

You can configure pgAdmin to [authenticate its users using LDAP](https://www.pgadmin.org/docs/pgadmin4/latest/ldap.html)
passwords. In addition to setting `AUTHENTICATION_SOURCES` and `LDAP_SERVER_URI`, you need to
enable `LDAP_AUTO_CREATE_USER`:

```yaml
  userInterface:
    pgAdmin:
      config:
        settings:
          AUTHENTICATION_SOURCES: ['ldap']
          LDAP_AUTO_CREATE_USER: True
          LDAP_SERVER_URI: ldaps://my.ds.example.com
```

When using a dedicated user to bind, you can store the `LDAP_BIND_PASSWORD` setting in a Secret and
reference it through the [`ldapBindPassword`]({{< relref "/references/crd#postgresclusterspecuserinterfacepgadminconfigldapbindpassword" >}})
field:

```yaml
  userInterface:
    pgAdmin:
      config:
        ldapBindPassword:
          name: ldappass
          key: mypw
```

## Deleting pgAdmin 4

You can remove the pgAdmin 4 deployment by removing the `userInterface` field from the spec.
