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
All API requests made by executing HTTP verb (GET, POST, PUT or DELETE) against the API endpoint URL require an `Authorization header` with an access token. By logging in to `quay.io` you will receive an access token.  

* Save the following excerpt into a file, for example `login.sh`  
```bash
#!/bin/sh

USERNAME=$1
PASSWORD=$2

curl -H "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
{
    "user": {
        "username": "'"${USERNAME}"'",
        "password": "'"${PASSWORD}"'"
    }
}'
```

* Execute `login.sh` by providing your `quay.io` account user name and password as arguments
```bash
$ chmod a+x login.sh
$ ./login.sh {username} {password}
```

Upon successful invocation, you will receive a JSON response with an access token, as shown below. Save the value of the `token` field from the JSON response. We will use this token to make calls to `quay.io` API. 
```json
{
    "token": "basic XWtgc2hlbTpsZWR6ZXCwbYlf"
}
```

## Push Operator Manifest
* Save the manifest into a file, for example `myapp.yaml`
* Copy the following excerpt and save it into a file, for example `push.sh`. Put appropriate values for the variables.

```bash
#!/bin/sh

FILENAME="file name (without the extension)"
NAMESPACE="namespace in quay.io"
REPOSITORY="repository name in quay.io"
RELEASE="version/release of the operator"
TOKEN="basic {access token here}"

function cleanup() {
    rm -f ${FILENAME}.tar.gz
}
trap cleanup EXIT

tar czf ${FILENAME}.tar.gz ${FILENAME}.yaml

BLOB=$(cat ${FILENAME}.tar.gz | base64 -w 0)

curl -H "Content-Type: application/json" \
     -H "Authorization: ${TOKEN}" \
     -XPOST https://quay.io/cnr/api/v1/packages/${NAMESPACE}/${REPOSITORY} -d '
{
    "blob": "'"${BLOB}"'",
    "release": "'"${RELEASE}"'",
    "media_type": "helm"
}'
```

If you have saved the manifest into `myapp.yaml` and want to push it to `myoperators/myapp:1.0.0` then the variables will have the following values.
```bash
FILENAME="myapp"
NAMESPACE="myoperators"
REPOSITORY="myapp"
RELEASE="1.0.0"
```

* Execute the script 
```bash
$ chmod a+x push.sh
$ ./push.sh
```

You can visit `quay.io` and browse the operator manifest in your desired namespace.

### New Version/Release of Operator Manifest
Each release of an operator manifest is considered immutable. If you try to push to an existing release you will get the following error from quay.
```json
{
    "error": {
        "code":"package-exists","details":{},"message":"package exists already"
    }
}
```

If you change your operator manifest then bump up the version in `RELEASE` before pushing the manifest.

### Delete Operator Manifest
*You will need `admin` privileges to delete repositories.*

For now, we have been deleting manifest(s) manually by going to `quay.io`. Go to the `Settings` tab of your desired repository and click the `Delete Application` button. 