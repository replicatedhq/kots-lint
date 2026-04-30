#!/usr/bin/env bash
# Run integration tests against the kots-lint docker image.
#
# Builds (or reuses) the kots-lint image, starts a container, waits for /livez,
# runs the Go integration test suite against it, and tears the container down.
#
# Environment variables:
#   IMAGE        Docker image to test. Default: kots-lint:integration-test
#   SKIP_BUILD   If set, skip docker build and assume IMAGE already exists.
#   PORT         Host port to bind the container to. Default: 8082
#   KEEP_RUNNING If set, do not stop the container after tests finish.
#   GO_TEST_ARGS Extra args appended to `go test` (e.g. -run TestLint/foo -v).
set -euo pipefail

cd "$(dirname "$0")/.."

IMAGE="${IMAGE:-kots-lint:integration-test}"
PORT="${PORT:-8082}"
CONTAINER_NAME="kots-lint-integration-$$"
BASE_URL="http://localhost:${PORT}"

cleanup() {
    if [ -z "${KEEP_RUNNING:-}" ]; then
        echo ">> Stopping container ${CONTAINER_NAME}"
        docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    else
        echo ">> KEEP_RUNNING set; leaving ${CONTAINER_NAME} running on ${BASE_URL}"
    fi
}
trap cleanup EXIT

if [ -z "${SKIP_BUILD:-}" ]; then
    echo ">> Building image ${IMAGE}"
    docker build -t "${IMAGE}" .
fi

echo ">> Starting container ${CONTAINER_NAME} on port ${PORT}"
docker run -d --rm --name "${CONTAINER_NAME}" -p "${PORT}:8082" "${IMAGE}" >/dev/null

echo ">> Waiting for ${BASE_URL}/livez"
for i in $(seq 1 60); do
    if curl -fsS -o /dev/null "${BASE_URL}/livez"; then
        echo ">> Service is ready (took ${i}s)"
        break
    fi
    if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo "!! Container exited unexpectedly. Logs:"
        docker logs "${CONTAINER_NAME}" 2>&1 || true
        exit 1
    fi
    sleep 1
    if [ "$i" -eq 60 ]; then
        echo "!! Timed out waiting for ${BASE_URL}/livez. Logs:"
        docker logs "${CONTAINER_NAME}" 2>&1 || true
        exit 1
    fi
done

echo ">> Running integration tests against ${BASE_URL}"
set +e
KOTS_LINT_BASE_URL="${BASE_URL}" \
    go test -v -count=1 -tags integration -timeout 5m ./test/integration/... ${GO_TEST_ARGS:-}
TEST_EXIT=$?
set -e

echo ">> Container logs (${CONTAINER_NAME}):"
docker logs "${CONTAINER_NAME}" 2>&1 || true

exit ${TEST_EXIT}
