package gossip

import (
    "context"
    "fmt"
    "github.com/hashicorp/memberlist"
    "log"
    "net"
    "openmesh.network/aggregationpoc/internal/model"
    "sync"
    "time"
)

// Instance is the memberlist gossip instance for membership management and real-time data sharing.
type Instance struct {
    Name       string // Name for identifying this peer
    GossipPort int    // Port used for gossip communication
    Cluster    *memberlist.Memberlist
    Peers      []model.Peer // Known peers. Usually it's the result of the last round of health check
    PeersLock  sync.Mutex
}

// NewInstance create a Gossip instance
func NewInstance(name string, gossipPort int) *Instance {
    // Initialise a memberlist.List for this instance
    conf := memberlist.DefaultLocalConfig()
    conf.BindPort = int(gossipPort)
    conf.Name = name
    // TODO Add delegates for message sharing here
    //conf.Delegate = &DataDelegate{}

    cluster, err := memberlist.Create(conf)
    if err != nil {
        log.Fatalf("Failed to create gossip instance: %s", err.Error())
    }

    return &Instance{
        Name:       name,
        GossipPort: gossipPort,
        Cluster:    cluster,
        Peers:      make([]model.Peer, 0),
    }
}

// Start starting try to join an existing cluster via some known peers and starting health check
func (i *Instance) Start(ctx context.Context, knownPeers []string) {
    i.Join(ctx, knownPeers)
    go i.startHealthCheck(ctx)
}

// Join is for joining the cluster via some known peers
// If knownPeers is empty, it creates a new cluster with this peer itself
func (i *Instance) Join(ctx context.Context, knownPeers []string) {
    if len(knownPeers) <= 0 {
        return
    }
    // Try joining the cluster every 3 seconds
    t := time.NewTicker(3 * time.Second)
    for {
        select {
        case <-t.C:
            // Try to join the cluster
            _, err := i.Cluster.Join(knownPeers)
            if err != nil {
                log.Printf("Failed to join the cluster. Error: %s", err.Error())
                continue
            }
            log.Printf("Successfully joined the cluster via %v.", knownPeers)
            return
        case <-ctx.Done():
            return
        }
    }
}

// Leave is for leaving the cluster joined
func (i *Instance) Leave() error {
    if err := i.Cluster.Leave(5 * time.Second); err != nil {
        return err
    }
    log.Printf("Successfully left the existing cluster.")
    return nil
}

// startHealthCheck execute the health check process every 5 seconds
func (i *Instance) startHealthCheck(ctx context.Context) {
    t := time.NewTicker(5 * time.Second)
    for {
        select {
        case <-t.C:
            log.Printf("Starting health check at %s", time.Now().Format("15:04:05"))
            i.healthCheck()
            log.Printf("Health check finished at %s", time.Now().Format("15:04:05"))
        case <-ctx.Done():
            return
        }
    }
}

// healthCheck is checking the status for all members and update the list
func (i *Instance) healthCheck() {
    timeout := 5 * time.Second
    var peers []model.Peer // Health check result
    for _, peer := range i.Cluster.Members() {
        hostname := peer.Addr.String()
        gossipPort := peer.Port
        name := peer.Name

        p := model.Peer{
            Name:       name,
            Hostname:   hostname,
            GossipPort: int(gossipPort),
        }
        conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", hostname, gossipPort), timeout)
        // Successfully connected = alive, otherwise = dead
        p.Alive = err == nil
        peers = append(peers, p)

        if conn != nil {
            if err := conn.Close(); err != nil {
                log.Printf("Failed to close connection after health check: %s", err.Error())
            }
        }
    }
    // For debugging
    log.Printf("Health check result: %#v", peers)

    // Replace the original peers slice with the newest result
    i.PeersLock.Lock()
    defer i.PeersLock.Unlock()
    i.Peers = peers
}
