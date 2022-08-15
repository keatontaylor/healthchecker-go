package healthchecker

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
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

type ponStatus struct {
	url         string
	voltage     float64
	temperature float64
	txpower     float64
	rxpower     float64
	biascurrent float64
}

type ponStatistics struct {
	url                         string
	ponSentBytes                float64
	ponReceivedBytes            float64
	ponSentPackets              float64
	ponReceivedPackets          float64
	ponSentUnicastPackets       float64
	ponReceievedUnicastPackets  float64
	ponSentMulticastPackets     float64
	ponReceivedMulticastPackets float64
	ponSentBroadcastPackets     float64
	ponReceivedBroadcastPackets float64
	ponFecErrors                float64
	ponHecErrors                float64
	ponPacketsDropped           float64
	ponPausePacketsSent         float64
	ponPausePacketsReceived     float64
}

type HealthChecker struct {
	ctx                         context.Context
	urlStatus                   *prometheus.GaugeVec
	urlMs                       *prometheus.GaugeVec
	urlDNS                      *prometheus.GaugeVec
	urlFirstByte                *prometheus.GaugeVec
	urlConnectTime              *prometheus.GaugeVec
	ponVoltage                  *prometheus.GaugeVec
	ponTemperature              *prometheus.GaugeVec
	ponTxPower                  *prometheus.GaugeVec
	ponRxPower                  *prometheus.GaugeVec
	ponBiasCurrent              *prometheus.GaugeVec
	ponSentBytes                *prometheus.GaugeVec
	ponReceivedBytes            *prometheus.GaugeVec
	ponSentPackets              *prometheus.GaugeVec
	ponReceivedPackets          *prometheus.GaugeVec
	ponSentUnicastPackets       *prometheus.GaugeVec
	ponReceievedUnicastPackets  *prometheus.GaugeVec
	ponSentMulticastPackets     *prometheus.GaugeVec
	ponReceivedMulticastPackets *prometheus.GaugeVec
	ponSentBroadcastPackets     *prometheus.GaugeVec
	ponReceivedBroadcastPackets *prometheus.GaugeVec
	ponFecErrors                *prometheus.GaugeVec
	ponHecErrors                *prometheus.GaugeVec
	ponPacketsDropped           *prometheus.GaugeVec
	ponPausePacketsSent         *prometheus.GaugeVec
	ponPausePacketsReceived     *prometheus.GaugeVec

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
		ponVoltage: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_voltage",
			Help:      "Voltage of the SFP ONT",
		}, []string{"url"}),
		ponTemperature: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_temperature",
			Help:      "temperature of the SFP ONT",
		}, []string{"url"}),
		ponTxPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_tx_power",
			Help:      "SFP ONT TX Power",
		}, []string{"url"}),
		ponRxPower: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_rx_power",
			Help:      "SFP ONT RX Power",
		}, []string{"url"}),
		ponBiasCurrent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_bias_current",
			Help:      "SFP ONT Bias Current",
		}, []string{"url"}),
		ponSentBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_sent_bytes",
			Help:      "Bytes sent over PON network",
		}, []string{"url"}),
		ponReceivedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_receieved_bytes",
			Help:      "Bytes received over PON network",
		}, []string{"url"}),
		ponReceivedPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_receieved_packets",
			Help:      "Packets received over PON network",
		}, []string{"url"}),
		ponSentPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_sent_packets",
			Help:      "Packetes sent over PON network",
		}, []string{"url"}),
		ponSentUnicastPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_sent_unicast_packets",
			Help:      "Unicast packets sent over PON network",
		}, []string{"url"}),
		ponReceievedUnicastPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_received_unicast_packets",
			Help:      "Unicast packets received over PON network",
		}, []string{"url"}),
		ponSentMulticastPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_sent_multicast_packets",
			Help:      "Mutlicast packets sent over PON network",
		}, []string{"url"}),
		ponReceivedMulticastPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_received_multicast_packets",
			Help:      "Mutlicast packets received over PON network",
		}, []string{"url"}),
		ponSentBroadcastPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_sent_broadcast_packets",
			Help:      "Broadcast packets sent over PON network",
		}, []string{"url"}),
		ponReceivedBroadcastPackets: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_received_broadcast_packets",
			Help:      "Broadcast packets Received over PON network",
		}, []string{"url"}),
		ponFecErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_fec_errors",
			Help:      "FEC errors on the pon network",
		}, []string{"url"}),
		ponHecErrors: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_hec_errors",
			Help:      "HEC errors on the pon network",
		}, []string{"url"}),
		ponPacketsDropped: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_packets_dropped",
			Help:      "Packets dropped on the pon network",
		}, []string{"url"}),
		ponPausePacketsReceived: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_pause_packets_received",
			Help:      "Pause packets received on the pon network",
		}, []string{"url"}),
		ponPausePacketsSent: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pon",
			Subsystem: "external",
			Name:      "pon_pause_packets_sent",
			Help:      "Pause packets sent on the pon network",
		}, []string{"url"}),
		healthcheck_invertval: inverval,
		urls:                  urls,
	}
	prometheus.MustRegister(hc.urlStatus,
		hc.urlMs,
		hc.urlDNS,
		hc.urlConnectTime,
		hc.urlFirstByte,
		hc.ponVoltage,
		hc.ponTemperature,
		hc.ponTxPower,
		hc.ponRxPower,
		hc.ponBiasCurrent,
		hc.ponSentBytes,
		hc.ponReceivedBytes,
		hc.ponSentPackets,
		hc.ponReceivedPackets,
		hc.ponSentUnicastPackets,
		hc.ponReceievedUnicastPackets,
		hc.ponSentMulticastPackets,
		hc.ponReceivedMulticastPackets,
		hc.ponSentBroadcastPackets,
		hc.ponReceivedBroadcastPackets,
		hc.ponFecErrors,
		hc.ponHecErrors,
		hc.ponPacketsDropped,
		hc.ponPausePacketsSent,
		hc.ponPausePacketsReceived,
	)
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

