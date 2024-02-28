package ipfs

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	mrand "math/rand"

	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/ipfs/go-cid"

	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multicodec"

	"github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	chunker "github.com/ipfs/boxo/chunker"

	// offline "github.com/ipfs/boxo/exchange/offline"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/boxo/ipld/unixfs/importer/balanced"
	uih "github.com/ipfs/boxo/ipld/unixfs/importer/helpers"

	// blocks "github.com/ipfs/go-block-format"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"

	bsclient "github.com/ipfs/boxo/bitswap/client"
	bsnet "github.com/ipfs/boxo/bitswap/network"
	bsserver "github.com/ipfs/boxo/bitswap/server"
)

const DEFAULT_STORAGE_BYTES = 20 * 1024 * 1024
const DEFAULT_BLOCK_SIZE = 256 * 1024

// we will use the default chunk size, for the fixed chunk size. It's 256Kib
type Source struct {
	Name string
	Size int64
	Cid  string // an id to the ROOT
}

// TODO: give this a better name
type Status int8

const (
	DOWN Status = iota
	CONNECTING_TO_PEERS
	GETTING_METADATA
	ADJUSTING_WANTED_BLOCKS
	DOWNLOADING_BLOCKS
	SEEDING_BLOCKS
)

type Instance struct {
	CapacityInBytes   uint
	Bservice          blockservice.BlockService
	Bstore            blockstore.Blockstore
	Bsnetwork         bsnet.BitSwapNetwork
	Bsserver          *bsserver.Server
	Bsclient          *bsclient.Client
	Host              host.Host
	PeersBacklog      []string
	PeersBacklogMutex sync.Mutex
	Sources           []Source
	StorageSize       int
	Status            Status

	LeafBlocks     map[string][]cid.Cid // leaves in the IPFS tree, which means blocks that store raw data
	BlocksToSeed   map[string][]int
	BlocksSeeding  map[string][]int
	BlockMapsMutex sync.Mutex
}

func blocksInSize(size int64) int64 {
	blockCount := int64(size) / (DEFAULT_BLOCK_SIZE)
	// blocks HAVE to fit the data so if they don't divide nicelly, we need an extra chunk to fit the data
	if size < DEFAULT_BLOCK_SIZE && size > 0 {
		blockCount = 1
	} else if size%int64(DEFAULT_BLOCK_SIZE) != 0 && size > int64(chunker.DefaultBlockSize) {
		blockCount += 1
	}

	return blockCount
}

func (s *Source) BlockCount() int64 {
	return blocksInSize(s.Size)
}

func (s *Source) BlockSize(index int) (int64, error) {
	blocksCount := blocksInSize(s.Size)
	if int64(index) > blocksCount-1 {
		return 0, errors.New("Index out of range")
	} else {
		if index == int(blocksCount)-1 {
			// size is remainder
			return (s.Size) - ((blocksCount - 1) * (DEFAULT_BLOCK_SIZE)), nil
		} else {
			return DEFAULT_BLOCK_SIZE, nil
		}
	}
}

func HostToString(h host.Host) string {
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", h.ID().String()))

	addr := h.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func makeHost(listenPort int, randseed int64) (host.Host, error) {
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", os.Getenv("XNODE_IP"), listenPort)),
		libp2p.Identity(priv),
	}

	return libp2p.New(opts...)
}

// This runs the seed server, which will read all sources in the sources directory and seed them forever
func (inst *Instance) runSeedServer(ctx context.Context) {
	entries, err := os.ReadDir("sources")

	if err != nil {
		log.Fatal(err)
	}

	inst.Status = GETTING_METADATA

	for _, e := range entries {

		f, _ := e.Info()

		if !f.IsDir() {
			fmt.Println(e.Name())

			inst.BlocksToSeed[f.Name()] = make([]int, blocksInSize(int64(f.Size())))
			for i := 0; i < int(blocksInSize(int64(f.Size()))); i++ {
				inst.BlocksToSeed[f.Name()][i] = i
			}

			c, size, err := inst.seedFile("./sources/" + f.Name())

			inst.BlocksSeeding[f.Name()] = make([]int, blocksInSize(int64(size)))
			for i := 0; i < int(blocksInSize(int64(size))); i++ {
				inst.BlocksSeeding[f.Name()][i] = i
			}

			fmt.Println("Now seeding", c, "", size/1024, "KB")

			if err != nil {
				panic(err)
			}
		}
	}

	inst.Status = SEEDING_BLOCKS

	n := bsnet.NewFromIpfsHost(inst.Host, routinghelpers.Null{})
	inst.Bsserver = bsserver.New(ctx, inst.Bsnetwork, inst.Bstore)

	n.Start(inst.Bsserver)
}

