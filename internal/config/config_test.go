package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_UseLocalIP(t *testing.T) {
	c := NewConfig()
	err := c.LoadConfig("./testdata/agent-use-localhostIp.toml")
	assert.NoError(t, err)
	t.Logf("use_localIp_as_host = %v \n", c.Agent.UseLocalIPAsHost)
}