func (hc *HealthChecker) updatePonStatus(cm *ponStatus) {
	log.Printf("Updating custom metrics: url: %s, voltage: %.0f, temp: %.0f, rxpower: %.0f, txpower: %.0f, biascurrent: %.0f",
		cm.url,
		cm.voltage,
		cm.temperature,
		cm.rxpower,
		cm.txpower,
		cm.biascurrent,
	)

	hc.ponVoltage.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.voltage)
	hc.ponTemperature.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.temperature)
	hc.ponRxPower.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.rxpower)
	hc.ponTxPower.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.txpower)
	hc.ponBiasCurrent.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.biascurrent)
}

func (hc *HealthChecker) updatePonStatistics(cm *ponStatistics) {
	hc.ponSentBytes.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponSentBytes)
	hc.ponReceivedBytes.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponReceivedBytes)
	hc.ponSentPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponSentPackets)
	hc.ponReceivedPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponReceivedPackets)
	hc.ponSentUnicastPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponSentUnicastPackets)
	hc.ponReceievedUnicastPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponReceievedUnicastPackets)
	hc.ponSentMulticastPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponSentMulticastPackets)
	hc.ponReceivedBroadcastPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponReceivedBroadcastPackets)
	hc.ponSentBroadcastPackets.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponSentBroadcastPackets)
	hc.ponFecErrors.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponFecErrors)
	hc.ponHecErrors.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponHecErrors)
	hc.ponPacketsDropped.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponPacketsDropped)
	hc.ponPausePacketsReceived.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponPausePacketsReceived)
	hc.ponPausePacketsSent.With(prometheus.Labels{
		"url": cm.url,
	}).Set(cm.ponPausePacketsSent)
}

func (hc *HealthChecker) crudeLogin(url_str string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	params := url.Values{}
	params.Add("challenge", ``)
	params.Add("username", `admin`)
	params.Add("password", `admin`)
	params.Add("save", `Login`)
	params.Add("submit-url", `/admin/login.asp`)
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest("POST", "http://192.168.1.1/boaform/admin/formLogin", body)
	if err != nil {
		// handle err
	}
	resp, err := client.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()
}

