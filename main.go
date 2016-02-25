package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	vers_g = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "version_get",
		Subsystem: "ext_watcher",
		Namespace: "ipfs",
		Help:      "time it takes to get /version on gateways",
	})

	blog_g = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "blog_get",
		Subsystem: "ext_watcher",
		Namespace: "ipfs",
		Help:      "time it takes to get blog.ipfs.io",
	})
)

func init() {
	prometheus.MustRegister(vers_g)
	prometheus.MustRegister(blog_g)
}

func monitorHttpEndpoint(g prometheus.Gauge, url string, interval time.Duration) {
	for range time.Tick(interval) {
		before := time.Now()
		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			continue
		}
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		took := time.Now().Sub(before)
		g.Add(took.Seconds())
	}
}

func main() {
	addr := flag.String("addr", ":9999", "address to serve metrics on")
	flag.Parse()

	go monitorHttpEndpoint(vers_g, "https://ipfs.io/version", time.Second*5)
	go monitorHttpEndpoint(blog_g, "https://blog.ipfs.io", time.Second*30)

	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(*addr, nil)
}
