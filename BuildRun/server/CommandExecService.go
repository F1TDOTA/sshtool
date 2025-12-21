package server

import (
	"BuildRun/pkg/conf"
	"BuildRun/pkg/session"
	"encoding/json"
	"fmt"
	"strconv"
)

type CommandExecService struct {
	Title    string
	confObj  *conf.SshAllHost
	sessMgr  *session.SessMgr
	toastSvr *ToastService
}

func NewCommandExecService(confObj *conf.SshAllHost, toastSvr *ToastService) *CommandExecService {
	return &CommandExecService{
		Title:    "命令执行助手",
		confObj:  confObj,
		sessMgr:  session.NewSessMgr(),
		toastSvr: toastSvr,
	}
}

func (s *CommandExecService) handleSshCmd(dstIp string, dstPath string, cmdExec string) error {
	c, ok := s.confObj.GetIpConf(dstIp)
	if ok != true {
		return fmt.Errorf("ssh dstIp:%s not exist", dstIp)
	}

	sess := s.sessMgr.GetOneSess(session.SessTypeSSH, c, dstIp)
	if sess == nil {
		return fmt.Errorf("sessmgr get ip: %s sess fail", dstIp)
	}

	if sshSess, ok := sess.(*session.SshSess); ok {
		fmt.Printf("cmdExec: %s", cmdExec)
		err := sshSess.ExecCommand(cmdExec)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("session is not *SshSess, actual type: %T", sess)
	}

	return nil
}

func (s *CommandExecService) handleToast(needToast bool, strMsg string) error {
	if !needToast {
		fmt.Println("CommandExecService no toast")
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

func (s *CommandExecService) HandleCommand(cmdJson JsonCmd) error {
	//命令格式如下：
	//{"oper_action":"exec_cmd","dst_host":"192.168.1.180","dst_path":"/home/r/", "cmd_exec":"ls -l"}
	dstIp := cmdJson.DstHost
	dstPath := cmdJson.DstPath

	cmdExec := cmdJson.CmdExec
	if dstPath != "" {
		cmdExec = fmt.Sprintf("cd %s && %s", dstPath, cmdExec)
	}

	boolValue, err := strconv.ParseBool(cmdJson.NeedToast)
	needToast := true
	if err == nil {
		needToast = boolValue
	}

	// 执行命令
	err = s.handleSshCmd(dstIp, dstPath, cmdExec)
	if err != nil {
		return err
	}

	// 弹出提示信息
	strMsg := fmt.Sprintf("命令 [%s] 执行成功\n", cmdExec)
	s.handleToast(needToast, strMsg)

	return nil
}

func (s *CommandExecService) PrintAllSess() {
	//命令格式如下：
	//{"oper_action":"print_ssh_session"}
	fmt.Println("======ssh sess start==================")
	s.sessMgr.PrintAllSess()
	fmt.Println("======ssh sess end====================")
}
