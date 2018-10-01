
FROM openshift/origin-release:golang-1.10 as builder
WORKDIR /go/src/github.com/operator-framework/operator-marketplace
COPY . .
RUN make osbs-build

FROM alpine:3.6
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN useradd marketplace-operator
USER marketplace-operator
COPY --from=builder /go/src/github.com/operator-framework/operator-marketplace/tmp/_output/bin/marketplace-operator /usr/bin

LABEL io.k8s.display-name="OpenShift Marketplace Operator" \
      io.k8s.description="This is a component of OpenShift Container Platform and manages the OpenShift Marketplace." \
      io.openshift.tags="openshift,marketplace" \
      maintainer="AOS Marketplace <aos-marketplace@redhat.com>"

# entrypoint specified in operator.yaml as `marketplace-operator`
CMD ["/usr/bin/marketplace-operator"]
