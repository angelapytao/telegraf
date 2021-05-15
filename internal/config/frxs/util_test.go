package util

import (
	"os"
	"testing"
)

func TestConfig_GetLocalIP(t *testing.T) {
	localIP, _ := GetLocalIP()
	t.Logf("Local IP is : [%s]\n", localIP)
	ip, _ := GetAvaliableLocalIP()
	t.Logf("Avaliable Local IP is : [%s]\n", ip)
}

func TestPortInUse(t *testing.T) {
	port := 5786
	pid := GetPidByPort(port)
	t.Logf("Port [%d] in use, Pid : [%v]\n", port, pid)
}

func TestExport(t *testing.T) {
	SetLocalEnvVariable("LOCAL_HOST", "1.3.5.66")

	env_val, _ := os.LookupEnv("LOCAL_HOST")
	t.Logf("LOCAL_HOST: %s \n", env_val)

	path, _ := os.LookupEnv("CONFIG_DIR_D")
	t.Logf("CONFIG_DIR_D: %s \n", path)
}