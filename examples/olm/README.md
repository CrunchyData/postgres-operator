# Postgres Operator 4.0 OLM Installation on Openshift 4.0

Follow the following instructions to install the postgres operator verison 4.0 on Openshift 4.0 using
OLM.

  - mkdir -p $HOME/odev/src/github.com/crunchydata
  - cd $HOME/odev/src/github.com/crunchydata
  - git clone https://github.com/crunchydata/postgres-operator
  - sudo yum install epel-release
  - sudo yum install golang
  - cd postgres-operator
  - git checkout develop
  - source examples/olm/envs.sh
  - go get github.com/blang/expenv
  - make cleannamespaces
  - make setupnamespaces
  - make installrbac
  - cd examples/olm/bundle
  - oc create -f pgo.configmap.yaml
  - oc create -f pgo.catalogsource.yaml
  - oc create -f pgo.operatorgroup.yaml
  - oc create -f pgo.subscription.yaml
 

# postgres operator 4.0 CRDs

  - pgbackups.crunchydata.com
  - pgclusters.crunchydata.com
  - pgpolicies.crunchydata.com
  - pgreplicas.crunchydata.com
  - pgtasks.crunchydata.com
 
# Namespaces
 
The operator install assumes the following namespaces:
  - pgo (this namespace holds the operator itself)
  - pgouser1, pgouser2 ( these are the target namespaces Postgres clusters will be deployed into)
 
# RBAC  - Service Accounts
  - postgres-operator SA is deployed into the pgo namespace
  - pgo-backrest SA is deployed into the pgouser1,pgouser2 namespaces

# RBAC - Roles
  - pgo-backrest-role is deployed into the pgouser1, pgouser2 namespaces
  - pgo-role is deployed into the pgouser1, pgouser2 namespaces

# RBAC - Roles
  - pgo-backrest-role-binding created in pgouser1, pgouser2 namespaces
  - pgo-role-binding created in pgouser1,pgouser2 namespaces

# RBAC - ClusterRoles
  - pgopclusterrole created in pgo namespace
  - pgopclusterrolecrd  created in pgo namespace
  - pgopclusterrolesecret created in pgo namespace

# RBAC - ClusterRoleBindings
  - pgopclusterbinding-pgo created in pgo namespace
  - pgopclusterbindingcrd-pgo  created in pgo namespace
  - pgopclusterbindingsecret-pgo  created in pgo namespace

# Teardown
  - oc project pgo
  - oc delete configmap  dev-operators
  - oc delete subscription --all
  - oc delete catalogsource dev-operators
  - oc delete operatorgroup dev-operators
