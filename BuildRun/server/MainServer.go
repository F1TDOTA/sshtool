package server

import (
	"BuildRun/pkg/conf"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
)

type JsonCmd struct {
	OperAction string `json:"oper_action"`
	OperType   string `json:"oper_type"`
	SrcHost    string `json:"src_host"`
	SrcPath    string `json:"src_path"`
	DstHost    string `json:"dst_host"`
	DstPath    string `json:"dst_path"`
	CmdExec    string `json:"cmd_exec"`
	WarnMsg    string `json:"warn_msg"`
}

type JsonResp struct {
	Success string `json:"success"`
	OutMsg  string `json:"msg"`
}

// ===============================主服务实现============================
type Server struct {
	ipAddr       string
	bindPort     int
	ln           net.Listener
	confObj      *conf.SshConfig
	fileService  *FileTransferService
	ToastCh      chan string
	ToastService *ToastService
	CmdService   *CommandExecService
}

func NewServer(ipAddr string, bindPort int) *Server {
	// 初始配置服务
	confObj := conf.NewSshConfig()
	confObj.LoadHostConf()

	s := &Server{
		ipAddr:   ipAddr,
		bindPort: bindPort,
		confObj:  &confObj,
	}

	// 初始化toast服务
	s.ToastCh = make(chan string)
	s.ToastService = NewToastService(s.ToastCh)
	s.ToastService.Run()

	// 初始化scp会话
	s.fileService = NewFileTransferService(s.ToastService)

	// 初始化ssh会话
	s.CmdService = NewCommandExecService(s.ToastService)

	return s
}

func (s *Server) Start() {

	// 启动监听
	addr := fmt.Sprintf("%s:%d", s.ipAddr, s.bindPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Error listening on %s: %s\n", addr, err)
		return
	}
	s.ln = ln
	fmt.Printf("Listening on %s\n", addr)

	// 启动Accept
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %s\n", err)
			continue
		}
		go s.handleConnection(conn)
	}

}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		if err == io.EOF {
			fmt.Printf("Client: %s disconnected\n", conn.RemoteAddr())
		}
		fmt.Printf("Error reading from connection: %s\n", err)
		return
	}

	// 输出接收到的消息
	msg := strings.TrimSpace(string(buf[:n]))
	fmt.Println("")
	fmt.Printf("received ip: %s, msg: %s\n", conn.RemoteAddr(), msg)

	// json解析
	var cmdJson JsonCmd
	err = json.Unmarshal([]byte(msg), &cmdJson)
	if err != nil {
		fmt.Printf("Error unmarshalling json: %s\n", err)
		return
	}

	// 根据命令分配给不同的Service
	cmd := cmdJson.OperAction
	switch cmd {
	case "send_file":
		err = s.fileService.HandleCommand(s.confObj, cmdJson)
		break
	case "exec_cmd":
		err = s.CmdService.HandleCommand(s.confObj, cmdJson)
		break
	case "print_scp_session":
		s.fileService.PrintAllSess()
		break
	case "print_ssh_session":
		s.CmdService.PrintAllSess()
		break
	case "notify":
		s.ToastService.HandleCommand(cmdJson)
		break
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		break
	}

	// 返回数据
	var jsonObj JsonResp
	if err != nil {
		jsonObj.Success = "false"
		jsonObj.OutMsg = err.Error()
	} else {
		jsonObj.Success = "true"
	}

	resp, _ := json.Marshal(jsonObj)
	_, err = conn.Write([]byte(resp))
	if err != nil {
		fmt.Printf("Error writing to connection: %s\n", err)
		return
	}
}

func (s *Server) Stop() {

}
