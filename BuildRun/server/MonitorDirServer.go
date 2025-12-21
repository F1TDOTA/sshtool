package server

import (
	"BuildRun/pkg/conf"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type MonitorDirService struct {
	Title       string
	fileService *FileTransferService
	sshConfObj  *conf.SshAllHost
	monConfObj  *conf.MonitorConf
	toastSvr    *ToastService
	mu          sync.Mutex
	lastSeen    map[string]time.Time
	debounce    time.Duration
}

func NewMonitorDirService(sshConfObj *conf.SshAllHost, monConfObj *conf.MonitorConf, fileSvr *FileTransferService, toastSvr *ToastService, debounce time.Duration) *MonitorDirService {
	return &MonitorDirService{
		Title:       "文件监控助手",
		sshConfObj:  sshConfObj,
		monConfObj:  monConfObj,
		fileService: fileSvr,
		lastSeen:    make(map[string]time.Time),
		debounce:    debounce,
		toastSvr:    toastSvr,
	}
}

func (s *MonitorDirService) addWatchRecursive(watcher *fsnotify.Watcher, dir string) error {
	const maxWatchDirs = 200
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
			//log.Printf("开始监控目录: %s\n", path)
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

func (s *MonitorDirService) sendCmdToScpFile(strFilePath string, strDstHost string, strUploadPath string, needToast string) {
	//{"oper_action":"send_file","oper_type":"local_file","src_path":"c:/monitor/test.cpp","dst_host":"192.168.1.180","dst_path":"/home/r/test.cpp"}
	var jsonFileCmd = JsonCmd{
		OperAction: "send_file",
		OperType:   "local_file",
		SrcPath:    strFilePath,
		DstHost:    strDstHost,
		DstPath:    strUploadPath,
		NeedToast:  needToast,
	}

	fmt.Println("json file cmd: ", jsonFileCmd)
	err := s.fileService.HandleCommand(jsonFileCmd)
	if err != nil {
		fmt.Println("sendCmdToScpFile err: ", err)
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

func (s *MonitorDirService) shouldHandle(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if t, ok := s.lastSeen[name]; ok {
		if now.Sub(t) < s.debounce {
			return false
		}
	}

	s.lastSeen[name] = now
	return true
}

func (s *MonitorDirService) initialSync(strMonitorDir string, strUploadIp string, strUploadRoot string) (int, error) {
	fmt.Println("start to initial sync")
	syncCount := 0

	err := filepath.WalkDir(strMonitorDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// 排除特定目录
		if strings.Contains(path, ".svn") ||
			strings.Contains(path, ".git") ||
			strings.Contains(path, ".vscode") ||
			strings.Contains(path, ".cursor") {
			return nil
		}

		syncCount++
		return s.handleOneFileUpload(strMonitorDir, strUploadIp, strUploadRoot, path, "false")
	})

	return syncCount, err
}

func (s *MonitorDirService) handleOneFileUpload(strMonitorDir string, strUploadIp string, strUploadRoot string, strModifyFile string, needToast string) error {
	strFileName, err := filepath.Rel(strMonitorDir, strModifyFile)
	if err != nil {
		return fmt.Errorf("filepath.Rel failed: err=%v file=%s", err, strModifyFile)
	}

	strUploadPath := s.toLinuxPath(filepath.Join(strUploadRoot, strFileName))

	if s.shouldHandle(strModifyFile) {
		s.sendCmdToScpFile(strModifyFile, strUploadIp, strUploadPath, needToast)
	}

	return nil
}

// 事件处理（简化版）
func (s *MonitorDirService) handleFileEvent(event fsnotify.Event, watcher *fsnotify.Watcher, strMonitorDir, strSshIp, strUploadRoot string) error {
	fmt.Println("")
	fmt.Println("事件: ", event)
	strEventName := string(event.Name)

	// 如果有新目录被创建 -> 递归添加到监控
	if event.Op&fsnotify.Create == fsnotify.Create {
		fi, err := os.Stat(strEventName)
		if err == nil && fi.IsDir() {
			// 新建目录时也要加入监控
			if err := s.addWatchRecursive(watcher, strEventName); err != nil {
				log.Println("添加新目录失败:", err)
				return fmt.Errorf("addWatchRecursive dir:%s  err:%v", strEventName, err)
			}
		}
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		fmt.Println("文件/目录创建:", strEventName)
	}

	if event.Op&fsnotify.Write == fsnotify.Write {
		fmt.Println("文件修改:", strEventName)
		// 修改后的文件进行上传
		err := s.handleOneFileUpload(strMonitorDir, strSshIp, strUploadRoot, strEventName, "true")
		if err != nil {
			fmt.Println("handleOneFileUpload err: ", err)
			return fmt.Errorf("handleOneFileUpload event:%s err:%v", strEventName, err)
		}
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

	return nil
}

func (s *MonitorDirService) handleToast(strMsg string) error {
	var jsonToast = &JsonToastCmd{
		Title:   s.Title,
		Message: strMsg,
	}

	jsonStr, err := json.Marshal(jsonToast)
	if err != nil {
		return fmt.Errorf("jsonToast marshal err %s\n", err)
	}

	s.toastSvr.SendToastMsg(string(jsonStr))
	return nil
}

func (s *MonitorDirService) Run() error {

	// 判断本地目录是否存在
	strMonitorDir := s.monConfObj.MonitorDir
	if !s.dirExists(strMonitorDir) {
		return fmt.Errorf("monitor dir %s not exists", strMonitorDir)
	}

	// 判断SSH服务器是否存在
	strSshIp := s.extractIP(s.monConfObj.UploadHost)
	fmt.Printf("parse monitor ip: %s\n", strSshIp)
	_, ok := s.sshConfObj.GetIpConf(strSshIp)
	if ok != true {
		return fmt.Errorf("dst ssh ip %s not exists", strSshIp)
	}

	// 检查上传路径是否为目录
	strUploadRoot := string(s.monConfObj.UploadPath)
	if strUploadRoot == "" {
		return fmt.Errorf("upload path is empty")
	}

	// 检查是否需要初始化
	if s.monConfObj.InitSyncAll == 1 {
		syncCount, err := s.initialSync(strMonitorDir, strSshIp, strUploadRoot)
		if err != nil {
			return err
		}
		strMsg := fmt.Sprintf("全量同步文件总数共计: %d个\n", syncCount)
		s.handleToast(strMsg)
	}

	// 创建监控器
	fmt.Printf("start to monitor dir: %s -> host: %s[%s]\n", strMonitorDir, strSshIp, strUploadRoot)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify NewWatcher failed: %v\n", err)
	}
	defer watcher.Close()

	// 初始递归添加所有目录
	if err := s.addWatchRecursive(watcher, strMonitorDir); err != nil {
		return fmt.Errorf("addWatchRecursive failed: %v\n", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("fsnotify Events failed")
			}
			if err := s.handleFileEvent(event, watcher, strMonitorDir, strSshIp, strUploadRoot); err != nil {
				return err
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("fsnotify Errors failed %v", err)
			}
			log.Println("错误:", err)
		}
	}
}

func (s *MonitorDirService) Stop() error {
	return nil
}
