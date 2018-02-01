FROM alpine:3.6

COPY kubelet-exporter /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/kubelet-exporter"]
