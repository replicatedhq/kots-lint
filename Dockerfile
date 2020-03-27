FROM golang:1.13

ADD ./pkg/kots/rego /rego
ADD ./kubernetes-json-schema /kubernetes-json-schema
ADD . /go/src/github.com/replicatedhq/kots-lint
WORKDIR /go/src/github.com/replicatedhq/kots-lint

EXPOSE 8082

RUN make build

CMD ["make", "run"]
