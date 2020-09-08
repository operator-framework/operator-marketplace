# How To Authenticate Private registry Repositories

If you have an registry repository that is backed by authentication, you can specify an authentication token in a Secret. To do this, create a Secret in the same namespace as your Catalog Source:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: marketplacesecret
  namespace: openshift-marketplace
type: Opaque
stringData:
    token: "basic yourtokenhere=="
```

Then, to associate that secret with a `CatalogSource`, simply add a reference to the secret in the spec:

```yaml
apiVersion: "operators.coreos.com/v1alpha1"
kind: "CatalogSource"
metadata:
  name: "certified-operators"
  namespace: "openshift-marketplace"
spec:
  sourceType: grpc
  image: registry.redhat.io/redhat/certified-operator-index:v4.6
  displayName: "Certified Operators"
  publisher: "Red Hat"
  priority: -200
  updateStrategy:
    registryPoll:
      interval: 10m
  secrets:
    - marketplacesecret
```

That's it! While accessing catalog entries, each secret in the list will be tried sequentially. The secret list should be kept short to avoid too many secrets being tried for each time the catalog gets accessed.