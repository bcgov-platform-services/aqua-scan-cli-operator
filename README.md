## Aqua Scanner Account Operator


This operator allows teams to create a CRD `AquaScannerAccount` in their tools namespace. When it is created the operator will manage a scope aqua account with scan priviledges. It will then store the credentials of the scan account as a status field in the operator. 

## Environment

The operator requires the following variables in the runtime:

1. `AQUA_URL string`: the base url to the aqua instance
2. `AQUA_USER string`: the aqua service account username that is needed to interact with the aqua api
3. `AQUA_PASSWORD string`: the credentials for the service account

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