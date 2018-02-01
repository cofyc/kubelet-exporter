FROM golang:1.9.2

ADD . /go/src/github.com/cofyc/kubelet-exporter

RUN set -eux \
    && cd /go/src/github.com/cofyc/kubelet-exporter \
    && go get github.com/golang/dep/cmd/dep \
    && dep ensure -v -vendor-only \
    && go install github.com/cofyc/kubelet-exporter/cmd/kubelet-exporter

FROM alpine:3.7

RUN set -eux \
    && apk --no-cache add ca-certificates

COPY --from=0 /go/bin/kubelet-exporter /usr/local/bin
ENTRYPOINT ["/usr/local/bin/kubelet-exporter"]
