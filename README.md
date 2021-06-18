# argocd-helm-vault

`argocd-helm-vault` is helm wrapper that replaces the values with `vault` keys in the helm vaules and output.

[![License](https://img.shields.io/github/license/lorislab/argocd-helm-vault?style=for-the-badge&logo=apache)](https://www.apache.org/licenses/LICENSE-2.0)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/lorislab/argocd-helm-vault?sort=semver&logo=github&style=for-the-badge)](https://github.com/lorislab/argocd-helm-vault/releases/latest)

# How to

The helm wrapper searches for this pattern `<vault:{full-path}#{key}>` in detail kv-v2 `<vault:{engine}/data/{path}#{key}>`

Variables:
* engine - vault engine name
* path - vault path to secret
* key - vault secret key

For example `<vault:secret/data/webapp/config#username>`

It is possible to use function `b64enc` in the pattern for example: `<vault:secret/data/webapp/config#username | b64enc>`

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: test
spec:
  destination:
    name: ''
    namespace: test
    server: 'https://kubernetes.default.svc'
  source:
    path: ''
    repoURL: 'https://helm.repo.url/'
    targetRevision: '>=0.0.0-0'
    chart: ping-quarkus
    helm:
      releaseName: test
      values: |
        app:
          env:
            USERNAME: <vault:secret/data/webapp/config#username>
            PASSWORD: <vault:secret/data/webapp/config#password> 
  project: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

By default the wrapper replaces vault keys only in the values files due to the variable `ARGOCD_HELM_VAULT_VALUES`. 
To activate the replacement of vault keys in the helm output, you have to set the variable `ARGOCD_HELM_VAULT_OUTPUT` to `true`.

# Installation

To install the `argocd-helm-vault` in the ArgoCD docker image `argocd-repo-server` you need to:

* Rename `helm` to `_helm`.
* Rename `helm2` to `_helm2`
* Download and install `argocd-helm-vault` and rename it to `helm`

For example:
```docker
FROM ghcr.io/lorislab/argocd-helm-vault:0.4.0 as release

FROM quay.io/argoproj/argocd:v2.0.3

USER root

RUN mv /usr/local/bin/helm /usr/local/bin/_helm
RUN mv /usr/local/bin/helm2 /usr/local/bin/_helm2
COPY --from=release /usr/local/bin/argocd-helm-vault /usr/local/bin/helm

USER argocd
```

# Configuration

The wrapper could be configured with these variables:

* ARGOCD_HELM_VAULT_CMD - original helm command. Default `_helm`
* ARGOCD_HELM_VAULT_ROLE_ID - vault AppRole `RoleID`. Default `""`.
* ARGOCD_HELM_VAULT_SECRET_ID - vault AppRole `SecretID`. Default `""`.
* ARGOCD_HELM_VAULT_VALUES - replace vault keys in the helm values files. Default `true`.
* ARGOCD_HELM_VAULT_OUTPUT - replace vault keys in the helm output. Default `false`.
* VAULT_ADDR - vault server URL. Default `""`
