ARG IMAGE=debian:10.7-slim

FROM $IMAGE AS builder
ARG VERSION=1.0.0

ENV FILENAME=argocd-helm-vault_${VERSION}_Linux_x86_64.tar.gz

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates

RUN curl https://github.com/lorislab/argocd-helm-vault/releases/download/${VERSION}/${FILENAME} -O -J -L && \
    tar xfz $FILENAME argocd-helm-vault && \
    chmod +x argocd-helm-vault

FROM debian:10.7-slim

LABEL org.opencontainers.image.source https://github.com/lorislab/argocd-helm-vault

COPY --from=builder argocd-helm-vault /usr/local/bin/argocd-helm-vault

