#!/bin/bash

set -euo pipefail

function kubernetes_schemas() {
    local version=$1

    # Prefix the version with v if it's not already there
    version="v${version#v}"

    # Create a temporary directory for cloning
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Clone the kubernetes-json-schema repository
    # This repo is really large, so we use a sparse checkout to only get the version we need
    (cd "$TMP_DIR" && git init && git remote add -f origin https://github.com/yannh/kubernetes-json-schema.git)
    (cd "$TMP_DIR" && git config core.sparseCheckout true)
    echo "$version-standalone-strict/" >> "$TMP_DIR"/.git/info/sparse-checkout
    (cd "$TMP_DIR" && git pull origin master)

    # Create kubernetes schema directory if it doesn't exist
    mkdir -p kubernetes_json_schema/schema/"$version"-standalone-strict

    # Copy all schema files
    cp -r "$TMP_DIR"/"$version"-standalone-strict/*.json kubernetes_json_schema/schema/"$version"-standalone-strict/

    echo "Successfully copied kubernetes $version schemas"
}

function replace_kubernetes_version() {
	local version=$1

    # Strip the version of any v prefix
    version="${version#v}"

    # Replace the version in the kubernetes_json_schema/schema.go file
    sed "s/KUBERNETES_LINT_VERSION = \"[^\"]*\"/\KUBERNETES_LINT_VERSION = \"$version\"/g" kubernetes_json_schema/schema.go > kubernetes_json_schema/schema.go.tmp
    mv kubernetes_json_schema/schema.go.tmp kubernetes_json_schema/schema.go
}

function main() {
	kubernetes_schemas "$@"
	replace_kubernetes_version "$@"
}

main "$@"
