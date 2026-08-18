// Package metrics 提供可选的 Prometheus 指标与 HTTP 暴露端点。
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 代理服务的运行指标。
type Metrics struct {
	SendTotal         *prometheus.CounterVec
	BackendErrorTotal *prometheus.CounterVec
	ActiveConnections prometheus.Gauge
}

// New 创建并注册指标。
func New() *Metrics {
	m := &Metrics{
		SendTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mailproxy_send_total",
			Help: "Total number of mails relayed, by result.",
		}, []string{"result"}),
		BackendErrorTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mailproxy_backend_error_total",
			Help: "Total number of backend SMTP errors, by backend id.",
		}, []string{"backend"}),
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mailproxy_active_connections",
			Help: "Number of active client connections.",
		}),
	}
	prometheus.MustRegister(m.SendTotal, m.BackendErrorTotal, m.ActiveConnections)
	return m
}

// Handler 返回 Prometheus 抓取端点的 HTTP handler。
func Handler() http.Handler {
	return promhttp.Handler()
}
