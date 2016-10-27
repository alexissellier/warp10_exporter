package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var (
	warpAddr      = flag.String("warp.addr", "127.0.0.1:4242", "Address of sensision")
	listenAddress = flag.String("web.listen-address", ":9121", "Address to listen on for web interface and telemetry.")
	metricPath    = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	debug         = flag.Bool("debug", false, "Debug mode displays what is fetch.")
)

type warp struct {
	warpAddr string
	metrics  map[string]warpMetric
}

type warpMetric struct {
	desc    *prometheus.Desc
	valType prometheus.ValueType
}

func parseFloatOrZero(s string) float64 {
	res, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("Cannot parse %s\n", s)
		return 0.0
	}
	return res
}

func NewWarpExporter(addr string) *warp {
	e := warp{
		warpAddr: addr,
		metrics: map[string]warpMetric{
			"warp_ingress_update_requests": {
				desc:    prometheus.NewDesc("warp_ingress_update_requests", "Number of request", nil, nil),
				valType: prometheus.CounterValue,
			},
			"warp_ingress_update_parseerrors": {
				desc:    prometheus.NewDesc("warp_ingress_update_parseerrors", "Number of parse error", nil, nil),
				valType: prometheus.CounterValue,
			},
			"warp_ingress_update_invalidtoken": {
				desc:    prometheus.NewDesc("warp_ingress_update_invalidtoken", "Number of invalid token", nil, nil),
				valType: prometheus.CounterValue,
			},
		},
	}
	return &e
}

func (w *warp) scrapeSensisionMetrics(ch chan<- prometheus.Metric) {
	resp, err := http.Get("http://" + w.warpAddr + "/metrics")
	if err == nil {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		for _, line := range strings.Split(string(body), "\n") {
			tokens := strings.Split(line, " ")
			if len(tokens) == 3 {
				metric := strings.Replace(tokens[1], ".", "_", -1)
				metric = metric[:strings.IndexRune(metric, '{')]
				if val, ok := w.metrics[metric]; ok {
					value := parseFloatOrZero(tokens[2])
					if *debug {
						log.Printf("Sensision output %s\n", line)
						log.Printf("Metric name %s, value %f\n", metric, value)
					}
					ch <- prometheus.MustNewConstMetric(val.desc, val.valType, value)
				}
			}
		}
	}
}

func (w *warp) Describe(ch chan<- *prometheus.Desc) {
	for _, i := range w.metrics {
		ch <- i.desc
	}
}

func (w *warp) Collect(ch chan<- prometheus.Metric) {
	if *debug {
		log.Printf("Start collecting")
	}
	w.scrapeSensisionMetrics(ch)
	if *debug {
		log.Printf("Stop collecting")
	}
}

func main() {
	flag.Parse()
	e := NewWarpExporter(*warpAddr)
	prometheus.MustRegister(e)
	http.Handle(*metricPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		       <head><title>Warp10 exporter</title></head>
		       <body>
		       <h1>Warp10 exporter</h1>
		       <p><a href='` + *metricPath + `'>Metrics</a></p>
		       </body>
		       </html>`))
	})
	log.Printf("providing metrics at %s%s", *listenAddress, *metricPath)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
