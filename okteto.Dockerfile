FROM golang:1.24

EXPOSE 8082
EXPOSE 2345

ENV PROJECT_PATH=/go/src/github.com/replicatedhq/kots-lint

WORKDIR $PROJECT_PATH

RUN go install github.com/go-delve/delve/cmd/dlv@v1.20.2

COPY . .

RUN --mount=target=/root/.cache,type=cache make build

CMD ["make", "run"]
