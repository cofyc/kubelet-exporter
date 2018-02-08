
all: test
	go install github.com/cofyc/kubelet-exporter/cmd/kubelet-exporter

test:
	go test -timeout 5m github.com/cofyc/kubelet-exporter/...
