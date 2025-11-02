package server

import (
	"BuildRun/pkg/conf"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type MonitorDirService struct {
	fileService *FileTransferService
	sshConfObj  *conf.SshAllHost
	monConfObj  *conf.MonitorConf
	AppId       string
}

func NewMonitorDirService(sshConfObj *conf.SshAllHost, monConfObj *conf.MonitorConf, fileSvr *FileTransferService) *MonitorDirService {
	return &MonitorDirService{
		sshConfObj:  sshConfObj,
		monConfObj:  monConfObj,
		fileService: fileSvr,
		AppId:       "文件监控助手",
	}
}

func (s *MonitorDirService) addWatchRecursive(watcher *fsnotify.Watcher, dir string) error {
	const maxWatchDirs = 100
	count := 0

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// 排除特定目录
		if strings.Contains(path, ".svn") ||
			strings.Contains(path, ".git") ||
			strings.Contains(path, ".vscode") ||
			strings.Contains(path, ".cursor") {
			return filepath.SkipDir
		}

		// 达到上限后停止遍历
		if count >= maxWatchDirs {
			log.Printf("watch dir %s is exceeds max watch dirs %d\n", path, maxWatchDirs)
			return filepath.SkipDir
		}

		if err := watcher.Add(path); err != nil {
			log.Printf("无法监控目录 %s: %v\n", path, err)
		} else {
			log.Printf("开始监控目录: %s\n", path)
			count++
		}

		return nil
	})
}

func (s *MonitorDirService) dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // 不存在或无法访问
	}
	return info.IsDir() // 存在且为目录
}

func (s *MonitorDirService) extractName(strName string) string {
	start := strings.Index(strName, "[")
	end := strings.Index(strName, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strName[start+1 : end]
}

func (s *MonitorDirService) extractIP(strName string) string {
	// 找第二组方括号
	first := strings.Index(strName, "]-[")
	if first == -1 {
		return ""
	}
	ipPort := strName[first+3 : len(strName)-1] // 去掉 ]-[ 和最后一个 ]
	parts := strings.Split(ipPort, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func (s *MonitorDirService) sendCmdToScpFile(strFilePath string, strDstHost string, strUploadPath string) {
	//{"oper_action":"send_file","oper_type":"local_file","src_path":"c:/monitor/test.cpp","dst_host":"192.168.1.180","dst_path":"/home/r/test.cpp"}
	var jsonFileCmd = JsonCmd{
		OperAction: "send_file",
		OperType:   "local_file",
		SrcPath:    strFilePath,
		DstHost:    strDstHost,
		DstPath:    strUploadPath,
	}

	fmt.Println("json file cmd: %v", jsonFileCmd)
	err := s.fileService.HandleCommand(s.sshConfObj, jsonFileCmd)
	if err != nil {
		fmt.Println("sendCmdToScpFile err: %v", err)
	}
}

func (s *MonitorDirService) toLinuxPath(p string) string {
	// 将所有反斜杠替换为正斜杠
	p = strings.ReplaceAll(p, "\\", "/")

	// 若不是以 '/' 开头，则补上
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func (s *MonitorDirService) Run() error {

	// 判断本地目录是否存在
	strMonitorDir := s.monConfObj.MonitorDir
	if !s.dirExists(strMonitorDir) {
		return fmt.Errorf("monitor dir %s not exists", strMonitorDir)
	}

	// 判断SSH服务器是否存在
	strSshIp := s.extractIP(s.monConfObj.UploadHost)
	fmt.Printf("monitor ip: %s\n", strSshIp)
	_, ok := s.sshConfObj.GetIpConf(strSshIp)
	if ok != true {
		return fmt.Errorf("dst ssh ip %s not exists", strSshIp)
	}

	// 检查上传路径是否为目录
	strUploadPath := string(s.monConfObj.UploadPath)

	// 创建监控器
	fmt.Printf("start to monitor dir: %s -> host: %s[%s]\n", strMonitorDir, strSshIp, strUploadPath)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify NewWatcher failed: %v\n", err)
	}
	defer watcher.Close()

	// 初始递归添加所有目录
	if err := s.addWatchRecursive(watcher, strMonitorDir); err != nil {
		return fmt.Errorf("addWatchRecursive failed: %v\n", err)
	}

	// 事件循环
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("fsnotify Events failed\n")
			}
			fmt.Println("事件:", event)

			// 如果有新目录被创建 -> 递归添加到监控
			if event.Op&fsnotify.Create == fsnotify.Create {
				fi, err := os.Stat(event.Name)
				if err == nil && fi.IsDir() {
					// 新建目录时也要加入监控
					if err := s.addWatchRecursive(watcher, event.Name); err != nil {
						log.Println("添加新目录失败:", err)
					}
				}
			}

			if event.Op&fsnotify.Create == fsnotify.Create {
				fmt.Println("文件/目录创建:", event.Name)
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Println("文件修改:", event.Name)
				// 修改后的文件进行上传
				strModifyFile := string(event.Name)
				strFileName := filepath.Base(strModifyFile)
				strUploadName := s.toLinuxPath(filepath.Join(strUploadPath, strFileName))
				s.sendCmdToScpFile(strModifyFile, strSshIp, strUploadName)
			}
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				fmt.Println("文件/目录删除:", event.Name)
			}
			if event.Op&fsnotify.Rename == fsnotify.Rename {
				fmt.Println("文件/目录重命名:", event.Name)
			}
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				fmt.Println("文件权限修改:", event.Name)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("fsnotify Errors failed %v\n", err)
			}
			fmt.Println("错误:", err)
		}
	}
}
