FROM golang:1.17 AS builder

ADD . /go/src/github.com/replicatedhq/kots-lint
WORKDIR /go/src/github.com/replicatedhq/kots-lint

RUN make build


FROM debian:buster-slim

RUN apt-get update -y && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/src/github.com/replicatedhq/kots-lint/bin /app

EXPOSE 8082

CMD ["/app/kots-lint"]
