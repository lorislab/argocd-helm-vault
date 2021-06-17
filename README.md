# argocd-helm-vault

`` is helm wrapper which replace the values with `vault` in the helm output.

[![License](https://img.shields.io/github/license/lorislab/argocd-helm-vault?style=for-the-badge&logo=apache)](https://www.apache.org/licenses/LICENSE-2.0)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/lorislab/argocd-helm-vault?sort=semver&logo=github&style=for-the-badge)](https://github.com/lorislab/argocd-helm-vault/releases/latest)


# Installation

To install the `argocd-helm-vault` in the ArgoCD docker image you need to:

* Rename `helm` to `_helm`.
* Rename `helm2` to `_helm2`
* Download and install `argocd-helm-vault` and rename it to `helm`

For example:
```docker
FROM ghcr.io/lorislab/argocd-helm-vault:0.2.0 as release

FROM quay.io/argoproj/argocd:v2.0.2

USER root

RUN mv /usr/local/bin/helm /usr/local/bin/_helm
RUN mv /usr/local/bin/helm2 /usr/local/bin/_helm2
COPY --from=release /usr/local/bin/argocd-helm-vault /usr/local/bin/helm

USER argocd
```

# Configuration

* ARGOCD_HELM_VAULT_CMD - original helm command. Default `_helm`
* ARGOCD_HELM_VAULT_ROLE_ID - vault AppRole `RoleID`. Default empty.
* ARGOCD_HELM_VAULT_SECRET_ID - vault AppRole `SecretID`. Default emtpy.
* VAULT_ADDR - vault URL. Default empty
