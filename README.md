# Xnode Resource Aggregation Layer Proof-of-Concept (PoC)

## Project Layout

- `main.go`: The main function. Entrance of the program.
- `internal` directory: Internal libraries used by this project.
  - `api` directory: HTTP RESTful API related things.
  - `gossip` directory: Membership management and eventual consistent data sharing.
  - `instance` directory: Peer instance aka top-level instance.
  - `model` directory: Data types used by this project.

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

## Libp2p (mDNS and DHT) Usage

The libp2p instance uses mDNS for peer discovery and join the existing DHT.

This requires the following environment variables:

1. `XNODE_GROUP_NAME`: Unique string to identify and connect to group of nodes. Used in mDNS. Default: `Xnode`.
2. `XNODE_P2P_PORT`: Port for libp2p communications. Default: `10090`.
