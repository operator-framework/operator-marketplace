# How To Push Operator Manifest(s) To App Registry
This document shows how one can push operator manifest(s) into `quay.io` or any other compatible app registry server.

*The example script(s) in this document refers to quay.io*

## Format of Operator Manifest
*As proof of concept, we are storing the operator manifest into one file. It could change in the future.*

The operator manifest file has the following structure -
```yaml
name: certified operators
publisher: redhat

data:
  customResourceDefinitions: |-
  clusterServiceVersions: |-
  packages: |-
```

## Pre-Requisite
*You need to have an account with quay.io. If you don't have one you can sign up for it.* 

### Obtain Access Token
All API requests made by executing HTTP verb (GET, POST, PUT or DELETE) against the API endpoint URL require an `Authorization header` with an access token. By logging in to `quay.io` you will receive an access token. You can get the token running the following bash script:

```bash
$ ./scripts/get-quay-token
```

You will be prompted to provide your `quay.io` account user name and password. Please ensure you have given the script the correct executable permissions by running:

```bash
$ chmod +x ./scripts/get-quay-token
```

Upon successful invocation, you will receive a JSON response with an access token, as shown below. Save the value of the `token` field from the JSON response. We will use this token to make calls to `quay.io` API. 
```json
{
    "token": "basic XWtgc2hlbTpsZWR6ZXCwbYlf"
}
```

## Push Operator Manifest

* Save the manifest into a file, for example `myapp.yaml`

You can now push the manifests to quay by using the following bash script:

```bash
$ ./scripts/push-to-quay
```

You will be prompted for the path to the file, and the namespace, repo, release and token related to you quay registry. 

For example, if you have saved the manifest into `myapp.yaml` and want to push it to `myoperators/myapp:1.0.0` then the variables will have the following values.

    file: myapp.yaml
    namespace: myoperators
    repository: myapp
    release: 1.0.0
    token: basic XWtgc2hlbTpsZWR6ZXCwbYlf

Please ensure you have given the script appropriate executable permissions by running:

```bash
$ chmod +x ./scripts/push-to-quay
```

You should now be able to visit `quay.io` and browse the operator manifest in your desired namespace.

## Troubleshooting

### New Version/Release of Operator Manifest

Each release of an operator manifest is considered immutable. If you try to push to an existing release you will get the following error from quay.
```json
{
    "error": {
        "code":"package-exists","details":{},"message":"package exists already"
    }
}
```

If you change your operator manifest, you should always bump up the version of the version/release before pushing the manifest.

### Delete Operator Manifest

*You will need `admin` privileges to delete repositories.*

For now, we have been deleting manifest(s) manually by going to `quay.io`. Go to the `Settings` tab of your desired repository and click the `Delete Application` button. 

### Creating New Manifests

By default, when you push a new manifest to quay.io, the repository will be private. Ensure that you make it public in order for it to be visible by the marketplace operator.

### Updating the OperatorSource

Currently, update is not supported. If you would like to update the OperatorSource CR, you must delete the previous OperatorSource resource, delete the marketplace pod and finally delete the package-server pod in the `openshift-operator-lifecycle-manager` namespace (this is to ensure that all cache and memory has been cleared - otherwise you may see stale operator data in the UI).