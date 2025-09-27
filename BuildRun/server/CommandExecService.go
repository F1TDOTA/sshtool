package server

import (
	"BuildRun/pkg/conf"
	"BuildRun/pkg/session"
	"encoding/json"
	"fmt"
)

type CommandExecService struct {
	sessMgr  *session.SshMgr
	strCmd   string
	toastSvr *ToastService
	AppId    string
}

func NewCommandExecService(toastSvr *ToastService) *CommandExecService {
	return &CommandExecService{
		sessMgr:  session.NewSshMgr(),
		strCmd:   "",
		toastSvr: toastSvr,
		AppId:    "命令执行助手",
	}
}

func (s *CommandExecService) HandleCommand(confObj *conf.SshConfig, cmdJson JsonCmd) error {
	//命令格式如下：
	//{"oper_action":"exec_cmd","dst_host":"192.168.1.180","dst_path":"/home/r/", "cmd_exec":"ls -l"}
	dstIp := cmdJson.DstHost
	dstPath := cmdJson.DstPath
	cmdExec := cmdJson.CmdExec

	c, ok := confObj.GetIpConf(dstIp)
	if ok != true {
		return fmt.Errorf("ssh dstIp:%s not exist\n", dstIp)
	}

	sess := s.sessMgr.GetOneSess(c, dstIp)
	if sess == nil {
		return fmt.Errorf("sessmgr get ip: %s sess fail\n", dstIp)
	}

	cmdExec = fmt.Sprintf("cd %s && %s", dstPath, cmdExec)
	err := sess.ExecCommand(cmdExec)
	if err != nil {
		return err
	}

	// 弹出提示信息
	var jsonToast = &JsonToastCmd{
		AppId:   s.AppId,
		Message: fmt.Sprintf("命令 [%s] 执行成功\n", cmdExec),
	}

	jsonStr, err := json.Marshal(jsonToast)
	if err != nil {
		return fmt.Errorf("jsonToast marshal err %s\n", err)
	}
	s.toastSvr.SendMsg(string(jsonStr))

	return nil
}

func (s *CommandExecService) PrintAllSess() {
	//命令格式如下：
	//{"oper_action":"print_ssh_session"}
	s.sessMgr.PrintAllSess()
}
