# Resource Aggregation Layer Proof of Concept

A proof-of-concept and testbed for future RAL project.

## Project Layout

- `main.go`: The main function. Entrance of the program.
- `internal` directory: Internal libraries used by this project.
  - `api` directory: HTTP RESTful API + HTMX related things.
  - `ipfs` directory: The bulk of the block management logic.
  - `gossip` directory: Membership management and eventual consistent data sharing.
  - `instance` directory: Peer instance aka top-level instance.
  - `model` directory: Data types used by this project.
  - `index.html` file: The UI code.

## How to run


1. Fill the sources directory with files you want to seed
1. run `go run util/generate-sources.go`
1. run `./build.sh` (calls go build and docker build) or just `CGO_ENABLED=0 GOOS=linux go build -o resource-aggregation-poc && docker build -t xnode:latest .`
1. run `docker compose up`
1. open up index.html on a browser to watch your nodes in action, or go to `http://192.168.1.111:9080/dashboard`


---


## Getting up to speed
The codebase is small for now. 
If you just follow the flow of the program from main.go, 
you should get an idea of how everything is ran.

### IPFS
The code for file sharing, splitting and all that.

#### IPFS Rundown
Because it deals with the IPFS internals I'll give you a rough explanation here:
 
IPFS stores it's data as CIDs (Content IDs).
Basically, these CIDs will either point to raw data (literally a hash of an array of bytes), or to more metadata (an array of CIDs).
This is done to turn large amounts of data into graphs that can be easily traversed, without needing to download all every block's metadata.
So the root node stores an array of CIDs which are the children, and they store some more children which eventually store CIDs pointing to data.

Blocks are the CID + the "data" of an object.
Again, when I say data it includes things like child CIds or in some cases file types.
"Raw blocks" or "Leaf blocks" are blocks that map to actual data.

Nodes are a kind of data that CIDs can store which is essentially a CID + Size of underlying data + Links (meaning children).
To traverse this we have to use a DAG utility and to fetch it we have to use a DAG service.
It's only a few lines of code and relatively easy to parse, look for a function called `recurseGetleaves` for reference.

In the IPFS API, we are able to "Get" blocks which will either fetch them from local storage if present, or look them up on our libp2p network if absent.
ny blocks you Get are saved to your local "blockstore" (basically a `map[cid.Cid][]bytes`), which in our configuration means you will also share it across the network.
Getting blocks in this way means that .

These things communicate via the libp2p library which abstracts all connections. 
Essentially these interfaces (bitswap, blockstore, etc) have access to a libp2p `host.Host` which stores and manages connection state.
All we do is call `host.Connect(...)` to some known peers to test the connection.

#### IPFS in this project

We set up XNode1 to be a file seeder.
Instead of running the same logic as the other nodes; 
it calls `runSeedServer(...)` which opens up all the files in the sources folder, turns them into CIDs and seeds all the chunks.
This simulates someone passing the data into the network.

The other nodes are made aware of these sources through a json file.
This sources.json file lists the sources we'll be fetching.
It lists their names, sizes, and CIDs.
This is meant to stub a blockchain or smart contract which would store this in a decentralized way.
We store it as a separate file to independently verify it's working (so the CIDs have to match for example).

You can regenerate the sources.json with `go run util/generate-sources.go`.
It will take any files in the sources directory and format them appropriately.

In terms of our implementation of these things, take a look at the Start and New functions in `ipfs.go` they are fairly straightforward.
All we do is:
1. Set up all the ipfs stuff (bitswap, blockstore, blockservice, ...)
1. Connect to peers (not actually necessary since we also use p2p.go for connecting on the same host, but it's there for clarity)
1. Fetch all the metadata (Parse the root CID and all the children recursively)
1. Decide which blocks we want (current strategy is to chose randomly until we are out of space or available blocks). These are stored on the BlocksToSeed map.
1. Actually Get all the blocks we want. For each successful Get we log in the BlocksSeeding map.
1. Repeat previous step, or last 2 steps if the maximum storage changed in size.

### HTTP
We're using HTTP to receive health checks from docker.
That's in internal/api/http.go.

It's also used for the HTMX UI.
Just look for the routes starting with /htmx.
To show internal data we just pass a reference to the ipfs instance.

### Libp2p (mDNS and DHT) Usage
The libp2p instance uses mDNS for peer discovery and join the existing DHT.

This requires the following environment variables:

1. `XNODE_GROUP_NAME`: Unique string to identify and connect to group of nodes. Used in mDNS. Default: `Xnode`.
2. `XNODE_P2P_PORT`: Port for libp2p communications. Default: `10090`.

### Gossip
The gossip code starts on the internal/api/gossip.go `Start` function.
That's where we launch all the goroutines for tracking peers.

It's not really used in the current version of the program.

### Member Management (Gossip) Usage

The member management part requires three environment variables:

1. `XNODE_NAME`: Unique name for identifying this Xnode. Default: `Xnode-1`.
2. `XNODE_GOSSIP_PORT`: Port for Gossip protocol communication. Default: `9090`.
3. `XNODE_KNOWN_PEERS`: The addresses of other Xnodes for joining the existing Xnode cluster. If this is unset or blank (default), it will start a new Xnode cluster. Example: `172.17.0.2:9090,172.17.0.3:9091`.


---

# TODO:

## Bugs
- [X] ipfsInstance.Start() is blocking so gossip doesn't run
- [X] Nodes only seed half the data they store??? Or maybe metadata makes it look doubled?
    - turns out i was calling make() on the leaves and setting the size instead of the cap 🤦
- [X] Xnode-1 doesn't shutdown gracefully
- [ ] Xnode-1 takes up 1.5x the storage it should. Maybe blockstore is duplicated??
    - my suspicion is the file data itself might be being kept around by stale pointers

## IPFS
    - Self assign storage
        - Nodes have a picture of what data is available
            - *Compromise*: Internal list of all sources
        - Nodes chose the blocks they're storing as a subset of main list of sources
            - Should the listings include exact stats about format of blocks?
                - Size, Amount, and Layout? Basically the leaf nodes
                - Otherwise nodes have to fetch ACTUAL composition of the network from peers so they can't plan a greedy strategy before downloading metadata
        - Nodes 
    - [X] Download files
    - [X] Seed files
    - [X] Self allocate portions and only download those
    - [X] Should all be seeding the metadata
    - [X] Add status for each node (looking for peers, fetching metadata, picking blocks, seeding, etc...)
    - [X] Simplify process to generate sources, move the script from the other repo to over here

## GUI
    - [X] Get basic frontend working
        - [X] HTMX that makes GET requests to nodes for intel
        - [X] Show blocks status
        - [X] Show wanted blocks
        - [X] Kill nodes
        - [X] Change data size of nodes
