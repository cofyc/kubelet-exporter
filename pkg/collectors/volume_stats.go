package collectors

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context/ctxhttp"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
)

const (
	volumeStatsCapacityBytesKey  = "kubelet_volume_stats_capacity_bytes"
	volumeStatsAvailableBytesKey = "kubelet_volume_stats_available_bytes"
	volumeStatsUsedBytesKey      = "kubelet_volume_stats_used_bytes"
	volumeStatsInodesKey         = "kubelet_volume_stats_inodes"
	volumeStatsInodesFreeKey     = "kubelet_volume_stats_inodes_free"
	volumeStatsInodesUsedKey     = "kubelet_volume_stats_inodes_used"
)

var (
	volumeStatsCapacityBytes = prometheus.NewDesc(
		volumeStatsCapacityBytesKey,
		"Capacity in bytes of the volume",
		[]string{"namespace", "persistentvolumeclaim"}, nil,
	)
	volumeStatsAvailableBytes = prometheus.NewDesc(
		volumeStatsAvailableBytesKey,
		"Number of available bytes in the volume",
		[]string{"namespace", "persistentvolumeclaim"}, nil,
	)
	volumeStatsUsedBytes = prometheus.NewDesc(
		volumeStatsUsedBytesKey,
		"Number of used bytes in the volume",
		[]string{"namespace", "persistentvolumeclaim"}, nil,
	)
	volumeStatsInodes = prometheus.NewDesc(
		volumeStatsInodesKey,
		"Maximum number of inodes in the volume",
		[]string{"namespace", "persistentvolumeclaim"}, nil,
	)
	volumeStatsInodesFree = prometheus.NewDesc(
		volumeStatsInodesFreeKey,
		"Number of free inodes in the volume",
		[]string{"namespace", "persistentvolumeclaim"}, nil,
	)
	volumeStatsInodesUsed = prometheus.NewDesc(
		volumeStatsInodesUsedKey,
		"Number of used inodes in the volume",
		[]string{"namespace", "persistentvolumeclaim"}, nil,
	)
)

// volumeStatsCollector collects metrics from kubelet stats summary.
type volumeStatsCollector struct {
	host string
}

// NewVolumeStatsCollector creates a new volume stats prometheus collector.
func NewVolumeStatsCollector(host string) prometheus.Collector {
	return &volumeStatsCollector{host: host}
}

// Describe implements the prometheus.Collector interface.
func (collector *volumeStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- volumeStatsCapacityBytes
	ch <- volumeStatsAvailableBytes
	ch <- volumeStatsUsedBytes
	ch <- volumeStatsInodes
	ch <- volumeStatsInodesFree
	ch <- volumeStatsInodesUsed
}

// Collect implements the prometheus.Collector interface.
func (collector *volumeStatsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := ctxhttp.Get(ctx, http.DefaultClient, collector.host)
	if err != nil {
		glog.Errorf("failed to get stats from %s: %v", collector.host, err)
		return
	}
	defer resp.Body.Close()
	rBody, _ := ioutil.ReadAll(resp.Body)

	statsSummary := v1alpha1.Summary{}
	err = json.Unmarshal(rBody, &statsSummary)
	if err != nil {
		glog.Errorf("failed to parse stats summary from %s: %v", collector.host, err)
		return
	}

	addGauge := func(desc *prometheus.Desc, pvcRef *v1alpha1.PVCReference, v float64, lv ...string) {
		lv = append([]string{pvcRef.Namespace, pvcRef.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	if statsSummary.Pods != nil {
		allPVCs := sets.String{}
		for _, podStats := range statsSummary.Pods {
			if podStats.VolumeStats == nil {
				continue
			}
			for _, volumeStat := range podStats.VolumeStats {
				pvcRef := volumeStat.PVCRef
				if pvcRef == nil {
					// ignore if no PVC reference
					continue
				}
				pvcUniqStr := pvcRef.Namespace + "/" + pvcRef.Name
				if allPVCs.Has(pvcUniqStr) {
					// ignore if already collected
					continue
				}
				addGauge(volumeStatsCapacityBytes, pvcRef, float64(*volumeStat.CapacityBytes))
				addGauge(volumeStatsAvailableBytes, pvcRef, float64(*volumeStat.AvailableBytes))
				addGauge(volumeStatsUsedBytes, pvcRef, float64(*volumeStat.UsedBytes))
				addGauge(volumeStatsInodes, pvcRef, float64(*volumeStat.Inodes))
				addGauge(volumeStatsInodesFree, pvcRef, float64(*volumeStat.InodesFree))
				addGauge(volumeStatsInodesUsed, pvcRef, float64(*volumeStat.InodesUsed))
				allPVCs.Insert(pvcUniqStr)
			}
		}
	}
}
