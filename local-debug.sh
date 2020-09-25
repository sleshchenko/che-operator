#!/bin/bash
#
# Copyright (c) 2019 Red Hat, Inc.
# This program and the accompanying materials are made
# available under the terms of the Eclipse Public License 2.0
# which is available at https://www.eclipse.org/legal/epl-2.0/
#
# SPDX-License-Identifier: EPL-2.0
#
# Contributors:
#   Red Hat, Inc. - initial API and implementation

set -e

if [ $# -ne 1 ]; then
    echo -e "Wrong number of parameters.\nUsage: ./local-debug.sh <custom-resource-yaml>\n"
    exit 1
fi

command -v delv >/dev/null 2>&1 || { echo "operator-sdk is not installed. Aborting."; exit 1; }
command -v operator-sdk >/dev/null 2>&1 || { echo -e $RED"operator-sdk is not installed. Aborting."$NC; exit 1; }

CHE_NAMESPACE=che

set +e
kubectl create namespace $CHE_NAMESPACE
set -e

export OPERATOR_NAME=che-operator
export CONSOLE_LINK_NAME=che
export CONSOLE_LINK_DISPLAY_NAME=Eclipse Che
export CONSOLE_LINK_SECTION=Red Hat Applications
export CONSOLE_LINK_IMAGE=/dashboard/assets/branding/loader.svg
export CHE_IDENTITY_SECRET=che-identity-secret
export CHE_IDENTITY_POSTGRES_SECRET=che-identity-postgres-secret
export CHE_POSTGRES_SECRET=che-postgres-secret
export CHE_SERVER_TRUST_STORE_CONFIGMAP_NAME=ca-certs

export CHE_FLAVOR=che
export CONSOLE_LINK_NAME=che
export CONSOLE_LINK_DISPLAY_NAME=Eclipse Che
export CHE_VERSION=nightly
export RELATED_IMAGE_che_server=quay.io/eclipse/che-server:nightly
export RELATED_IMAGE_plugin_registry=quay.io/eclipse/che-plugin-registry:nightly
export RELATED_IMAGE_devfile_registry=quay.io/eclipse/che-devfile-registry:nightly
export RELATED_IMAGE_che_tls_secrets_creation_job=quay.io/eclipse/che-tls-secret-creator:alpine-d1ed4ad
export RELATED_IMAGE_pvc_jobs=registry.access.redhat.com/ubi8-minimal:8.2
export RELATED_IMAGE_postgres=centos/postgresql-96-centos7:9.6
export RELATED_IMAGE_keycloak=quay.io/eclipse/che-keycloak:nightly
export RELATED_IMAGE_che_workspace_plugin_broker_metadata=quay.io/eclipse/che-plugin-metadata-broker:v3.4.0
export RELATED_IMAGE_che_workspace_plugin_broker_artifacts=quay.io/eclipse/che-plugin-artifacts-broker:v3.4.0
export RELATED_IMAGE_che_server_secure_exposer_jwt_proxy_image=quay.io/eclipse/che-jwtproxy:0.10.0
export RELATED_IMAGE_single_host_gateway=docker.io/traefik:v2.2.8
export RELATED_IMAGE_single_host_gateway_config_sidecar=quay.io/che-incubator/configbump:0.1.4

kubectl apply -f deploy/crds/org_v1_che_crd.yaml
kubectl apply -f $1 -n $CHE_NAMESPACE
cp templates/keycloak_provision /tmp/keycloak_provision

operator-sdk run --local --watch-namespace ${CHE_NAMESPACE} --enable-delve