func (hc *HealthChecker) fetchStats(url string) {

	var start, connect, dns time.Time

	var connectMS, dnsMS, firstbyteMS, totalMS, status, temperature, voltage, txpower, rxpower, biascurrent float64

	var ponSentBytes float64
	var ponReceivedBytes float64
	var ponSentPackets float64
	var ponReceivedPackets float64
	var ponSentUnicastPackets float64
	var ponReceievedUnicastPackets float64
	var ponSentMulticastPackets float64
	var ponReceivedMulticastPackets float64
	var ponSentBroadcastPackets float64
	var ponReceivedBroadcastPackets float64
	var ponFecErrors float64
	var ponHecErrors float64
	var ponPacketsDropped float64
	var ponPausePacketsSent float64
	var ponPausePacketsReceived float64

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
			responseData, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Fatal(err)
			}

			if url == "http://192.168.1.1/status_pon.asp" {
				responseString := string(responseData)
				doc := soup.HTMLParse(responseString)
				links := doc.FindAll("font")
				temperature, _ = strconv.ParseFloat(strings.Split(links[4].Text(), " ")[0], 64)
				voltage, _ = strconv.ParseFloat(strings.Split(links[6].Text(), " ")[0], 64)
				txpower, _ = strconv.ParseFloat(strings.Split(links[8].Text(), " ")[0], 64)
				rxpower, _ = strconv.ParseFloat(strings.Split(links[10].Text(), " ")[0], 64)
				biascurrent, _ = strconv.ParseFloat(strings.Split(links[12].Text(), " ")[0], 64)
				hc.updatePonStatus(
					&ponStatus{
						url:         url,
						temperature: temperature,
						voltage:     voltage,
						txpower:     txpower,
						rxpower:     rxpower,
						biascurrent: biascurrent,
					},
				)
			} else if url == "http://192.168.1.1/admin/pon-stats.asp" {
				responseString := string(responseData)
				doc := soup.HTMLParse(responseString)
				links := doc.FindAll("td")

				ponSentBytes, _ = strconv.ParseFloat(strings.Split(links[1].Text(), " ")[0], 64)
				ponReceivedBytes, _ = strconv.ParseFloat(strings.Split(links[2].Text(), " ")[0], 64)
				ponSentPackets, _ = strconv.ParseFloat(strings.Split(links[3].Text(), " ")[0], 64)
				ponReceivedPackets, _ = strconv.ParseFloat(strings.Split(links[4].Text(), " ")[0], 64)
				ponSentUnicastPackets, _ = strconv.ParseFloat(strings.Split(links[5].Text(), " ")[0], 64)
				ponReceievedUnicastPackets, _ = strconv.ParseFloat(strings.Split(links[6].Text(), " ")[0], 64)
				ponSentMulticastPackets, _ = strconv.ParseFloat(strings.Split(links[7].Text(), " ")[0], 64)
				ponReceivedMulticastPackets, _ = strconv.ParseFloat(strings.Split(links[8].Text(), " ")[0], 64)
				ponSentBroadcastPackets, _ = strconv.ParseFloat(strings.Split(links[9].Text(), " ")[0], 64)
				ponReceivedBroadcastPackets, _ = strconv.ParseFloat(strings.Split(links[10].Text(), " ")[0], 64)
				ponFecErrors, _ = strconv.ParseFloat(strings.Split(links[11].Text(), " ")[0], 64)
				ponHecErrors, _ = strconv.ParseFloat(strings.Split(links[12].Text(), " ")[0], 64)
				ponPacketsDropped, _ = strconv.ParseFloat(strings.Split(links[13].Text(), " ")[0], 64)
				ponPausePacketsSent, _ = strconv.ParseFloat(strings.Split(links[14].Text(), " ")[0], 64)
				ponPausePacketsReceived, _ = strconv.ParseFloat(strings.Split(links[15].Text(), " ")[0], 64)
				hc.updatePonStatistics(
					&ponStatistics{
						url:                         url,
						ponSentBytes:                ponSentBytes,
						ponReceivedBytes:            ponReceivedBytes,
						ponSentPackets:              ponSentPackets,
						ponReceivedPackets:          ponReceivedPackets,
						ponSentUnicastPackets:       ponSentUnicastPackets,
						ponReceievedUnicastPackets:  ponReceievedUnicastPackets,
						ponSentMulticastPackets:     ponSentMulticastPackets,
						ponReceivedMulticastPackets: ponReceivedMulticastPackets,
						ponSentBroadcastPackets:     ponSentBroadcastPackets,
						ponReceivedBroadcastPackets: ponReceivedBroadcastPackets,
						ponFecErrors:                ponFecErrors,
						ponHecErrors:                ponHecErrors,
						ponPacketsDropped:           ponPacketsDropped,
						ponPausePacketsSent:         ponPausePacketsSent,
						ponPausePacketsReceived:     ponPausePacketsReceived,
					},
				)
			}

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
					hc.crudeLogin(u)
					hc.fetchStats(u)
				}
			case <-hc.ctx.Done():
				log.Println("Gracefully stopping metrics collector")
				return
			}
		}
	}()
}
