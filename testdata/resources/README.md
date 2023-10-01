# Solver testdata directory

- install [cert-manager](https://github.com/cert-manager/cert-manager)
    - `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.1/cert-manager.yaml`
- instal the issuer:
    - **NOTE**: look at `./deploy/beget/values.yaml`
    - `helm install webhook-beget ./deploy/beget -f ./deploy/beget/values.yaml -n cert-manager`
    - `kubectl apply -f testdata/resources/secret.yaml`
    - `kubectl apply -f testdata/resources/issuer.yaml`
- request certificate
    - `kubectl apply -f testdata/resources/certificate.yaml`
- add the certificate to a service
    - install [nginx ingress](https://cert-manager.io/docs/tutorials/acme/nginx-ingress/)
        - `helm install quickstart ingress-nginx/ingress-nginx`
    - `kubectl apply -f testdata/resources/service.yaml`
- namespaced:
    - `kubectl create ns someother`
    - request certificate
        - `kubectl apply -f testdata/resources/someothernamespace/certificate.yaml`
    - add the certificate to a service
        - `kubectl apply -f testdata/resources/someothernamespace/service.yaml`

For debugging use:

- `kubectl logs webhook-your-pod-name -f`
- `kubectl get certificate -A` then describe
- `kubectl get certificaterequest -A` then describe
- `kubectl get secret -A`


