#!/bin/bash

MANIFESTS="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )/../manifests"

SOURCE_MANIFEST="${MANIFESTS}/09_operator.yaml"
DESTINATION_MANIFEST="${MANIFESTS}/09_operator-ibm-cloud-managed.yaml"
cp "${SOURCE_MANIFEST}" "${DESTINATION_MANIFEST}"

YQ="go run ./vendor/github.com/mikefarah/yq/v3/"
${YQ} d -d'*' --inplace "${DESTINATION_MANIFEST}" 'metadata.annotations'
${YQ} w -d'*' --inplace --style=double "${DESTINATION_MANIFEST}" 'metadata.annotations['config.openshift.io/inject-proxy']' "marketplace-operator"
${YQ} w -d'*' --inplace --style=double "${DESTINATION_MANIFEST}" 'metadata.annotations['include.release.openshift.io/ibm-cloud-managed']' true
${YQ} d -d'*' --inplace "${DESTINATION_MANIFEST}" 'spec.template.spec.nodeSelector."node-role.kubernetes.io/master"'
