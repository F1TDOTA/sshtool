package conf

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type MonitorConf struct {
	confPath   string
	MonitorDir string `json:"monitor_dir"`
	UploadHost string `json:"upload_host"`
	UploadPath string `json:"upload_path"`
}

func NewMonitorConf() *MonitorConf {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("无法获取当前路径: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	return &MonitorConf{
		confPath: filepath.Join(exeDir, "config.ini"),
	}
}

func (c *MonitorConf) LoadMonitorConf() {
	fmt.Println("user conf: ", c.confPath)
	cfg, err := ini.Load(c.confPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
		return
	}

	c.MonitorDir = cfg.Section("monitor").Key("monitor_dir").String()
	c.UploadHost = cfg.Section("monitor").Key("upload_host").String()
	c.UploadPath = cfg.Section("monitor").Key("upload_path").String()

	fmt.Printf("monitor dir: %s\n", c.MonitorDir)
	fmt.Printf("upload host: %s\n", c.UploadHost)
	fmt.Printf("upload path: %s\n", c.UploadPath)
	return
}
