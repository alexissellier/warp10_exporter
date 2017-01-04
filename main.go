package main

import (
	"flag"
	"bufio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
	"strings"
	"regexp"
)

var (
	warpAddr      = flag.String("warp.addr", "127.0.0.1:4242", "Address of sensision")
	listenAddress = flag.String("web.listen-address", ":9121", "Address to listen on for web interface and telemetry.")
	metricPath    = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	debug         = flag.Bool("debug", false, "Debug mode displays what is fetch.")
)

var reg   *regexp.Regexp

func init() {
	reg = regexp.MustCompile(`((?:\.?\w+)+)\{([^}]*)\} (\w*)`)
}

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

func newWarpExporter(addr string) *warp {
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
			"warp_directory_streaming_requests": {
				desc:    prometheus.NewDesc("warp_directory_streaming_requests", "Number of directory request", nil, nil),
				valType: prometheus.CounterValue,
			},
			"warp_directory_metadata_cache_size": {
				desc:    prometheus.NewDesc("warp_directory_metadata_cache_size", "Metadata cache size", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_directory_metadata_cache_hits": {
				desc:    prometheus.NewDesc("warp_directory_metadata_cache_hits", "Number of directory cache hit", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_directory_classes" : {
				desc:    prometheus.NewDesc("warp_directory_classes", "Number of distinct classes managed by Directory", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_directory_hbase_puts": {
				desc:    prometheus.NewDesc("warp_directory_hbase_puts", "Number of directory hbase puts", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_directory_kafka_faileddecrypts": {
				desc:    prometheus.NewDesc("warp_directory_kafka_faileddecrypts", "Fail to descrypt kafka message", nil, nil),
				valType: prometheus.CounterValue,
			},
			"warp_directory_gts": {
				desc:    prometheus.NewDesc("warp_directory_gts", "Number of GTS metadata managed by Directory", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_store_hbase_puts_committed": {
				desc: prometheus.NewDesc("warp_store_hbase_puts_committed", "Number of HBase Puts committed in Store", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_script_requests" : {
				desc: prometheus.NewDesc("warp_script_requests", "Number of script request", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_script_ops" : {
				desc: prometheus.NewDesc("warp_script_ops", "Number of ops executed", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_hbase_client_scanners" : {
				desc: prometheus.NewDesc("warp_hbase_client_scanners", "Number of hbase client scanners", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_store_aborts"  :{
				desc: prometheus.NewDesc("warp_store_aborts", "Number of aborts experienced", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_store_hbase_conn_resets" : {
				desc: prometheus.NewDesc("warp_store_hbase_conn_resets", "Number of times the HBase connection was reset", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_store_kafka_commits_overdue" : {
				desc: prometheus.NewDesc("warp_store_kafka_commits_overdue", "Number of kafka commits overdue", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_store_gtsdecoders" : {
				desc: prometheus.NewDesc("warp_store_gtsdecoders", "Number of gts decoded", nil, nil),
				valType: prometheus.GaugeValue,
			},
			"warp_throttling_rate_perapp_global" : {
				desc: prometheus.NewDesc("warp_throttling_rate_perapp_global", "Throtlling rate", nil, nil),
				valType: prometheus.CounterValue,
			},
			"warp_ingress_update_datapoints_global" : {
				desc: prometheus.NewDesc("warp_ingress_update_datapoints_global", "Datapoints update per second", nil,nil),
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
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			tokens := reg.FindAllStringSubmatch(strings.Trim(line, "\t\r\n"), -1)
			metric_name := strings.Replace(tokens[0][1], ".", "_", -1)
			if val, ok := w.metrics[metric_name]; ok {
				value := parseFloatOrZero(tokens[0][3])
				if *debug {
					log.Printf("Sensision output %s\n", line)
					log.Printf("Metric name %s, value %f\n", metric_name, value)
				}
				ch <- prometheus.MustNewConstMetric(val.desc, val.valType, value)
			}
		}
	} else {
		log.Printf("Cannot fetch sensision output\n")
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
	e := newWarpExporter(*warpAddr)
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
