FROM registry.access.redhat.com/ubi8/ubi-micro

COPY licenses /licenses

COPY bin/postgres-operator /usr/local/bin

USER 2

CMD ["postgres-operator"]
