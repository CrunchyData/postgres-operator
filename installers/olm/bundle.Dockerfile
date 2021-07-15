# Used to build the bundle image. This file is ignored by the community operator
# registries which work with bundle directories instead.
# https://operator-framework.github.io/community-operators/packaging-operator/

FROM scratch AS builder

COPY manifests/ /build/manifests/
COPY metadata/ /build/metadata/
COPY tests/ /build/tests


FROM scratch

# ANNOTATIONS is replaced with bundle.annotations.yaml
LABEL \
	${ANNOTATIONS}

COPY --from=builder /build/ /
