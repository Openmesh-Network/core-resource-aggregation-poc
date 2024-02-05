# Xnode Resource Aggregation Layer Proof-of-Concept (PoC)

## Dockerfile Usage

```shell
docker build -t poc:latest .
# Just for test
docker run --rm xnode:latest
```

## Member Management (Gossip) Usage

The member management part requires three environment variables:

1. `XNODE_NAME`: Unique name for identifying this Xnode. Default: `Xnode-1`.
2. `XNODE_GOSSIP_PORT`: Port for Gossip protocol communication. Default: `9090`.
3. `XNODE_KNOWN_PEERS`: The addresses of other Xnodes for joining the existing Xnode cluster. If this is unset or blank (default), it will start a new Xnode cluster. Example: `172.17.0.2:9090,172.17.0.3:9091`.


## Getting up to speed
The codebase is small for now. 
If you just follow the flow of the program from main.go, 
you should get an idea of how everything is ran.

### HTTP
We're using HTTP to receive health checks from docker.
That's in internal/api/http.go.

### Gossip
The gossip code starts on the internal/api/gossip.go `Start` function.
That's where we launch all the goroutines for tracking peers.

### IPFS
The code for file sharing, splitting and all that.

# Notes
- The IPFS chunking thing breaks a thing up into 256kb blocks

# TODO:
## Bugs
- [X] ipfsInstance.Start() is blocking so gossip doesn't run
- [X] Nodes only seed half the data they store??? Or maybe metadata makes it look doubled?
    - turns out i was calling make() on the leaves and setting the size instead of the cap 🤦
- [ ] Xnode-1 doesn't shutdown gracefully
- [ ] Xnode-1 takes up double the storage it should. Maybe blockstore is duplicated??
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
    - [ ] Add status for each node (looking for peers, fetching metadata, picking blocks, seeding, etc...)
## GUI
    - [X] Get basic frontend working
        - [X] HTMX that makes GET requests to nodes for intel
        - [X] Show blocks status
        - [X] Show wanted blocks
        - [X] Kill nodes
        - [X] Change data size of nodes
