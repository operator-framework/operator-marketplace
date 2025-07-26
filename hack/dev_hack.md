
# Super Hacky Way to Get a Local Kind Cluster Working Enough to Manage OperatorHub and CatalogSource Objects

## Prerequisites
To use the Dockerfiles in this guide, you must log in to `registry.ci`. 

### Get Your OpenShift Token
1. Navigate to [OpenShift Console](https://console-openshift-console.apps.ci.l2s4.p1.openshiftapps.com).
2. In the top-right corner, click the menu and select **Copy Login Command**.
3. Copy the login command and use it in the following steps.

## Steps

### 1. Log in to `registry.ci`
Use the `podman` command to log in to the OpenShift registry. Replace the username and password with your own credentials:

```bash
podman login -u=btofel -p=sha256~<blahblahblah> registry.ci.openshift.org
```

> **Note**: Ensure your token is valid before proceeding.

### 2. Build the Marketplace Operator Docker Image
Now, you need to build the marketplace operator for ARM64 architecture:

```bash
podman build --arch arm64 -t localhost/marketplace-operator:latest -f Dockerfile
```

This will create a Docker image tagged as `localhost/marketplace-operator:latest`.

### 3. Save the Docker Image as a TAR File
Export the Docker image to a `.tar` file so it can be loaded into your `kind` cluster:

```bash
podman save -o marketplace-operator.tar localhost/marketplace-operator:latest
```

### 4. Load the Image into the Kind Cluster
Use `kind` to load the Docker image into your cluster named `catalogd`:

```bash
kind load image-archive marketplace-operator.tar --name=catalogd
```

### 5. Apply the Necessary Resources
Finally, apply the resources to the cluster using the following script:

```bash
hack/kind-apply.sh
```

This script will configure your local `kind` cluster enough to manage `OperatorHub` and `CatalogSource` objects.

> **Note**: This is a quick and dirty setup for local testing and development purposes.
        