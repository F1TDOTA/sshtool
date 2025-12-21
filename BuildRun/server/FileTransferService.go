package server

import (
	"BuildRun/pkg/conf"
	"BuildRun/pkg/session"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FileTransferService struct {
	Title    string
	tempPath string
	confObj  *conf.SshAllHost
	sessMgr  *session.SessMgr
	cmdSvr   *CommandExecService
	toastSvr *ToastService
}

func NewFileTransferService(confObj *conf.SshAllHost, cmdSvr *CommandExecService, toastSvr *ToastService) *FileTransferService {
	return &FileTransferService{
		Title:    "文件传输助手",
		tempPath: "./buildrun.tmp",
		confObj:  confObj,
		sessMgr:  session.NewSessMgr(),
		cmdSvr:   cmdSvr,
		toastSvr: toastSvr,
	}
}

func (s *FileTransferService) calMd5() (string, error) {
	file, err := os.OpenFile(s.tempPath, os.O_RDONLY, 0777)
	if err != nil {
		fmt.Printf("open file %s err %s\n", s.tempPath, err)
		return "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		fmt.Printf("Error md5sum file %s err %s\n", s.tempPath, err)
		return "", err
	}

	md5Val := fmt.Sprintf("%x", h.Sum(nil))
	fmt.Printf("name: %s => md5: %s\n", s.tempPath, md5Val)
	return md5Val, nil
}

func (s *FileTransferService) toLinuxPath(p string) string {
	// 将所有反斜杠替换为正斜杠
	p = strings.ReplaceAll(p, "\\", "/")

	// 若不是以 '/' 开头，则补上
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func (s *FileTransferService) handleMakeDir(dstIp string, dstPath string) error {
	dstDir := filepath.Dir(dstPath)
	dstLinuxDir := s.toLinuxPath(dstDir)

	mkdirCmd := fmt.Sprintf("mkdir -p %s", dstLinuxDir)
	var cmdJson = JsonCmd{
		OperAction: "exec_cmd",
		DstHost:    dstIp,
		CmdExec:    mkdirCmd,
		NeedToast:  "false",
	}

	return s.cmdSvr.HandleCommand(cmdJson)
}

func (s *FileTransferService) handleToast(needToast bool, strMsg string) error {
	if !needToast {
		fmt.Println("FileTransferService no toast")
		return nil
	}

	// 弹出提示信息
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

func (s *FileTransferService) handleFile(srcIp string, srcPath string, dstIp string, dstPath string, needToast bool) error {
	// 源机器
	srcHostConf, ok := s.confObj.GetIpConf(srcIp)
	if ok != true {
		return fmt.Errorf("srcIp %s conf not exists\n", srcIp)
	}

	sessSrc := s.sessMgr.GetOneSess(session.SessTypeSCP, srcHostConf, srcIp)
	if sessSrc == nil {
		return fmt.Errorf("srcIp: %s get session fail\n", srcIp)
	}

	// 目标机
	dstHostConf, ok := s.confObj.GetIpConf(dstIp)
	if ok != true {
		return fmt.Errorf("dstIp %s conf not exists\n", dstIp)
	}

	sessDst := s.sessMgr.GetOneSess(session.SessTypeSCP, dstHostConf, dstIp)
	if sessDst == nil {
		return fmt.Errorf("dstIp: %s get session fail\n", dstIp)
	}

	// 下载文件到本地
	if scpSessSrc, ok := sessSrc.(*session.ScpSess); ok {
		err := scpSessSrc.DownFileToLocal(srcPath, s.tempPath)
		if err != nil {
			return fmt.Errorf("down file: %s to local: %s error: %s\n", srcPath, s.tempPath, err)
		}
	} else {
		return fmt.Errorf("sessSrc is not scp session: %T \n", sessSrc)
	}

	// 计算文件md5(不用处理返回值)
	md5Val, _ := s.calMd5()

	// 新建目录(给命令服务发请求）
	err := s.handleMakeDir(dstIp, dstPath)
	if err != nil {
		return fmt.Errorf("handleLocalFile mkdir err %s\n", err)
	}

	// 上传文件
	if scpSessDst, ok := sessDst.(*session.ScpSess); ok {
		err = scpSessDst.UploadFileToRemote(s.tempPath, dstPath)
		if err != nil {
			return fmt.Errorf("upload file: %s to remote: %s error: %s\n", srcPath, dstPath, err)
		}
	} else {
		return fmt.Errorf("sessDst is not scp session: %T \n", sessDst)
	}

	strMsg := fmt.Sprintf("文件 [%s] 上传成功\nmd5: [%s]", dstPath, md5Val)
	return s.handleToast(needToast, strMsg)
}

func (s *FileTransferService) handleLocalFile(srcPath string, dstIp string, dstPath string, needToast bool) error {
	// 目标机
	dstHostConf, ok := s.confObj.GetIpConf(dstIp)
	if ok != true {
		return fmt.Errorf("handleLocalFile dstIp %s conf not exists\n", dstIp)
	}

	// scp会话
	sessScp := s.sessMgr.GetOneSess(session.SessTypeSCP, dstHostConf, dstIp)
	if sessScp == nil {
		return fmt.Errorf("handleLocalFile dstIp: %s get scp session fail\n", dstIp)
	}

	scpSessDst, ok := sessScp.(*session.ScpSess)
	if !ok {
		return fmt.Errorf("sessScp is not scp session: %T \n", sessScp)
	}

	// 新建目录(给命令服务发请求）
	err := s.handleMakeDir(dstIp, dstPath)
	if err != nil {
		return fmt.Errorf("handleLocalFile mkdir err %s\n", err)
	}

	// 上传文件
	err = scpSessDst.UploadFileToRemote(srcPath, dstPath)
	if err != nil {
		return fmt.Errorf("upload local file: %s to remote: %s error: %s\n", srcPath, dstPath, err)
	}

	strMsg := fmt.Sprintf("本地文件 [%s] 上传成功\n", dstPath)
	return s.handleToast(needToast, strMsg)
}

// ensureDirExists 确保目标路径的目录存在
func (s *FileTransferService) ensureLocalDirExists(filePath string) error {
	dir := filepath.Dir(filePath)
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 如果目录不存在，创建目录
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

// copyFile 将源文件复制到目标路径
func (s *FileTransferService) copyFileToDst(src, dst string) error {
	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %v", src, err)
	}
	defer sourceFile.Close()

	// 确保目标目录存在
	if err := s.ensureLocalDirExists(dst); err != nil {
		return err
	}

	// 打开目标文件
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %v", dst, err)
	}
	defer destFile.Close()

	// 复制源文件内容到目标文件
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy content to %s: %v", dst, err)
	}

	// 确保文件同步到磁盘
	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file %s: %v", dst, err)
	}

	return nil
}

