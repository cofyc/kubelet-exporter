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

// Copied from https://github.com/kubernetes/kubernetes/blob/1ac56d8cbbdb72e75fb9b083f3f76afd4010e71e/pkg/kubelet/apis/stats/v1alpha1/types.go.

// FsStats contains data about filesystem usage.
type FsStats struct {
	// AvailableBytes represents the storage space available (bytes) for the filesystem.
	// +optional
	AvailableBytes *uint64 `json:"availableBytes,omitempty"`
	// CapacityBytes represents the total capacity (bytes) of the filesystems underlying storage.
	// +optional
	CapacityBytes *uint64 `json:"capacityBytes,omitempty"`
	// UsedBytes represents the bytes used for a specific task on the filesystem.
	// This may differ from the total bytes used on the filesystem and may not equal CapacityBytes - AvailableBytes.
	// e.g. For ContainerStats.Rootfs this is the bytes used by the container rootfs on the filesystem.
	// +optional
	UsedBytes *uint64 `json:"usedBytes,omitempty"`
	// InodesFree represents the free inodes in the filesystem.
	// +optional
	InodesFree *uint64 `json:"inodesFree,omitempty"`
	// Inodes represents the total inodes in the filesystem.
	// +optional
	Inodes *uint64 `json:"inodes,omitempty"`
	// InodesUsed represents the inodes used by the filesystem
	// This may not equal Inodes - InodesFree because this filesystem may share inodes with other "filesystems"
	// e.g. For ContainerStats.Rootfs, this is the inodes used only by that container, and does not count inodes used by other containers.
	InodesUsed *uint64 `json:"inodesUsed,omitempty"`
}

// VolumeStats contains data about Volume filesystem usage.
type VolumeStats struct {
	// Embedded FsStats
	FsStats
	// Name is the name given to the Volume
	// +optional
	Name string `json:"name,omitempty"`
	// Reference to the PVC, if one exists
	// +optional
	PVCRef *PVCReference `json:"pvcRef,omitempty"`
}

// PVCReference contains enough information to describe the referenced PVC.
type PVCReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// PodReference contains enough information to locate the referenced pod.
type PodReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
}

// PodStats holds pod-level unprocessed sample stats.
type PodStats struct {
	// Reference to the measured Pod.
	PodRef PodReference `json:"podRef"`
	// Stats pertaining to volume usage of filesystem resources.
	// VolumeStats.UsedBytes is the number of bytes used by the Volume
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	VolumeStats []VolumeStats `json:"volume,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
}

type kubeletStatsSummary struct {
	Pods []PodStats `json:"pods"`
}

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

	statsSummary := kubeletStatsSummary{}
	err = json.Unmarshal(rBody, &statsSummary)
	if err != nil {
		glog.Errorf("failed to parse stats summary from %s: %v", collector.host, err)
		return
	}

	addGauge := func(desc *prometheus.Desc, pvcRef *PVCReference, v float64, lv ...string) {
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
				pvcUniqStr := pvcRef.Namespace + pvcRef.Name
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
