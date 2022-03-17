package healthchecker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

var hc *HealthChecker

func init() {
	ctx := context.Background()
	hc = NewHealthChecker(ctx, time.Duration(1), []string{})
}

func TestFetchStats200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/200")
		assert.Equal(t, r.Method, "GET")

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url := server.URL + "/200"
	hc.fetchStats(url)

	labels := prometheus.Labels{
		"url": url,
	}

	// Validate each metric has been collected for the site
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlConnectTime.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlDNS.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlFirstByte.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlMs.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlStatus.With(labels)))

	// Validate the site is reported healthy for response code 200
	assert.Equal(t, float64(1), testutil.ToFloat64(hc.urlStatus.With(labels)))
}

func TestFetchStats503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/503")
		assert.Equal(t, r.Method, "GET")

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	url := server.URL + "/503"
	hc.fetchStats(url)
	labels := prometheus.Labels{
		"url": url,
	}

	// Validate each metric has been collected for the site
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlConnectTime.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlDNS.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlFirstByte.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlMs.With(labels)))
	assert.Equal(t, 1, testutil.CollectAndCount(hc.urlStatus.With(labels)))

	// Validate the site is reported unhealthy for response code 500
	assert.Equal(t, float64(0), testutil.ToFloat64(hc.urlStatus.With(labels)))
}

func TestUpdateCustomMetrics(t *testing.T) {
	cm := &customMetric{
		url:         "testurl.com",
		status:      1,
		totalMS:     9,
		dnsMS:       2,
		firstbyteMS: 3,
		connectMS:   4,
	}

	hc.updateCustomMetrics(cm)

	labels := prometheus.Labels{
		"url": cm.url,
	}

	assert.Equal(t, float64(cm.status), testutil.ToFloat64(hc.urlStatus.With(labels)))
	assert.Equal(t, float64(cm.totalMS), testutil.ToFloat64(hc.urlMs.With(labels)))
	assert.Equal(t, float64(cm.dnsMS), testutil.ToFloat64(hc.urlDNS.With(labels)))
	assert.Equal(t, float64(cm.firstbyteMS), testutil.ToFloat64(hc.urlFirstByte.With(labels)))
	assert.Equal(t, float64(cm.connectMS), testutil.ToFloat64(hc.urlConnectTime.With(labels)))
}
