package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	core "gx/ipfs/QmToMeLQSNX1mqYV1vbtqfth9HtVgUi3FYJ732tAbxhp8G/go-ipfs/core"
	repo "gx/ipfs/QmToMeLQSNX1mqYV1vbtqfth9HtVgUi3FYJ732tAbxhp8G/go-ipfs/repo"
	cfg "gx/ipfs/QmToMeLQSNX1mqYV1vbtqfth9HtVgUi3FYJ732tAbxhp8G/go-ipfs/repo/config"
	peer "gx/ipfs/QmWUswjn261LSyVxWAEpMVtPdy8zmKBJJfBpG3Qdpa8ZsE/go-libp2p-peer"

	ci "gx/ipfs/QmPGxZ1DP2w45WcogpW1h43BvseXbfke9N91qotpoQcUeS/go-libp2p-crypto"
	ds "gx/ipfs/QmRWDav6mzWseLWeYfVd5fvUKiVe9xNH29YfMF438fG364/go-datastore"
)

func getRepo() (repo.Repo, error) {
	c := cfg.Config{}
	priv, _, err := ci.GenerateKeyPairWithReader(ci.RSA, 1024, rand.Reader)
	if err != nil {
		return nil, err
	}

	privkeyb, err := priv.Bytes()
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	c.Bootstrap = cfg.DefaultBootstrapAddresses
	c.Addresses.Swarm = []string{"/ip4/0.0.0.0/tcp/0"}
	c.Identity.PeerID = pid.Pretty()
	c.Identity.PrivKey = base64.StdEncoding.EncodeToString(privkeyb)

	return &repo.Mock{
		D: ds.NewMapDatastore(),
		C: c,
	}, nil
}

func makeNode(ctx context.Context) (*core.IpfsNode, error) {
	r, err := getRepo()
	if err != nil {
		return nil, err
	}

	return core.NewNode(ctx, &core.BuildCfg{
		Online: true,
		Repo:   r,
	})
}
