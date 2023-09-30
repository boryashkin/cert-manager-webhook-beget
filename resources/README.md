# Solver testdata directory

- Read 
    - https://cert-manager.io/docs/configuration/acme/dns01/
    - https://cert-manager.io/docs/configuration/acme/

- install cert-manager
- instal the issuer
    - helm install webhook-beget-local . -f values.yaml
    - kubectl apply -f secret.yaml
    - kubectl apply -f issuer.yaml
- request certificate
    - kubectp apply -f certificaete.yaml
