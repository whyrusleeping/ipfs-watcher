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
)

func init() {
	prometheus.MustRegister(vers_g)
}

func main() {
	addr := flag.String("addr", ":9999", "address to serve metrics on")
	flag.Parse()

	go func() {
		for range time.Tick(time.Second * 5) {
			before := time.Now()
			resp, err := http.Get("https://ipfs.io/version")
			if err != nil {
				fmt.Println(err)
				continue
			}
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
			took := time.Now().Sub(before)
			vers_g.Add(took.Seconds())
		}
	}()

	http.Handle("/metrics", prometheus.Handler())

	http.ListenAndServe(*addr, nil)
}
