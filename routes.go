package bulk_ops

import (
	"net/http"

	"github.com/bredtape/bulk_ops/xml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricHTTPStatus = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Count of HTTP requests with path, method and (return) code",
	}, []string{"path", "method", "code"})

	metricHTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_duration_seconds",
		Help: "Histogram of HTTP response duration with path, method and (return) code",
	}, []string{"path", "method", "code"})
)

func registerRoutes(mux *http.ServeMux) error {
	{
		path := "/xml/xpath/prune"
		mux.Handle("POST "+path,
			promhttp.InstrumentHandlerDuration(metricHTTPDuration.MustCurryWith(prometheus.Labels{"path": path}),
				promhttp.InstrumentHandlerCounter(metricHTTPStatus.MustCurryWith(prometheus.Labels{"path": path}),
					xml.HandlePruneXPath())))
	}
	mux.Handle("GET /metrics", promhttp.Handler())

	return nil
}
