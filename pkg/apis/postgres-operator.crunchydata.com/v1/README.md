# Multiversion CRD

The purpose of this README is to discuss the current/future experience of transitioning between
versions of the postgrescluster CRD, as well as to identify future work.

## Version sorting and how that affects retrieval

[Version sorting in Kubernetes](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority) means that v1 takes
precedence over v1beta1. Version sorting disregards storage version.

So if you run `kubectl explain postgrescluster.spec.userInterface` you will get the v1 explainer.
In order to get the explainer in a particular version form, you need to add the `--api-version` flag:

```bash
kubectl explain postgrescluster.spec.userInterface --api-version=postgres-operator.crunchydata.com/v1
```

For `kubectl get`, the way to specify api version is in the resource name. That is, rather than
`kubectl get postgrescluster`, you could put

```bash
kubectl get postgrescluster.v1beta1.postgres-operator.crunchydata.com hippo -o yaml
```

That will return the `v1beta1` representation of that cluster.

**Future work**: The CLI tool calls kubectl, so we may need to expose/add a flag to allow people to specify
versions; we may also need to update some of the create and other commands to allow multiple versions (if desired).

### K9s and other GUIs

I'm not sure what other tools people use, but I know k9s is pretty popular. Unfortunately,
I cannot find a way to specify the form a K8s object is retrieved in. See [here](https://github.com/derailed/k9s/issues/838).

## Transitioning from v1beta1 to v1

If you have a v1beta1 cluster and want to save it as v1, you can change the `apiVersion` field:

Change 

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
```

to

```yaml
apiVersion: postgres-operator.crunchydata.com/v1
```

And if the cluster is acceptable as a v1 object, it will be saved.

It may return a warning if some new XValidation rule is being tested. For instance, since we've added a rule
that the `spec.userInterface` field should be null in v1, if you have a postgrescluster with that field
in a v1beta1 but _do not_ change that field, then you can save your cluster as a v1 version even though it will
return a warning that that field should be null.

(This is a result of using validation ratcheting, which should be enabled in K8s 1.30+ / OCP 4.17+.)

If you want to test whether a save or adjustment will be successful, you can run a dry-run first. That is,
add `--dry-run=server` to your create/apply command. This will check against the object as it currently exists
for the server. 

If you get blocked or if you get a warning and want to eliminate that warning, the way to do that is to update
the spec or make changes that will enable that spec to be valid. Hopefully the error messages from the K8s
API will help determine the change that are required.

That is, if you have a `spec.userInterface`, and the error informs you that this field is no longer available in v1,
you may need to check our documentation on the preferred way to deploy a pgAdmin4 deployment.

(We may in the future want to actually provide steps for all of the fields that we are changing,
e.g., a migration guide.)

