# operator

=== Building

....
go get -u github.com/FiloSottile/gvt
gvt fetch -tag v2.0.0 k8s.io/client-go
gvt restore
make operatorserver
....

On minikube, to use the local docker image, in /etc/sysconfig/docker:
....
< DOCKER_CERT_PATH=/etc/docker
---
> if [ -z "${DOCKER_CERT_PATH}" ]; then
>   DOCKER_CERT_PATH=/etc/docker
> fi
....

then,
....
eval $(minikube docker-env)
....