func (s *FileTransferService) handleDownFile(srcIp string, srcPath string, dstPath string, needToast bool) error {
	// 源机器
	srcHostConf, ok := s.confObj.GetIpConf(srcIp)
	if ok != true {
		return fmt.Errorf("srcIp %s conf not exists\n", srcIp)
	}

	sessSrc := s.sessMgr.GetOneSess(session.SessTypeSCP, srcHostConf, srcIp)
	if sessSrc == nil {
		return fmt.Errorf("srcIp: %s get session fail\n", srcIp)
	}

	// 下载文件到本地
	if scpSessSrc, ok := sessSrc.(*session.ScpSess); ok {
		err := scpSessSrc.DownFileToLocal(srcPath, s.tempPath)
		if err != nil {
			return fmt.Errorf("down file: %s to local: %s error: %s\n", srcPath, s.tempPath, err)
		}
	} else {
		return fmt.Errorf("sessSrc is not scp session: %T \n", sessSrc)
	}

	// 计算文件md5(不用处理返回值)
	md5Val, _ := s.calMd5()

	// 调用复制函数
	err := s.copyFileToDst(s.tempPath, dstPath)
	if err != nil {
		return fmt.Errorf("copy file to %s error: %s\n", dstPath, err)
	}

	strMsg := fmt.Sprintf("文件 [%s] 下载到本地成功\nmd5: [%s]", dstPath, md5Val)
	return s.handleToast(needToast, strMsg)
}

func (s *FileTransferService) handleDir(dirPath string, dirRemotePath string) error {
	return nil
}

func (s *FileTransferService) HandleCommand(cmdJson JsonCmd) error {
	//命令格式如下：
	/* 从编译机下载上传到目标机
	{
		"oper_action": "send_file",
		"oper_type": "file",
		"need_toast": "true",
		"src_host": "192.168.1.240",
		"src_path": "/home/u/test.cpp",
		"dst_host": "192.168.1.180",
		"dst_path": "/home/r/a.c"
	}
	*/

	/*从编译机下载上传到目标机
	{
	    "oper_action": "send_file",
	    "oper_type": "local_file",
	    "need_toast": "true",
	    "src_path": "c:/monitor/test.cpp",
	    "dst_host": "192.168.1.180",
	    "dst_path": "/home/r/test.cpp"
	}
	*/

	/* 下载文件到本地：
	{
	    "oper_action": "send_file",
	    "oper_type": "down_file",
	    "need_toast": "true",
	    "src_host": "192.168.1.240",
	    "src_path": "/home/u/test.cpp",
	    "dst_path": "c:/test.cpp"
	}
	*/
	operType := cmdJson.OperType
	srcIp := cmdJson.SrcHost
	srcPath := cmdJson.SrcPath
	dstIp := cmdJson.DstHost
	dstPath := cmdJson.DstPath

	boolValue, err := strconv.ParseBool(cmdJson.NeedToast)
	needToast := true
	if err == nil {
		needToast = boolValue
	}

	if operType == "file" {
		return s.handleFile(srcIp, srcPath, dstIp, dstPath, needToast)
	} else if operType == "local_file" {
		return s.handleLocalFile(srcPath, dstIp, dstPath, needToast)
	} else if operType == "down_file" {
		return s.handleDownFile(srcIp, srcPath, dstPath, needToast)
	}

	return nil
}

func (s *FileTransferService) PrintAllSess() {
	//命令格式如下：
	//{"oper_action":"print_scp_session"}
	fmt.Println("======scp sess start==================")
	s.sessMgr.PrintAllSess()
	fmt.Println("======scp sess end====================")
	fmt.Println("")
}
