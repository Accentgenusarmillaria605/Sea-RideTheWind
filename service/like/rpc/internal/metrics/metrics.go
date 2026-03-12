package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	LikeActionCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "",
		Subsystem: "like-rpc",
		Name:      "action_total",
		Help:      "Total number of like/dislike actions",
	}, []string{"target_type", "action", "result"})

	LikeQueryCacheCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "",
		Subsystem: "like_rpc",
		Name:      "query_cache_total",
		Help:      "Cache hit/miss counts for like queries",
	}, []string{"target_type", "result"})
)

func InitMetrics() {
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	prometheus.Register(LikeActionCount)
	prometheus.Register(LikeQueryCacheCount)
}
