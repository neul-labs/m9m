package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_NilConfig(t *testing.T) {
	c := NewClient(nil)
	require.NotNil(t, c)
	assert.NotEmpty(t, c.socketPath)
	assert.Equal(t, DefaultConnectTimeout, c.connectTimeout)
	assert.Equal(t, DefaultRequestTimeout, c.requestTimeout)
}

func TestNewClient_CustomConfig(t *testing.T) {
	c := NewClient(&ClientConfig{
		SocketPath:     "/tmp/test.sock",
		ConnectTimeout: 10 * time.Second,
		RequestTimeout: 30 * time.Second,
	})
	require.NotNil(t, c)
	assert.Equal(t, "/tmp/test.sock", c.socketPath)
	assert.Equal(t, 10*time.Second, c.connectTimeout)
	assert.Equal(t, 30*time.Second, c.requestTimeout)
}

func TestNewClient_PartialConfig(t *testing.T) {
	c := NewClient(&ClientConfig{
		SocketPath: "/tmp/custom.sock",
	})
	require.NotNil(t, c)
	assert.Equal(t, "/tmp/custom.sock", c.socketPath)
	assert.Equal(t, DefaultConnectTimeout, c.connectTimeout)
	assert.Equal(t, DefaultRequestTimeout, c.requestTimeout)
}

func TestClient_IsRunning_NoSocket(t *testing.T) {
	c := NewClient(&ClientConfig{
		SocketPath: "/tmp/nonexistent-test-socket.sock",
	})
	assert.False(t, c.IsRunning())
}

func TestGetSocketPath(t *testing.T) {
	path := GetSocketPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "m9m")
}

func TestDefaultConstants(t *testing.T) {
	assert.Equal(t, 5*time.Second, DefaultConnectTimeout)
	assert.Equal(t, 60*time.Second, DefaultRequestTimeout)
}
