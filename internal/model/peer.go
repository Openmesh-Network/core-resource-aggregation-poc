package model

// Peer is a single Xnode instance
type Peer struct {
    Name       string
    Hostname   string
    GossipPort uint16
    Alive      bool
}
