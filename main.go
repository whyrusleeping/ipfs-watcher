package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"

	core "github.com/ipfs/go-ipfs/core"
)

var _ = os.DevNull

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
		g.Set(took.Seconds())
	}
}

func doPing(target string) (<-chan time.Duration, error) {
	u := url.URL{
		Host:   "localhost:5001",
		Path:   "/api/v0/ping/" + target,
		Scheme: "http",
	}

	v := url.Values{}
	v.Add("encoding", "json")
	v.Add("stream-channels", "true")

	url := u.String() + "?" + v.Encode()
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	out := make(chan time.Duration)
	go func() {
		defer close(out)
		var t struct {
			Time    time.Duration
			Success bool
		}

		dec := json.NewDecoder(resp.Body)
		for {
			err := dec.Decode(&t)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("ping error: %s\n", err)
				}
				return
			}

			if t.Success {
				out <- t.Time
			}
		}
	}()

	return out, nil
}

func monitorPings(g prometheus.Gauge, peerid string) {
	for {
		lat, err := doPing(peerid)
		if err != nil {
			fmt.Printf("ping %s error: %s\n", peerid, err)
			time.Sleep(time.Second)
			continue
		}

		for val := range lat {
			fmt.Printf("peer %s: %s\n", peerid, val)
			g.Set(val.Seconds())
		}
	}
}

var _ = context.Background
var _ = core.IpnsValidatorTag

/*
func tryResolve(path string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nd, err := core.NewNode(ctx, &core.BuildCfg{Online: true})
	if err != nil {
		panic(err)
	}

	bspid, err := peer.IDB58Decode("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	if err != nil {
		panic(err)
	}

	addr, err := ma.NewMultiaddr("/ip4/104.131.131.82/tcp/4001")
	if err != nil {
		panic(err)
	}

	pi := peer.PeerInfo{
		ID:    bspid,
		Addrs: []ma.Multiaddr{addr},
	}

	err = nd.PeerHost.Connect(ctx, pi)
	if err != nil {
		panic(err)
	}
}
*/

var bootstrappers = map[string]string{
	"neptune": "QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z",
	"pluto":   "QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"uranus":  "QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm",
	"saturn":  "QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"venus":   "QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"earth":   "QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	"mercury": "QmSoLMeWqB7YGVLJN3pNLQpmmEk35v6wYtsMGLzSr5QBU3",
	"jupiter": "QmSoLju6m7xTh3DuokvT3886QRYqxAzb1kShaanJgW36yx",
}

func main() {
	addr := flag.String("addr", ":9999", "address to serve metrics on")
	flag.Parse()

	go monitorHttpEndpoint(vers_g, "https://ipfs.io/version", time.Second*5)
	go monitorHttpEndpoint(blog_g, "http://blog.ipfs.io", time.Second*30)

	for k, v := range bootstrappers {
		ping_g := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:      "ping_" + k,
			Subsystem: "ext_watcher",
			Namespace: "ipfs",
			Help:      "time it takes to ping " + v,
		})

		prometheus.MustRegister(ping_g)
		go monitorPings(ping_g, v)
	}

	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(*addr, nil)
}
