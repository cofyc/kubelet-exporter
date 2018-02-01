# Metrics

| Metric name | Metric type | Labels |
|-------------|-------------|-------------|
|kubelet_volume_stats_capacity_bytes|Gauge|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_available_bytes|Gauge|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_used_bytes|Gauge|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_inodes|Gauge|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_inodes_free|Gauge|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_inodes_used|Gauge|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 

## References

- https://github.com/kubernetes/kubernetes/pull/51553
- https://github.com/kubernetes/community/pull/855
