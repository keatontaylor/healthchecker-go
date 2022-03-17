package healthchecker

import (
	"context"
	"log"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type customMetric struct {
	url         string
	status      float64
	totalMS     float64
	dnsMS       float64
	firstbyteMS float64
	connectMS   float64
}

type HealthChecker struct {
	ctx                   context.Context
	urlStatus             *prometheus.GaugeVec
	urlMs                 *prometheus.GaugeVec
	urlDNS                *prometheus.GaugeVec
	urlFirstByte          *prometheus.GaugeVec
	urlConnectTime        *prometheus.GaugeVec
	urls                  []string
	healthcheck_invertval time.Duration
}

func NewHealthChecker(ctx context.Context, inverval time.Duration, urls []string) (hc *HealthChecker) {
	hc = &HealthChecker{
		ctx: ctx,
		urlStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sample",
			Subsystem: "external",
			Name:      "url_up",
			Help:      "Status of the URL as a integer value",
		}, []string{"url"}),
		urlMs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sample",
			Subsystem: "external",
			Name:      "url_response_ms",
			Help:      "Response time in milliseconds it took for the URL to respond.",
		}, []string{"url"}),
		urlDNS: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sample",
			Subsystem: "external",
			Name:      "url_dns_ms",
			Help:      "Response time in milliseconds it took for the DNS request to take place.",
		}, []string{"url"}),
		urlFirstByte: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sample",
			Subsystem: "external",
			Name:      "url_first_byte_ms",
			Help:      "Response time in milliseconds it took to retrive the first byte.",
		}, []string{"url"}),
		urlConnectTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "sample",
			Subsystem: "external",
			Name:      "url_connect_time_ms",
			Help:      "Response time in milliseconds it took to establish the inital connection.",
		}, []string{"url"}),
		healthcheck_invertval: inverval,
		urls:                  urls,
	}
	prometheus.MustRegister(hc.urlStatus, hc.urlMs, hc.urlDNS, hc.urlConnectTime, hc.urlFirstByte)
	http.Handle("/metrics", promhttp.Handler())
	return hc
}

func (hc *HealthChecker) updateCustomMetrics(cm *customMetric) {
	log.Printf("Updating custom metrics: url: %s, connectMS: %.0f, dnsMS: %.0f, firstbyteMS: %.0f, totalMS: %.0f, status: %.0f",
		cm.url,
		cm.connectMS,
		cm.dnsMS,
		cm.firstbyteMS,
		cm.totalMS,
		cm.status,
	)
	hc.urlDNS.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.dnsMS)
	hc.urlConnectTime.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.connectMS)
	hc.urlMs.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.totalMS)
	hc.urlFirstByte.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.firstbyteMS)
	hc.urlStatus.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.status)
}

func (hc *HealthChecker) fetchStats(url string) {

	var start, connect, dns time.Time

	var connectMS, dnsMS, firstbyteMS, totalMS, status float64

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			dnsMS = float64(time.Since(dns).Milliseconds())
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			connectMS = float64(time.Since(connect).Milliseconds())
		},

		GotFirstResponseByte: func() {
			firstbyteMS = float64(time.Since(start).Milliseconds())
		},
	}

	req, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(hc.ctx, trace),
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		log.Println(err)
	}

	start = time.Now()
	if res, err := http.DefaultClient.Do(req); err != nil {
		log.Printf("error/timeout getting http request %s", err)
	} else {
		if res.StatusCode >= 200 && res.StatusCode <= 299 {
			status = 1
		} else {
			status = 0
		}
		totalMS = float64(time.Since(start).Milliseconds())
		hc.updateCustomMetrics(
			&customMetric{
				url:         url,
				dnsMS:       dnsMS,
				connectMS:   connectMS,
				firstbyteMS: firstbyteMS,
				totalMS:     totalMS,
				status:      status,
			},
		)
	}
}

func (hc *HealthChecker) StartCollector() {
	ticker := time.NewTicker(hc.healthcheck_invertval)
	log.Println("starting collector")
	go func() {
		for {
			select {
			case <-ticker.C:
				for _, u := range hc.urls {
					hc.fetchStats(u)
				}
			case <-hc.ctx.Done():
				log.Println("Gracefully stopping metrics collector")
				return
			}
		}
	}()
}
