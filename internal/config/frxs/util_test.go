package util

import (
	"testing"
)

func TestConfig_GetLocalIP(t *testing.T) {
	localIP, _ := GetLocalIP()
	t.Logf("Local IP is : [%s]\n", localIP)
	ip, _ := GetAvaliableLocalIP()
	t.Logf("Avaliable Local IP is : [%s]\n", ip)
}
