#!/bin/bash

# Apply the CRD from GitHub first
kubectl apply -f https://raw.githubusercontent.com/openshift/api/600991d550ac9ee3afbfe994cf0889bf9805a3f5/config/v1/0000_03_marketplace-operator_01_operatorhub.crd.yaml

# Apply the CatalogSource CRD from GitHub
kubectl apply -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/b31bd89a23d597ca440024458d0099a273642f8c/deploy/upstream/quickstart/crds.yaml

# Apply the first set of manifest files
kubectl apply -f manifests/0000_03_marketplace-operator_02_operatorhub.cr.yaml
kubectl apply -f manifests/01_namespace.yaml
kubectl apply -f manifests/04_service_account.yaml
kubectl apply -f manifests/05_role.yaml
kubectl apply -f manifests/06_role_binding.yaml
kubectl apply -f manifests/07_configmap.yaml
kubectl apply -f manifests/08_service.yaml

# Save the localhost image to a tar file and load it into the kind cluster
#echo "Saving the image as a tarball and loading it into kind cluster..."
#podman save -o marketplace-operator.tar localhost/marketplace-operator:latest
#kind load image-archive marketplace-operator.tar

# Apply the 09_operator.yaml with modifications on the fly (using localhost image)
yq eval '
  del(.spec.template.spec.nodeSelector) |
  del(.spec.template.spec.tolerations[] | select(.key == "node-role.kubernetes.io/master")) |
  del(.spec.template.spec.containers[].volumeMounts[] | select(.name == "marketplace-operator-metrics")) |
  del(.spec.template.spec.volumes[] | select(.name == "marketplace-operator-metrics")) |
  del(.spec.template.spec.containers[].args[] | select(. == "-tls-cert")) |
  del(.spec.template.spec.containers[].args[] | select(. == "/var/run/secrets/serving-cert/tls.crt")) |
  del(.spec.template.spec.containers[].args[] | select(. == "-tls-key")) |
  del(.spec.template.spec.containers[].args[] | select(. == "/var/run/secrets/serving-cert/tls.key")) |
  del(.spec.template.spec.securityContext.runAsNonRoot) |
  .spec.template.spec.containers[0].image = "localhost/marketplace-operator:latest"
' manifests/09_operator.yaml | kubectl apply -f -
