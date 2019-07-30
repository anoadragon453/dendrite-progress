package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	passingTests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "progress_passing_tests",
		Help: "The number of passing tests",
	})

	totalTests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "progress_total_tests",
		Help: "The total number of tests",
	})
)

// setPassingTests is a function that updates the prometheus metrics count for
// tests that dendrite is passing
func setPassingTests(count int) {
	passingTests.Set(float64(count))
}

// setTotalTests is a function that updates the prometheus metrics count for
// the total number of tests
func setTotalTests(count int) {
	totalTests.Set(float64(count))
}

// serveMetrics is a simple wrapper around promhttp.Handler
func serveMetrics() http.Handler {
	return promhttp.Handler()
}
