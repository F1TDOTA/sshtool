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

type SshHostArray struct {
	Hosts []SshHost `yaml:"hosts"`
}

type SshAllHost struct {
	sshMapHosts map[string]*SshHost
	confPath    string
}

func NewSshConfig() *SshAllHost {
	return &SshAllHost{
		sshMapHosts: make(map[string]*SshHost),
		confPath:    "./configs/ssh.yaml",
	}
}

func (c *SshAllHost) LoadHostConf() {
	data, err := os.ReadFile(c.confPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	var arr SshHostArray
	if err := yaml.Unmarshal(data, &arr); err != nil {
		fmt.Println(err)
		return
	}

	for _, host := range arr.Hosts {
		if host, ok := c.sshMapHosts[host.Host]; ok {
			fmt.Printf("host: %s is exist in map, ignore.\n", host.Host)
			continue
		}
		fmt.Printf("add host: %s to conf.\n", host.Host)
		c.sshMapHosts[host.Host] = &host
	}

	fmt.Printf("total cnt is %d\n", len(c.sshMapHosts))
}

func (c *SshAllHost) GetIpConf(strIp string) (*SshHost, bool) {
	host, ok := c.sshMapHosts[strIp]
	return host, ok
}
