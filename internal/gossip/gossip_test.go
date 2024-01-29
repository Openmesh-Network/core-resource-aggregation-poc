package gossip_test

import (
    "context"
    "github.com/stretchr/testify/assert"
    "openmesh.network/aggregationpoc/internal/gossip"
    "testing"
    "time"
)

func TestNewInstance(t *testing.T) {
    ins := gossip.NewInstance("Xnode-1", 9090)
    assert.NotNil(t, ins)
}

func TestInstance_Start(t *testing.T) {
    ins1 := gossip.NewInstance("Xnode-1", 9090)
    assert.NotNil(t, ins1)
    ins2 := gossip.NewInstance("Xnode-2", 9091)
    assert.NotNil(t, ins2)
    cancelCtx, cancel := context.WithCancel(context.Background())
    ins1.Start(cancelCtx, []string{})
    ins2.Start(cancelCtx, []string{"127.0.0.1:9090"})
    cancel()
}

func TestInstance_Leave(t *testing.T) {
    ins1 := gossip.NewInstance("Xnode-1", 9090)
    assert.NotNil(t, ins1)
    ins2 := gossip.NewInstance("Xnode-2", 9091)
    assert.NotNil(t, ins2)
    cancelCtx, cancel := context.WithCancel(context.Background())
    ins1.Start(cancelCtx, []string{})
    ins2.Start(cancelCtx, []string{"127.0.0.1:9090"})
    err := ins2.Leave()
    assert.Nil(t, err)
    cancel()
}

func TestInstance_HealthCheck(t *testing.T) {
    ins1 := gossip.NewInstance("Xnode-1", 9090)
    assert.NotNil(t, ins1)
    ins2 := gossip.NewInstance("Xnode-2", 9091)
    assert.NotNil(t, ins2)
    cancelCtx, cancel := context.WithCancel(context.Background())
    ins1.Start(cancelCtx, []string{})
    ins2.Start(cancelCtx, []string{"127.0.0.1:9090"})
    // Wait for the health check result
    time.Sleep(7 * time.Second)
    cancel()
}
