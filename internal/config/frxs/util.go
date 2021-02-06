package util

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Typed errors
var (
	ERR_NO_LOCAL_IP_FOUND = errors.New("No local IP Found.")
)

// 获取本机网卡IP
func GetLocalIP() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				return
			}
		}
	}

	err = ERR_NO_LOCAL_IP_FOUND
	return
}

func GetAvaliableLocalIP() (ipv4 string, err error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		err = ERR_NO_LOCAL_IP_FOUND
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ipv4 = ipnet.IP.String()
						return
					}
				}
			}
		}
	}
	return
}

func runInWindows(cmd string) (string, error) {
	result, err := exec.Command("cmd", "/c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(result)), err
}

func RunCommand(cmd string) (string, error) {
	if runtime.GOOS == "windows" {
		return runInWindows(cmd)
	} else {
		return runInLinux(cmd)
	}
}

func runInLinux(cmd string) (string, error) {
	// fmt.Println("Running Linux cmd:" + cmd)
	result, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(result)), err
}

// 根据进程名判断进程是否运行
func CheckProRunning(serverName string) (bool, error) {
	a := `ps ux | awk '/` + serverName + `/ && !/awk/ {print $2}'`
	pid, err := RunCommand(a)
	if err != nil {
		return false, err
	}
	return pid != "", nil
}

// 根据进程名称获取进程ID
func GetPid(serverName string) (string, error) {
	a := `ps ux | awk '/` + serverName + `/ && !/awk/ {print $2}'`
	pid, err := RunCommand(a)
	return pid, err
}

// 传入查询的端口号
// 返回端口号对应的进程PID，若没有找到相关进程，返回-1
func GetPidByPort(portNumber int) int {
	cmdStr := fmt.Sprintf("netstat -anvp tcp|grep '\\<%d\\>'|awk '{print $9}'", portNumber) // mac
	// cmdStr := fmt.Sprintf("sudo netstat -anp|grep 45388|awk '{print $7}'|awk -F '/' '{print $1}'", portNumber) // linux
	pid, err := RunCommand(cmdStr)
	if err != nil {
		return -1
	}

	processId, _ := strconv.Atoi(pid)
	return processId
}

// SetLocalEnvVariable 使用 os 库的 Setenv 函数来设置的环境变量
// 作用于整个进程的生命周期
func SetLocalEnvVariable(name, value string) bool {
	err := os.Setenv(name, value)
	return err == nil
}
