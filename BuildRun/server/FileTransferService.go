package server

import (
	"BuildRun/pkg/conf"
	"BuildRun/pkg/session"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type FileTransferService struct {
	sessMgr  *session.SessMgr
	tempPath string
	toastSvr *ToastService
	AppId    string
}

func NewFileTransferService(toastSvr *ToastService) *FileTransferService {
	return &FileTransferService{
		sessMgr:  session.NewSessMgr(),
		tempPath: "./buildrun.tmp",
		toastSvr: toastSvr,
		AppId:    "文件传输助手",
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

func (s *FileTransferService) handleFile(confObj *conf.SshAllHost, srcIp string, srcPath string, dstIp string, dstPath string) error {
	// 编译机
	srcHostConf, ok := confObj.GetIpConf(srcIp)
	if ok != true {
		return fmt.Errorf("srcIp %s conf not exists\n", srcIp)
	}

	sessSrc := s.sessMgr.GetOneSess(session.SessTypeSCP, srcHostConf, srcIp)
	if sessSrc == nil {
		return fmt.Errorf("srcIp: %s get session fail\n", srcIp)
	}

	// 目标机
	dstHostConf, ok := confObj.GetIpConf(dstIp)
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
	md5Val, err := s.calMd5()

	// 上传文件
	if scpSessDst, ok := sessDst.(*session.ScpSess); ok {
		err = scpSessDst.UploadFileToRemote(s.tempPath, dstPath)
		if err != nil {
			return fmt.Errorf("upload file: %s to remote: %s error: %s\n", srcPath, dstPath, err)
		}
	} else {
		return fmt.Errorf("sessDst is not scp session: %T \n", sessDst)
	}

	// 弹出提示信息
	var jsonToast = &JsonToastCmd{
		AppId:   s.AppId,
		Message: fmt.Sprintf("文件 [%s] 上传成功\nmd5: [%s]", dstPath, md5Val),
	}

	jsonStr, err := json.Marshal(jsonToast)
	if err != nil {
		return fmt.Errorf("jsonToast marshal err %s\n", err)
	}
	s.toastSvr.SendToastMsg(string(jsonStr))

	return nil
}

func (s *FileTransferService) handleDir(dirPath string, dirRemotePath string) error {
	return nil
}

func (s *FileTransferService) HandleCommand(conObj *conf.SshAllHost, cmdJson JsonCmd) error {

	//命令格式如下：
	//{"oper_action":"send_file","oper_type":"file","src_host":"192.168.1.240","src_path":"/home/u/test.cpp","dst_host":"192.168.1.180","dst_path":"/home/r/a.c"}
	operType := cmdJson.OperType
	srcIp := cmdJson.SrcHost
	srcPath := cmdJson.SrcPath
	dstIp := cmdJson.DstHost
	dstPath := cmdJson.DstPath

	if operType == "file" {
		return s.handleFile(conObj, srcIp, srcPath, dstIp, dstPath)
	} else if operType == "dir" {

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
