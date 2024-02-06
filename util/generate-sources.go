package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multicodec"

	"github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	chunker "github.com/ipfs/boxo/chunker"
	offline "github.com/ipfs/boxo/exchange/offline"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/boxo/ipld/unixfs/importer/balanced"
	uih "github.com/ipfs/boxo/ipld/unixfs/importer/helpers"
)

const exampleBinaryName = "m"

type Source struct {
	Name string
	Cid  string
	Size int64
}

func main() {
	flag.Parse()

	entries, err := os.ReadDir("./sources")

	if err != nil {
		log.Fatal(err)
	}

	builder := new(strings.Builder)

	for i, e := range entries {
		fmt.Println(e.Name())

		f, _ := e.Info()
		if !f.IsDir() {
			c, err := getCidFromFile("./sources/" + f.Name())

			if err != nil {
				panic(err)
			}

			if err != nil {
				panic(err)
			}

			bytes, _ := json.Marshal(Source{Name: f.Name(), Size: f.Size(), Cid: c.String()})

			if i > 0 {
				builder.WriteByte('\n')
			}

			builder.Write(bytes)
		}
	}
	os.WriteFile("sources.json", []byte(builder.String()), 0644)

	log.Println("Writing sources to file sources.json")

}

func getCidFromFile(filename string) (cid.Cid, error) {
	fileBytes, err := os.ReadFile(filename)

	if err != nil {
		return cid.Undef, err
	}
	fileReader := bytes.NewReader(fileBytes)

	ds := dsync.MutexWrap(&datastore.NullDatastore{})
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs)

	// why is there a block service and a block store????

	bsrv := blockservice.New(bs, offline.Exchange(bs))
	dsrv := merkledag.NewDAGService(bsrv)

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
	ufsBuilder, err := ufsImportParams.New(chunker.NewSizeSplitter(fileReader, chunker.DefaultBlockSize))
	if err != nil {
		return cid.Undef, err
	}
	nd, err := balanced.Layout(ufsBuilder)
	if err != nil {
		return cid.Undef, err
	}

	return nd.Cid(), nil
}
