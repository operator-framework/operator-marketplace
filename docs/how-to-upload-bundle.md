# How To Push an Operator Bundle To an App Registry

This document shows how one can push an operator bundle onto `quay.io` or any other compatible app registry server. All commands in this document should be run in the root of this repository.

## What is an Operator Bundle?

An operator bundle is a collection of data required to make an operator offering available to a cluster. Each operator bundle represents one operator offering. Multiple operator offerings can be pushed to an app registry namespace.

An operator bundle is stored in a `yaml` file with the following structure:

```yaml
data:
  customResourceDefinitions: |-
  clusterServiceVersions: |-
  packages: |-
```

An operator's CSV must contain the annotations mentioned [here](https://github.com/operator-framework/operator-marketplace/blob/master/docs/marketplace-required-csv-annotations.md) for it to be displayed properly within the Marketplace UI.

## Pre-Requisite

You need to have an account with `quay.io`. If you don't have one you can sign up for it at [quay.io](https://quay.io).

### Obtain Access Token

All API requests made by executing HTTP verb (GET, POST, PUT or DELETE) against the API endpoint URL require an `Authorization header` with an access token. By logging in to `quay.io` you will receive an access token. You can get the token running the following bash script:

```bash
$ ./scripts/get-quay-token
```

You will be prompted to provide your `quay.io` account user name and password.

Upon successful invocation, you will receive a JSON response with an access token, as shown below. Save the value of the `token` field from the JSON response as we will be using this token to make calls to `quay.io` API.

```json
{
    "token": "basic XWtgc2hlbTpsZWR6ZXCwbYlf"
}
```

## Push An Operator Bundle

You can now push an operator bundle to `quay.io` by using the following bash script:

```bash
$ ./scripts/push-to-quay
```

You will be prompted for the path to your operator bundle file, your `quay.io` namespace, your operator's name, the operator's version/release, and your access token obtained from the previous step. Your operator name corresponds to the repository in your namespace where your operator bundle is stored. The repository will be created automatically if it doesn't already exist.

For example, if you have saved the operator bundle into a file named `myapp.yaml` in the root if this repository and want to push it to `myoperators/myapp:1.0.0` then the inputs will have the following values:

```bash
Relative path to your operator bundle file: myapp.yaml
Namespace in quay.io: myoperators
Operator name: myapp
Version/Release of operator: 1.0.0
Quay.io token (TOKEN value of ./scripts/get-quay-token ): basic XWtgc2hlbTpsZWR6ZXCwbYlf
```

You should now be able to visit `quay.io` and browse the uploaded operator bundle in your desired namespace. Your namespace can have multiple operator offerings, and `quay.io` will display each as a distinct repository. You cannot view the contents of your operator bundles from the `quay.io` website.

## Troubleshooting

### New Version/Release of Operator Bundle

Each release of an operator bundle is considered immutable. If you try to push to an existing release you will get the following error from `quay.io`.

```json
{
    "error": {
        "code":"package-exists","details":{},"message":"package exists already"
    }
}
```

If you change your operator bundle, you should always bump up the version of the version/release before pushing the bundle.

### Delete Operator Bundles

*You will need `admin` privileges to delete repositories.*

Operator bundles can be deleted manually by going to `quay.io`. Go to the `Settings` tab of your desired repository and click the `Delete Application` button. This will delete all uploaded versions of your operator bundle. There is no way to delete individual versions of your operator bundle.

### Making your Operator Visible

By default, when pushing an operator bundle to a new repository on `quay.io`, the repository will be created as private. Ensure that you make it public for it to be visible to the marketplace operator. Go to the `Settings` tab of your repository and check that under the `Application Visibility` header the application is set to public.
