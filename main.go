package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	peer "gx/ipfs/QmQGwpJy9P4yXZySmqkZEXCmbBpJUb8xntCv8Ca4taZwDC/go-libp2p-peer"
	ma "gx/ipfs/QmYzDkkgAEmrcNzFCiYo6L1dTX4EAG1gZkbtdbd9trL4vd/go-multiaddr"
	pstore "gx/ipfs/QmZ62t46e9p7vMYqCmptwQC1RhRv5cpQ5cwoqYspedaXyq/go-libp2p-peerstore"

	core "github.com/ipfs/go-ipfs/core"
	cunix "github.com/ipfs/go-ipfs/core/coreunix"
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

	ipns_g = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "ipns_get",
		Subsystem: "ext_watcher",
		Namespace: "ipfs",
		Help:      "time it takes to get ipfs.io/ipns/<ID>",
		ConstLabels: prometheus.Labels{
			"host": "mars",
		},
	})

	newh_g = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "new_has_res",
		Subsystem: "ext_watcher",
		Namespace: "ipfs",
		Help:      "time it takes a new node to resolve their data on the gateway",
	})
)

func init() {
	prometheus.MustRegister(vers_g)
	prometheus.MustRegister(blog_g)
	prometheus.MustRegister(ipns_g)
}

func monitorHttpEndpoint(g prometheus.Gauge, url string, interval time.Duration) {
	for range time.Tick(interval) {
		took, err := timeHttpFetch(url)
		if err != nil {
			fmt.Println(err)
			continue
		}
		g.Set(took.Seconds())
	}
}

func timeHttpFetch(url string) (time.Duration, error) {
	before := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	took := time.Now().Sub(before)
	return took, nil
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
			g.Set(val.Seconds())
		}
	}
}

var _ = context.Background
var _ = core.IpnsValidatorTag

func monitorNewHashResolution(g prometheus.Gauge, period time.Duration) {
	for range time.Tick(period) {
		err := tryResolve(g)
		if err != nil {
			fmt.Println("new hash resolve err: ", err)
		}
	}
}

func tryResolve(g prometheus.Gauge) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nd, err := makeNode(ctx)
	if err != nil {
		return err
	}

	err = nd.Bootstrap(core.DefaultBootstrapConfig)
	if err != nil {
		return err
	}

	bspid, err := peer.IDB58Decode("QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ")
	if err != nil {
		return err
	}

	addr, err := ma.NewMultiaddr("/ip4/104.131.131.82/tcp/4001")
	if err != nil {
		return err
	}

	pi := pstore.PeerInfo{
		ID:    bspid,
		Addrs: []ma.Multiaddr{addr},
	}

	err = nd.PeerHost.Connect(ctx, pi)
	if err != nil {
		return err
	}

	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	h, err := cunix.Add(nd, io.LimitReader(rr, 2048))
	if err != nil {
		return err
	}

	dur, err := timeHttpFetch("https://v04x.ipfs.io/ipfs/" + h)
	if err != nil {
		ipns_g.Set(-1)
		return err
	}

	g.Set(dur.Seconds())
	return nil
}

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
	go monitorHttpEndpoint(ipns_g, "https://ipfs.io/ipns/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ", time.Minute)

	go monitorNewHashResolution(newh_g, time.Second*30)

	for k, v := range bootstrappers {
		ping_g := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:      "ping",
			Subsystem: "ext_watcher",
			Namespace: "ipfs",
			Help:      "time it takes to ping a host",
			ConstLabels: prometheus.Labels{
				"host": k,
			},
		})

		prometheus.MustRegister(ping_g)
		go monitorPings(ping_g, v)
	}

	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(*addr, nil)
}
