FROM golang:1.22 AS builder

ADD . /go/src/github.com/replicatedhq/kots-lint
WORKDIR /go/src/github.com/replicatedhq/kots-lint

RUN make build


FROM debian:buster-slim

ENV DEBUG_MODE on

COPY --from=builder /go/src/github.com/replicatedhq/kots-lint/bin /app

EXPOSE 8082

CMD ["/app/kots-lint"]
