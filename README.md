## Aqua Scanner Account Operator


This operator allows teams to create a CRD `AquaScannerAccount` in their tools namespace. When it is created the operator will manage a scope aqua account with scan priviledges. It will then store the credentials of the scan account as a status field in the operator. 

## Environment

The operator requires the following variables in the runtime:

1. `AQUA_URL string`: the base url to the aqua instance
2. `AQUA_USER string`: the aqua service account username that is needed to interact with the aqua api
3. `AQUA_PASSWORD string`: the credentials for the service account

## Installing Operator

> based off of the Go Operator SDK Documentation

`make install` will install the CRD. If you make changes to the CRD spec you will need to regenerate the CRD and install. Ensure CRD spec changes are backwards compatible. 

## Building

Use the [Openshift SDK guide](https://docs.openshift.com/container-platform/4.8/operators/operator_sdk/golang/osdk-golang-tutorial.html#osdk-bundle-deploy-olm_osdk-golang-tutorial) over the generic k8s one. 

The image is pushed into our shared dockerhub account. Only users with access to the account may push. 

### To build image locally

Building locally also involves running `envTest` which requires a special environment. The [kubebuilder documentation](https://book.kubebuilder.io/reference/envtest.html) outlines setting up env test correctly. Wherever you unpack the kubebuilder binaries is the path you will need to set for `KUBEBUILDER_ASSETS`. Once this is complete you can run `docker-build` similar to

`make docker-build KUBEBUILDER_ASSETS=/opt/kubebuilder/testbin/bin`

### To push image

`make docker-push`

### Generating Manifests

`make genmanifests` will generate the file called `operator.yaml`

### Other commands

You can view the remaining commands in the makefile


## Deploying

The operator expects a secret file called `aqua-scanner-operator-creds` with the following keys

- `AQUA_URL`: the base url (no trailing slash) to the aqua instance
- `AQUA_USER`: the aqua user account
- `AQUA_PASSWORD`: the aqua password for the account

> The aqua user must have `administrator` priviledges

## Development

Before you develop this operator further it is strongly recommended you run through the operator sdk tutorial as well as the kube builder tutorial. It will save you a TON of time!