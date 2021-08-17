## Aqua Scanner Account Operator


This operator allows teams to create a CRD `AquaScannerAccount` in their tools namespace. When it is created the operator will manage a scope aqua account with scan priviledges. It will then store the credentials of the scan account as a status field in the operator. 

## Environment

The operator requires the following variables in the runtime:

1. `AQUA_URL string`: the base url to the aqua instance
2. `AQUA_USER string`: the aqua service account username that is needed to interact with the aqua api
3. `AQUA_PASSWORD string`: the credentials for the service account