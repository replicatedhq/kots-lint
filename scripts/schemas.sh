#!/bin/bash

set -euo pipefail

function kots_schemas() {
    # Create a temporary directory for cloning
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Clone the kotskinds repository
    git clone --depth 1 https://github.com/replicatedhq/kotskinds.git "$TMP_DIR"

    # Create kots schema directory if it doesn't exist
    mkdir -p kubernetes_json_schema/schema/kots

    # Copy all schema files
    cp -r "$TMP_DIR"/schemas/*.json kubernetes_json_schema/schema/kots/

    echo "Successfully copied KOTS schemas"
}

function embedded_cluster_schemas() {
    # Create a temporary directory for cloning
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Clone the embedded-cluster repository
    git clone --depth 1 https://github.com/replicatedhq/embedded-cluster.git "$TMP_DIR"

    # Create embedded-cluster schema directory if it doesn't exist
    mkdir -p kubernetes_json_schema/schema/embeddedcluster

    # Copy all schema files
    cp -r "$TMP_DIR"/operator/schemas/*.json kubernetes_json_schema/schema/embeddedcluster/

    echo "Successfully copied embedded-cluster schemas"
}

function troubleshoot_schemas() {
    # Create a temporary directory for cloning
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Clone the troubleshoot repository
    git clone --depth 1 https://github.com/replicatedhq/troubleshoot.git "$TMP_DIR"

    # Create troubleshoot schema directory if it doesn't exist
    mkdir -p kubernetes_json_schema/schema/troubleshoot

    # Copy all schema files
    cp -r "$TMP_DIR"/schemas/*.json kubernetes_json_schema/schema/troubleshoot/

    echo "Successfully copied troubleshoot schemas"
}

function main() {
	kots_schemas
	embedded_cluster_schemas
	troubleshoot_schemas
}

main
