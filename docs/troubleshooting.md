# Troubleshooting Marketplace Operator

This document lists out some common failures related to the Marketplace Operator a user might encounter, along with ways to troubleshoot the failures. If you encounter a failure related to the Marketplace Operator that is not listed in this document, but should be, please open an issue or a PR to have the failure appended to this document. 

The troubleshooting steps listed here are for Marketplace resources like OperatorSource and CatalogSourceConfig. To troubleshoot [Operator-lifecycle-manager(OLM)](https://github.com/operator-framework/operator-lifecycle-manager) defined resources like ClusterServiceVersion, InstallPlan, Subscription etc, refer to the [OLM troubleshooting guide](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/debugging.md).

Note that all examples in this doc are are meant to be ran against an OpenShift cluster. All examples (except for the ones involving the UI) should work with `kubectl` and with the appropriate namespaces substituted in them.   

Table of contents
===================

1. [No packages show up in the UI (No OperatorHub Items Found)](#no-packages-show-up-in-the-ui-no-operatorhub-items-found)
2. [Operators(s) in an OperatorSource fail to show up in the UI](#operators-in-an-operatorsource-fail-to-show-up-in-the-ui) 
3. [Conflicting Package Names](#conflicting-package-names)
4. [Changes get overwritten on default CatalogSources](#changes-get-overwritten-on-default-catalogsources)

## No packages show up in the UI (No OperatorHub Items Found)

When you install an OpenShift cluster, OperatorHub comes with a default list of operators that can be installed onto the cluster. A number of resources need to be investigated if however you see a message like this: 

![Operator Hub Error Image](images/OperatorHubError.png)


First, ensure the CatalogSources are present:

```
$ oc get catalogsource -n openshift-marketplace 

NAME                  NAME                  TYPE      PUBLISHER   AGE
certified-operators   Certified Operators   grpc      Red Hat     23h
community-operators   Community Operators   grpc      Red Hat     23h
redhat-operators      Red Hat Operators     grpc      Red Hat     23h

```
If you need help debugging the CatalogSources, see [Where to go for help](#where-to-go-for-help). 

Next, make sure that the `packagemanifests` were created by the `OLM package-server`. 

```
$ oc get packagemanifests -n openshift-marketplace

NAME                             CATALOG               AGE
planetscale-certified            Certified Operators   40h
robin-operator                   Certified Operators   40h
storageos                        Certified Operators   40h
synopsys-certified               Certified Operators   40h
amq-streams                      Red Hat Operators     40h
codeready-workspaces             Red Hat Operators     40h
camel-k                          Community Operators   40h
cluster-logging                  Community Operators   40h
cockroachdb                      Community Operators   40h
descheduler                      Community Operators   40h
```

If no `packagemanifests` were created, check the logs for the `package-server` pods.

```
$ oc get pods -n openshift-operator-lifecycle-manager -l app=packageserver

NAME                                READY     STATUS    RESTARTS   AGE
packageserver-6c664f6d76-jj24b      1/1       Running   0          3h54m
packageserver-6c664f6d76-kzpd9      1/1       Running   0          3h53m

$ oc logs packageserver-6c664f6d76-jj24b -n openshift-operator-lifecycle-manager
$ oc logs packageserver-6c664f6d76-kzpd9 -n openshift-operator-lifecycle-manager

```

The logs for the `package-server` pods should contain information about why it might have failed to detect the presence of the `CatalogSources`. If the `package-server` pod does not show up or is crashing (anything other than healthy), try killing the pod or editing the deployment you see when you `oc get deployments -n openshift-operator-lifecycle-manager`.

If everything seems healthy, and still no packages show up in the UI, it could be a browser issue. Check the the browser console to see the logs for possible errors. At any point in the steps above if the error looks too complicated to debug, see [Where to go for help](#where-to-go-for-help). 

## Operator(s) in an OperatorSource fail to show up in the UI

If operators in a particular OperatorSource fail to show up in the UI, it could be because those operators had parsing errors and were ignored by the registry pod. To inspect the corresponding registry pod logs, first identify the name of the registry pod for the OperatorSource with `oc get pods -n openshift-marketplace` (the name of the pod should be of the format `<operator-source-name>-<random-characters>`). Get the logs for the pod with `oc logs <identified-pod-name> -n openshift-marketplace`.

If there are private app-registry repositories in your namespace, not specifying the authenticationToken in the OperatorSource CR will result in them not being listed. Please follow the steps [here](https://github.com/operator-framework/operator-marketplace/blob/master/docs/how-to-authenticate-private-repositories.md) for adding the token to the CR.  

## Conflicting Package Names

Package names are global within a CatalogSource. If two CatalogSources contain a package with the same name, the CatalogSource priority determines which one gets installed. A higher priority CatalogSource will be used before ones with lower priority. Users can view existing package names with the following command:

```bash
$ oc get packagemanifests -n openshift-marketplace
```

## Changes get overwritten on default CatalogSources

By default, Marketplace restores the [default CatalogSources](./defaults) if deleted or changed, including edits like adjusting priorities. To ensure these changes persist, you can use the `OperatorHub` API. This involves creating an `OperatorHub` object, similar to [the example OperatorHub](deploy/example/operatorhub.yaml).

Here is a description of the `OperatorHub` spec fields:

- `disableAllDefaultSources` is a boolean that can be used to disable all the default hub sources. If this is true, the `sources` field can be used to enable any default sources required. Otherwise, `sources` can be used to enable or disable a deault source.

- `sources` specifies a list of default hub sources and their configuration. If empty, then the value of `disableAllDefaultSources` determines whether the default sources are disabled. If `disableAllDefaultSources` is true, then `sources` can override the configuration to enable specific default sources.

# Where to go for help

* #kubernetes-operators channel in the [upstream Kubernetes slack instance](https://slack.k8s.io/)
* [google group for Operator-Framework](https://groups.google.com/forum/#!forum/operator-framework)