// Reads a file and seeds it on IPFS
func (inst *Instance) seedFile(filename string) (cid.Cid, uint64, error) {

	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return cid.Undef, 0, err
	}
	fileReader := bytes.NewReader(fileBytes)

	// NOTE Might have to change this... it used to use an offline blockservice which could be the correct approach here
	dsrv := merkledag.NewDAGService(inst.Bservice)

	ufsImportParams := uih.DagBuilderParams{
		Maxlinks:  uih.DefaultLinksPerBlock,
		RawLeaves: true,
		CidBuilder: cid.V1Builder{
			Codec:    uint64(multicodec.DagPb),
			MhType:   uint64(multicodec.Sha2_256),
			MhLength: -1,
		},
		Dagserv: dsrv,
		NoCopy:  false,
	}
	ufsBuilder, err := ufsImportParams.New(chunker.NewSizeSplitter(fileReader, DEFAULT_BLOCK_SIZE)) // Split the file up into fixed sized 256KiB blocks
	if err != nil {
		return cid.Undef, 0, err
	}
	nd, err := balanced.Layout(ufsBuilder) // Arrange the graph with a balanced layout
	if err != nil {
		return cid.Undef, 0, err
	}

	size, _ := nd.Size()
	return nd.Cid(), size, nil
}

func NewInstance() *Instance {
	// Max storage
	h, err := makeHost(0, mrand.Int63())
	if err != nil {
		panic(err)
	}

	inst := &Instance{
		Bservice:          nil,
		Bstore:            nil,
		Host:              h,
		PeersBacklog:      make([]string, 0),
		PeersBacklogMutex: sync.Mutex{},
	}

	{ // Open up sources.json and work some stuff out
		bytes, err := os.ReadFile("sources.json")
		if err != nil {
			panic(err)
		}

		lines := strings.Split(string(bytes), "\n")
		inst.Sources = make([]Source, len(lines))
		inst.BlocksToSeed = make(map[string][]int, len(inst.Sources))
		inst.BlocksSeeding = make(map[string][]int, len(inst.Sources))
		inst.LeafBlocks = make(map[string][]cid.Cid, len(inst.Sources))
		inst.StorageSize = DEFAULT_STORAGE_BYTES

		for i := range lines {
			err = json.Unmarshal([]byte(lines[i]), &inst.Sources[i])
			if err != nil {
				panic(err)
			}

			inst.BlocksToSeed[inst.Sources[i].Name] = make([]int, 0)
			inst.BlocksSeeding[inst.Sources[i].Name] = make([]int, 0)
			inst.LeafBlocks[inst.Sources[i].Name] = make([]cid.Cid, inst.Sources[i].BlockCount())
		}
	}

	{
		inst.Bsnetwork = bsnet.NewFromIpfsHost(inst.Host, routinghelpers.Null{})

		// TODO: move to a physical data store
		inst.Bstore = blockstore.NewBlockstore(dsync.MutexWrap(datastore.NewMapDatastore()))
		inst.Bstore = blockstore.NewIdStore(inst.Bstore)
	}

	return inst
}

