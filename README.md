<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="32" width="32" alt="cert-manager project logo" />
</p>

# [Beget](https://beget.com/p259374) DNS01 webhook 

## Status

The module is active, but the underlying API is rarely changing, not much to update yet. Give it a star, if you're using it.

## Installation

- Read 
    - https://cert-manager.io/docs/configuration/acme/dns01/
    - https://cert-manager.io/docs/configuration/acme/

- install [cert-manager](https://github.com/cert-manager/cert-manager)
    - `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.1/cert-manager.yaml`
- instal the issuer:
    - **NOTE**: The kubernetes resources used to install the Webhook should be deployed within the same namespace as the cert-manager ("cert-manager" by default, check ./deploy/values.yaml).
    - `helm repo add boryashkin https://boryashkin.github.io/helm-charts/`
    - `helm repo update`
    - `helm install cert-beget boryashkin/cert-manager-beget-webhook`
    - OR
      - pull this repo
      - `helm install webhook-beget ./deploy/beget -f ./deploy/values.yaml -n cert-manager`
- create a secret for beget API
- create an issuer
- request certificates
- add the certificates to services

Follow ***an example*** for details: [testdata/resources](testdata/resources/README.md).

## Tests

You can run the webhook test suite with:

```bash
$ TEST_ZONE_NAME=example.com. make test
```
