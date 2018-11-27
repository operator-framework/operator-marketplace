FROM openshift/origin-base

RUN useradd marketplace-operator
USER marketplace-operator

ADD build/_output/bin/operator-marketplace /usr/local/bin/operator-marketplace
