To deploy monitoring,

1. verify the namespace is correct in kustomization.yaml 
2. If you are deploying in openshift, edit deploy*.yaml and comment out fsGroup line under securityContext
3. kubectl apply -k .
