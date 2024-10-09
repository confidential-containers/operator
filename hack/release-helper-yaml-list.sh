#!/bin/bash

RHYL_PREREQS=(
    "config/samples/enclave-cc/sim/kustomization.yaml"
    "config/samples/enclave-cc/hw/kustomization.yaml"
    "config/samples/ccruntime/default/kustomization.yaml"
    "config/samples/ccruntime/peer-pods/kustomization.yaml"
    "config/samples/ccruntime/s390x/kustomization.yaml"
)

RHYL_KATA=(
    "config/samples/ccruntime/default/kustomization.yaml"
    "config/samples/ccruntime/peer-pods/kustomization.yaml"
    "config/samples/ccruntime/s390x/kustomization.yaml"
)
