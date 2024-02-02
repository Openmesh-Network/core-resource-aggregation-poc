package p2p_test

import (
    "context"
    "github.com/stretchr/testify/assert"
    "openmesh.network/aggregationpoc/internal/p2p"
    "testing"
    "time"
)

func TestNewLibP2PInstance(t *testing.T) {
    instance := p2p.NewLibP2PInstance(10090, "Xnode-test")
    assert.NotNil(t, instance)
    t.Logf("%#v", instance)
    instance.Stop()
}

func TestInstance_Start(t *testing.T) {
    instance := p2p.NewLibP2PInstance(10090, "Xnode-test")
    assert.NotNil(t, instance)
    err := instance.Start(context.Background())
    assert.Nil(t, err)
    err = instance.Stop()
    assert.Nil(t, err)
}

func TestDHT(t *testing.T) {
    // Initialise two peers within the same group
    gn := "Xnode-test"
    i1 := p2p.NewLibP2PInstance(10090, gn)
    i2 := p2p.NewLibP2PInstance(10091, gn)

    // Start peers
    err := i1.Start(context.Background())
    assert.Nil(t, err)
    err = i2.Start(context.Background())
    assert.Nil(t, err)

    time.Sleep(5 * time.Second)

    // Try DHT
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    err = i1.DHT.PutValue(ctx, "/xnode/instance_1", []byte("20"))
    assert.Nil(t, err)
    res, err := i2.DHT.GetValue(ctx, "/xnode/instance_1")
    assert.Nil(t, err)
    t.Logf("Got value in DHT: %s", string(res))

    // Cleanup
    err = i1.Stop()
    assert.Nil(t, err)
    err = i2.Stop()
    assert.Nil(t, err)
}
