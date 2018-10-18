FROM openshift/origin-base

RUN useradd marketplace-operator
USER marketplace-operator

ADD tmp/_output/bin/marketplace-operator /usr/local/bin/marketplace-operator
