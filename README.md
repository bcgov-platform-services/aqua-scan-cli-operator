## Aqua Scanner Account Operator


This operator allows teams to create a CRD `AquaScannerAccount` in their tools namespace. When it is created the operator will manage a scope aqua account with scan priviledges. It will then store the credentials of the scan account as a status field in the operator. 

- [Operation](#Operation)
- [Development](#Development)
  - [Webhook Certificate Generation](#webhook-certificate-generation)
## Operation

### Environment

The operator requires the following variables in the runtime:

1. `AQUA_URL string`: the base url to the aqua instance
2. `AQUA_USER string`: the aqua service account username that is needed to interact with the aqua api
3. `AQUA_PASSWORD string`: the credentials for the service account

### Installing Operator

> based off of the Go Operator SDK Documentation

`make install` will install the CRD. If you make changes to the CRD spec you will need to regenerate the CRD and install. Ensure CRD spec changes are backwards compatible. 

### Building

Use the [Openshift SDK guide](https://docs.openshift.com/container-platform/4.8/operators/operator_sdk/golang/osdk-golang-tutorial.html#osdk-bundle-deploy-olm_osdk-golang-tutorial) over the generic k8s one. 

The image is pushed into our shared dockerhub account. Only users with access to the account may push. 

#### Updating Image Version

Update the [`VERSION`](https://github.com/bcgov-platform-services/aqua-scan-cli-operator/blob/main/Makefile#L6) variable in the Makefile

#### To build image locally

Building locally also involves running `envTest` which requires a special environment. The [kubebuilder documentation](https://book.kubebuilder.io/reference/envtest.html) outlines setting up env test correctly. Wherever you unpack the kubebuilder binaries is the path you will need to set for `KUBEBUILDER_ASSETS`. Once this is complete you can run `docker-build` similar to

`make docker-build KUBEBUILDER_ASSETS=/opt/kubebuilder/testbin/bin`

#### To push image

`make docker-push`

### Generating Manifests

Manifest generation is still hokey and does require improvements. 

1. Make sure your kustomize manifests have been generated and everything is up to date. `make manifests && make generate`
2. Run the custom `make genmanifests`. This will generate a single `operator.yaml` file that you can use to translate to CCM
  - Remove the `Namespace` Object. In CCM we are not creating a seperate namespace for the operator (it lives where Aqua lives)
  - Remove any namespace references in other objects that refer to the above deleted `Namespace`
  - Remove the `app-version: <version>` label and selector from all objects (keeping the label/selector is backwards incompatible)
3. Test infrastructure code seperately by running `oc apply` as needed
4. Make sure you have the latest image in dockerhub (see README on docker builds and deploys)
I have made a custom make command to pull out all the prod infracode you'll need for CCM make genmanifests
a. use that code as a template for translating to CCM
b. ensure no new namespaces are created!
c. modify the translated code and remove:
namespace references
additional labels/selectors such as app-version: v1 from services and deployments
`make genmanifests` will generate the file called `operator.yaml`

### Other commands

You can view the remaining commands in the makefile


### Deploying

The operator expects a secret file called `aqua-scanner-operator-creds` with the following keys

- `AQUA_URL`: the base url (no trailing slash) to the aqua instance
- `AQUA_USER`: the aqua user account
- `AQUA_PASSWORD`: the aqua password for the account

> The aqua user must have `administrator` priviledges

## Development

Before you develop this operator further it is strongly recommended you run through the operator sdk tutorial as well as the kube builder tutorial. It will save you a TON of time!

### Webhook Certificate Generation

This codebase utilized the operator-sdk to generate a webhook to manage conversion between apiVersions that are available for the CRD and the storage version `v1`. In order for this to work, the webhook must serve traffic through HTTPS. The webhook expects a certificate to be located within `/tmp/k8s-webhook-server/serving-certs` inside `deployments.apps/aqua-scanner-operator-controller-manager` manager container.

Typically __Cert Manager__ would be used in this case to automatically manage generation and renewal of a certificate. At this time (Dec 2021), Cert Manager is not installable and so you will need another solution to generate a certificate. The option currently being used is a [service serving certificate](https://docs.openshift.com/container-platform/4.7/security/certificates/service-serving-certificate.html). 

## Managment

The operator is managed by ArgoCD in the private platform-services instance via CCM