func (inst *Instance) Start(ctx context.Context, httpPeers []string) {

	// NOTE(Tom): these interfaces do the actual storage, it's currently configured to do everything in RAM
	inst.Bsclient = bsclient.New(ctx, inst.Bsnetwork, inst.Bstore)
	inst.Bsserver = bsserver.New(ctx, inst.Bsnetwork, inst.Bstore)

	inst.Bservice = blockservice.New(inst.Bstore, inst.Bsclient)

	if os.Getenv("XNODE_NAME") == "Xnode-1" {
		// Have to run this on a different thread, otherwise this will block instance.Start(...) and never cancel the context
		go func() {
			inst.runSeedServer(ctx)

			<-ctx.Done()
		}()

		return
	}

	// Start sharing and caring!!
	inst.Bsnetwork.Start(inst.Bsclient, inst.Bsserver)

	go func() {
		log.Println("Getting peers")

		// TODO: Now we use libp2p method which is more efficient, keeping this for sake of clarity
		inst.Status = CONNECTING_TO_PEERS
		for iterations := 0; iterations < 3; iterations++ { // Get peers
			time.Sleep(time.Millisecond * 1000)

			// Note(Tom): This is not ideal.
			// What should happen in a protocol like this is nodes find each other completely randomly.
			// For this demo the nodes are aware of some immediate peers' HTTP ports.

			connectFromHostString := func(str string) {
				log.Println("Found address:", str)
				maddr, _ := multiaddr.NewMultiaddr(str)
				info, err := peer.AddrInfoFromP2pAddr(maddr)
				if err != nil {
					log.Println("Error getting info", err.Error())
				}

				// I'm in 😎
				err = inst.Host.Connect(ctx, *info)
				if err == nil {
					log.Println("Connected to address")
				} else {
					log.Println("Error connecting:", err.Error())
				}
			}

			log.Println("Trying to connect to peers...")

			for _, p := range httpPeers {
				// call endpoint that returns the ipv4 address
				fmt.Println(p)

				resp, err := http.Get("http://" + p + "/ipfsidentity")

				if err == nil {
					buf := new(strings.Builder)
					io.Copy(buf, resp.Body)

					connectFromHostString(buf.String())
				} else {
					log.Println("Couldn't connect to server:", err.Error())
				}
			}
		}

		{
			inst.Status = GETTING_METADATA
			dserv := merkledag.NewReadOnlyDagService(merkledag.NewSession(ctx, merkledag.NewDAGService(inst.Bservice)))

			for _, source := range inst.Sources {
				log.Println("Looking at source:", source)
				cidStr := source.Cid

				// This blocks until we get info
				// XXX: make this non-blocking, add a timeout or something...
				node, err := dserv.Get(ctx, cid.MustParse(cidStr))

				if err != nil {
					log.Println(err)
					panic("")
				}

				leaves := inst.LeafBlocks[source.Name]
				size, _ := node.Size()
				blockCount := int(blocksInSize(int64(size)))

				log.Println("chunkcount", blockCount)

				// XXX: maybe use a stack instead of recursion...
				// also use GetMany instead of Get
				leafCount := 0
				var getleaves func(c cid.Cid)
				getleaves = func(c cid.Cid) {
					node, _ := dserv.Get(ctx, c)

					links := node.Links()

					if len(links) > 0 {
						cids := make([]cid.Cid, len(links))
						for i := range links {
							cids[i] = links[i].Cid
						}

						if cids[0].Type() == cid.Raw {
							for _, cc := range cids {
								leaves[leafCount] = cc
								leafCount++
							}
							return
						} else {
							for i := range cids {
								getleaves(cids[i])
							}
							return
						}
					} else if c.Type() == cid.Raw {
						// XXX: This might not be threadsafe, just in case you  want to multithread :)
						leaves[leafCount] = c
						leafCount++
					} else {
						// crappy assertion
						log.Panicln("This should be impossible")
					}
				}

				getleaves(node.Cid())
			}
		}

		randomizeBlocks := func() { // Which blocks should I seed (as a node (as a millionaire))
			// Amazing distribution algorithm™️
			inst.Status = ADJUSTING_WANTED_BLOCKS
			log.Println("Working out which blocks to seed")

			unstoredBytesAcrossAllSources := int64(0)
			for _, s := range inst.Sources {
				unstoredBytesAcrossAllSources += s.Size
			}

			totalBlockCount := int64(0)
			for _, s := range inst.Sources {
				totalBlockCount += s.BlockCount()
			}

			freeStorage := int64(inst.StorageSize)

			log.Println("Size of all sources", unstoredBytesAcrossAllSources)
			log.Println("Free storage avalaible", unstoredBytesAcrossAllSources)

			totalIterations := 0

			newBlocksToSeed := make(map[string][]int, len(inst.Sources))

			// TODO: Add a bias to keep blocks it has over blocks it doesn't have
			for totalIterations < 10000 && unstoredBytesAcrossAllSources > 0 && freeStorage > 0 {
				totalIterations++

				blockIndex := mrand.Intn(int(totalBlockCount))

				// find source that stores index
				var source Source
				for _, s := range inst.Sources {
					if blockIndex-int(s.BlockCount()) < 0 {
						// this should work?
						source = s
						break
					} else {
						blockIndex -= int(s.BlockCount())
					}
				}

				blocksInSource := int(source.BlockCount())

				blocksForSource := newBlocksToSeed[source.Name]

				if blocksInSource <= len(blocksForSource) {
					// log.Println("Failed here", totalIterations)
					// reroll
					continue
				} else {
					// make sure block index isn't already stored

					// XXX: This is a naive implementation, this should be optimized to not take into account blocks we already store

					// try to find a block that we don't have
					potentialBlockIndex := mrand.Intn(blocksInSource)
					uniqueBlockIndex := true

					// is this index in the thing?
					for _, i := range blocksForSource {
						if i == potentialBlockIndex {
							uniqueBlockIndex = false
						}
					}

					if uniqueBlockIndex {
						size := 0
						if potentialBlockIndex == blocksInSource-1 {
							// the size is the remainder since the last block is only as big as the leftover data
							size = int(source.Size) - ((blocksInSource - 1) * int(DEFAULT_BLOCK_SIZE))
						} else {
							// the size is just the chunk size!!
							size = int(DEFAULT_BLOCK_SIZE)
						}

						if freeStorage-int64(size) <= 0 {
							// sadly this is too big
							// log.Println("Not enough storage")
							continue
						} else {
							// Don't do this until we know we have enough space
							// add to registered list of blocks tracked
							newBlocksToSeed[source.Name] = append(newBlocksToSeed[source.Name], potentialBlockIndex)
							// log.Println("blocks increased")

							// decrease max storage
							// decrease space available
							freeStorage -= int64(size)
							unstoredBytesAcrossAllSources -= int64(size)
							// fmt.Println("free storage", freeStorage)
							// fmt.Println("unallocated storage", unstoredBytesAcrossAllSources)
						}
					} else {
						// log.Println("Block index not unique")
						continue
					}
				}
			}

			// copy new blocks to seed to existing map
			log.Println("Waiting for block mutex")
			inst.BlockMapsMutex.Lock()
			inst.BlocksToSeed = newBlocksToSeed
			inst.BlockMapsMutex.Unlock()
		}

		randomizeBlocks()
		prevSize := inst.StorageSize

		for {
			t := time.NewTicker(500 * time.Millisecond)

			select {
			case <-t.C:

				{ // block adjustment
					if prevSize != inst.StorageSize {
						randomizeBlocks()
					}

					prevSize = inst.StorageSize
				}

				{ // download data to be seeded

					// TODO: check if BlocksToSeed and BlocksSeeding are different before running all this code
					inst.Status = DOWNLOADING_BLOCKS

					// NOTE(Tom): This might be inefficient. Haven't tested though
					dserv := merkledag.NewReadOnlyDagService(merkledag.NewSession(ctx, merkledag.NewDAGService(inst.Bservice)))

					// get the leaves' metadata!!
					for _, source := range inst.Sources {
						inst.BlockMapsMutex.Lock()

						// clear previous array of seeded blocks
						inst.BlocksSeeding[source.Name] = make([]int, 0)
						seedingBlocks := inst.BlocksToSeed[source.Name]

						fmt.Println("Blocks for this source:", len(seedingBlocks), seedingBlocks)

						leaves := inst.LeafBlocks[source.Name]

						var wg sync.WaitGroup
						var mu sync.Mutex

						for i := 0; i < int(source.BlockCount()); i++ {
							isInSeedList := false

							for _, j := range seedingBlocks {
								if i == j && i < len(leaves) {
									if leaves[i].Defined() {
										wg.Add(1)
										go func(index int) {
											ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
											defer cancel()

											_, err := dserv.Get(ctx, leaves[index])
											if err == nil {
												mu.Lock()
												defer mu.Unlock()
												inst.BlocksSeeding[source.Name] = append(inst.BlocksSeeding[source.Name], index)
											}
											wg.Done()
										}(i)

										isInSeedList = true
									} else {
										log.Println("Fail!", i, leaves[i])
									}
								}
							}

							if !isInSeedList {
								inst.Bservice.DeleteBlock(ctx, leaves[i])
							}
						}

						wg.Wait()
						inst.BlockMapsMutex.Unlock()
					}

					inst.Status = SEEDING_BLOCKS
				}

				continue
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (inst *Instance) Stop() {
	// TODO: consider implementing this...
}
