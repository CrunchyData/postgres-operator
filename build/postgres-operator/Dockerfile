FROM registry.access.redhat.com/ubi8/ubi-minimal

COPY licenses /licenses

COPY bin/postgres-operator /usr/local/bin

RUN mkdir -p /opt/crunchy/conf

COPY hack/tools/queries /opt/crunchy/conf

RUN chgrp -R 0 /opt/crunchy/conf && chmod -R g=u opt/crunchy/conf

USER 2

CMD ["postgres-operator"]
