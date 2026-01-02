package conf

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gopkg.in/ini.v1"
)

type MonitorConf struct {
	confPath    string
	MonitorDir  string `json:"monitor_dir"`
	UploadHost  string `json:"upload_host"`
	UploadPath  string `json:"upload_path"`
	InitSyncAll int    `json:"init_sync_all"`
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

func (c *MonitorConf) EnsureUTF8(str string) (string, error) {
	// 已经是 UTF-8，直接返回
	if utf8.ValidString(str) {
		return str, nil
	}

	// 假定是 GBK，转成 UTF-8
	reader := transform.NewReader(
		bytes.NewReader([]byte(str)),
		simplifiedchinese.GBK.NewDecoder(),
	)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", fmt.Errorf("convert GBK to UTF-8 failed: %w", err)
	}

	return buf.String(), nil
}

func (c *MonitorConf) LoadMonitorConf() {
	fmt.Println("user conf: ", c.confPath)
	cfg, err := ini.Load(c.confPath)
	if err != nil {
		fmt.Printf("加载配置文件失败: %v", err)
		return
	}

	strTempMonitorDir := cfg.Section("monitor").Key("monitor_dir").String()
	strTempUploadHost := cfg.Section("monitor").Key("upload_host").String()
	strTempUploadRoot := cfg.Section("monitor").Key("upload_path").String()
	c.InitSyncAll = cfg.Section("monitor").Key("init_sync_all").MustInt(0)

	strMonitorDir, err := c.EnsureUTF8(strTempMonitorDir)
	if err != nil {
		fmt.Printf("monitor dir EnsureUTF8 fail: %v", err)
		return
	}
	c.MonitorDir = strMonitorDir

	strUploadHost, err := c.EnsureUTF8(strTempUploadHost)
	if err != nil {
		fmt.Printf("upload host EnsureUTF8 fail: %v", err)
		return
	}
	c.UploadHost = strUploadHost

	strUploadRoot, err := c.EnsureUTF8(strTempUploadRoot)
	if err != nil {
		fmt.Printf("upload root EnsureUTF8 fail: %v", err)
		return
	}
	c.UploadPath = strUploadRoot

	fmt.Printf("monitor dir: %s\n", c.MonitorDir)
	fmt.Printf("upload host: %s\n", c.UploadHost)
	fmt.Printf("upload path: %s\n", c.UploadPath)
	fmt.Printf("init sync all: %d\n", c.InitSyncAll)
	return
}
