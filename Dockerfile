FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.25-openshift-4.22 AS builder
WORKDIR /go/src/github.com/operator-framework/operator-marketplace
COPY . .
RUN make osbs-build

FROM registry.ci.openshift.org/ocp/4.22:base-rhel9-minimal
RUN useradd marketplace-operator
USER marketplace-operator
COPY --from=builder /go/src/github.com/operator-framework/operator-marketplace/build/_output/bin/marketplace-operator /usr/bin
COPY defaults /defaults
COPY manifests /manifests
COPY vendor/github.com/openshift/api/config/v1/zz_generated.crd-manifests/*operatorhubs.crd.yaml /manifests

LABEL io.k8s.display-name="OpenShift Marketplace Operator" \
      io.k8s.description="This is a component of OpenShift Container Platform and manages the OpenShift Marketplace." \
      io.openshift.tags="openshift,marketplace" \
      io.openshift.release.operator=true \
      maintainer="AOS Marketplace <aos-marketplace@redhat.com>"

# entrypoint specified in operator.yaml as `marketplace-operator`
CMD ["/usr/bin/marketplace-operator"]
