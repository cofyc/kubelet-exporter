package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/cofyc/kubelet-exporter/pkg/collectors"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

func metricsServer(registry prometheus.Gatherer, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := fmt.Sprintf(":%d", port)

	glog.Infof("Starting metrics server: %s", listenAddress)
	// Add metricsPath
	http.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	// Add healthzPath
	http.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	// Add index
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	<head>
		<title>Kube Metrics Server</title>
	</head>
	<body>
		<h1>Kube Metrics</h1>
		<ul>
			<li><a href='` + metricsPath + `'>metrics</a></li>
			<li><a href='` + healthzPath + `'>healthz</a></li>
		</ul>
	</body>
</html>`))
	})
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

var (
	optHelp           bool
	optPort           int
	optKubeletAddress string
)

func init() {
	flag.BoolVar(&optHelp, "help", false, "print help info and exit")
	flag.IntVar(&optPort, "port", 9859, "port to expose metrics on")
	flag.StringVar(&optKubeletAddress, "kubelet-address", "http://localhost:10255", "address of kubelet")
}

func main() {
	// We log to stderr because glog will default to logging to a file.
	flag.Set("logtostderr", "true")
	flag.Parse()

	if optHelp {
		flag.Usage()
		return
	}

	registry := prometheus.NewRegistry()
	u, err := url.Parse(optKubeletAddress)
	if err != nil {
		log.Fatal(err)
	}
	u.Path = "stats/summary"
	registry.MustRegister(collectors.NewVolumeStatsCollector(u.String()))
	metricsServer(registry, optPort)
}
