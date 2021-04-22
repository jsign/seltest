package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/ipfs/go-cid"
	graphsync "github.com/ipfs/go-graphsync/impl"
	"github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/go-graphsync/storeutil"
	dstest "github.com/ipfs/go-merkledag/test"
	"github.com/ipfs/go-unixfsnode"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestRetrieval(t *testing.T) {
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/4001"
	ctx := context.Background()
	localNode, err := libp2p.New(ctx)
	require.NoError(t, err)

	id, err := peer.Decode("12D3KooWRk8482WjWye4YKf4uowcgZNoyWQgWMQAue5q7FfUzHC1") // Change with PeerID of your go-ipfs node
	require.NoError(t, err)
	ipfsMultiaddr, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)

	err = localNode.Connect(ctx, peer.AddrInfo{
		ID:    id,
		Addrs: []multiaddr.Multiaddr{ipfsMultiaddr},
	})
	require.NoError(t, err)

	network := network.NewFromLibp2pHost(localNode)
	bserv := dstest.Bserv()
	loader := storeutil.LinkSystemForBlockstore(bserv.Blockstore())
	loader.NodeReifier = unixfsnode.Reify

	exchange := graphsync.New(ctx, network, loader)
	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)

	all := ssb.ExploreRecursive(selector.RecursionLimitDepth(10), ssb.ExploreAll(ssb.ExploreRecursiveEdge()))
	layer1 := ssb.ExploreFields(func(e builder.ExploreFieldsSpecBuilder) {
		e.Insert("one.txt", all)
	})
	layer0 := ssb.ExploreFields(func(e builder.ExploreFieldsSpecBuilder) {
		e.Insert("one", layer1)
	})
	z, err := cid.Decode("bafybeigjp5wnoz4xi3dqzk6zu7pegxqzlp4rlpcu7baneuj3figisrhc2e")
	require.NoError(t, err)
	clink := cidlink.Link{Cid: z}
	ch1, errch := exchange.Request(ctx, id, clink, layer0.Node()) // change to all.Node() and things appear.

	for n := range ch1 {
		fmt.Printf("nodes chan: %v\n", n)
	}
	err, ok := <-errch
	if ok {
		fmt.Printf("error: %v\n", err)
	}

	onetxtCid, _ := cid.Decode("bafkreif5shm27nwnnw2frztc5pmh5m5azhoxfwqhbfimr5cyd53254tzgu") // cid of one.txt
	has, err := bserv.Blockstore().Has(onetxtCid)
	require.NoError(t, err)
	fmt.Printf("Has?: %v\n", has)

	_ = all
	_ = layer1
	_ = layer0
}
