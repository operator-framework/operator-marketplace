# Creating an Operator Artifact from Custom Resource Files

This document walks through the process of creating an operator artifact from your operator resources.

## Gather your Operator's Resource Files

Obtain all `CustomResourceDefinitions`, `ClusterServiceVersions`, and `packages` related to your operator. Place all of these files into a directory named after your operator in this directory. The name of the directory you create will be used in the following steps as well.

## Add your Operator to values.yaml

Modify `values.yaml` to list your operator under `operators`. Make sure the name listed matches the name of the directory you created in the previous step.

## Create helm template for your Operator

Copy `templates/amq-streams.yaml` into a new file named `templates/${YOUR-OPERATOR}.yaml`. In this new file, replace all instances of `amq-streams` with the name of your operator, which should match the name of the directory you created in the first step.

## Run helm template

The following command will create operator artifacts for all operators listed in `values.yaml`. If you only want to create the operator artifact for your operator, modify `values.yaml` to solely list your operator under `operators`.

```bash
helm template -f values.yaml . --output-dir .
```

Your operator artifact(s) will be created in the `./operator-artifacts/templates/` directory.

## Pushing your Operator Artifact to an App Registry

Follow the steps [here](../docs/how-to-upload-artifact.md) for how to push your Operator Artifact to Quay's App Registry and add your operator offering to an OpenShift cluster.
