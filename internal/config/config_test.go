package config

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/influxdata/telegraf/plugins/inputs/http_listener_v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_UseLocalIP(t *testing.T) {
	c := NewConfig()
	err := c.LoadConfig("./testdata/agent-use-localhostIp.toml")
	assert.NoError(t, err)
	t.Logf("use_localIp_as_host = %v \n", c.Agent.UseLocalIPAsHost)
}

func TestWriteConfig(t *testing.T) {
	c := NewConfig()
	err := c.LoadConfig("./testdata/write_http_listen_addr.toml")
	assert.NoError(t, err)
	require.Equal(t, 1, len(c.Inputs))

	inputHTTPListener, ok := c.Inputs[0].Input.(*http_listener_v2.HTTPListenerV2)
	assert.Equal(t, true, ok)
	assert.Equal(t, ":8235", inputHTTPListener.ServiceAddress)
	t.Logf("config = %v \n", c)
}

func TestTextUnmarshaler(t *testing.T) {
	output := []byte(fmt.Sprintf(`[[inputs.http_listener_v2]]
  service_address = ":%s"
  data_format = "json"
	`, "1221"))

	fmt.Printf("Marshaled:\n%s", output)

	err := ioutil.WriteFile("./testdata/subconfig/http_listen_v2.toml", output, 0o644)
	if err != nil {
		fmt.Printf("WriteFile err: \n%v", err)
	}
}

func TestNotify(t *testing.T) {
	body := fmt.Sprintf(`{
		"ip": "%s",
		"port": "%s"
	}`, "172.21.146.243", "8236")

	t.Logf("body = %v \n", body)
}
