package conf

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

type SshHost struct {
	Name       string `yaml:"name"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	User       string `yaml:"user"`
	Pass       string `yaml:"pass,omitempty"`
	PrivateKey string `yaml:"private_key,omitempty"`
}

type SshAllHost struct {
	sshMapHosts map[string]*SshHost
	confPath    string
}

func NewSshConfig() *SshAllHost {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("无法获取当前路径: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	return &SshAllHost{
		sshMapHosts: make(map[string]*SshHost),
		confPath:    filepath.Join(exeDir, "config.ini"),
	}
}

func (c *SshAllHost) LoadHostConf() {
	cfg, err := ini.Load(c.confPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
		return
	}

	hostList := cfg.Section("ssh").Key("hosts").String()
	hostNames := strings.Split(hostList, ",")
	for i := range hostNames {
		hostNames[i] = strings.TrimSpace(hostNames[i])
	}

	for _, secName := range hostNames {
		sec := cfg.Section(secName)
		if sec == nil {
			fmt.Printf("警告: 未找到 [%s] 段\n", secName)
			continue
		}

		strHost := sec.Key("host").String()
		if strHost == "" {
			fmt.Printf("section %s host is empty\n", secName)
			continue
		}

		strPort := sec.Key("port").String()
		if strPort == "" {
			fmt.Printf("section %s port is empty\n", secName)
			continue
		}

		strUser := sec.Key("user").String()
		if strUser == "" {
			fmt.Printf("section %s user is empty\n", secName)
			continue
		}

		h := SshHost{
			Name:       secName,
			Host:       strHost,
			Port:       strPort,
			User:       strUser,
			Pass:       sec.Key("pass").String(),
			PrivateKey: sec.Key("private_key").String(),
		}

		if host, ok := c.sshMapHosts[strHost]; ok {
			fmt.Printf("host: %s is exist in map, ignore.\n", host.Host)
			continue
		}
		fmt.Printf("add host: %s to conf.\n", strHost)
		c.sshMapHosts[strHost] = &h
	}

	fmt.Printf("total cnt is %d\n", len(c.sshMapHosts))
}

func (c *SshAllHost) GetIpConf(strIp string) (*SshHost, bool) {
	host, ok := c.sshMapHosts[strIp]
	return host, ok
}
