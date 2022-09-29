FROM golang:1.17

EXPOSE 8082

ENV PROJECT_PATH=/go/src/github.com/replicatedhq/kots-lint
WORKDIR $PROJECT_PATH

COPY . .

RUN --mount=target=/root/.cache,type=cache make build

CMD ["make", "run"]
