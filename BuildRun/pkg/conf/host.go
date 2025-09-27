package conf

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SshHost struct {
	Name       string `yaml:"name"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	User       string `yaml:"user"`
	Pass       string `yaml:"pass,omitempty"`
	PrivateKey string `yaml:"private_key,omitempty"`
}

type SshConfig struct {
	sshHosts map[string]*SshHost
}

func NewSshConfig() SshConfig {
	return SshConfig{
		sshHosts: make(map[string]*SshHost),
	}
}

func (h *SshConfig) LoadHostConf() {
	data, err := os.ReadFile("./configs/ssh.yaml")
	if err != nil {
		fmt.Println(err)
		return
	}

	var configs []SshHost
	if err := yaml.Unmarshal(data, &configs); err != nil {
		fmt.Println(err)
		return
	}

	for _, host := range configs {
		if host, ok := h.sshHosts[host.Host]; ok {
			fmt.Printf("host: %s is exist in map, ignore.\n", host.Host)
			continue
		}
		fmt.Printf("add host: %s to conf.\n", host.Host)
		h.sshHosts[host.Host] = &host
	}

	fmt.Printf("total cnt is %d\n", len(h.sshHosts))
}

func (h *SshConfig) GetIpConf(strIp string) (*SshHost, bool) {
	host, ok := h.sshHosts[strIp]
	return host, ok
}
