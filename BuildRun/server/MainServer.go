package server

import (
	"BuildRun/pkg/conf"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
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
	NeedToast  string `json:"need_toast"`
}

type JsonResp struct {
	Success string `json:"success"`
	OutMsg  string `json:"msg"`
}

// ===============================主服务实现============================
type Server struct {
	ipAddr         string
	bindPort       int
	ln             net.Listener
	sshConfObj     *conf.SshAllHost
	monConfObj     *conf.MonitorConf
	fileService    *FileTransferService
	CmdService     *CommandExecService
	ToastCh        chan string
	ToastService   *ToastService
	DingCh         chan string
	DingService    *DdNoticeService
	MonitorService *MonitorDirService
}

func NewServer(ipAddr string, bindPort int) (*Server, error) {
	// 初始SSH配置
	sshConfObj := conf.NewSshConfig()
	sshConfObj.LoadHostConf()

	// 初始化监控配置
	monConfObj := conf.NewMonitorConf()
	monConfObj.LoadMonitorConf()

	s := &Server{
		ipAddr:     ipAddr,
		bindPort:   bindPort,
		sshConfObj: sshConfObj,
		monConfObj: monConfObj,
	}

	// 初始化toast服务
	s.ToastCh = make(chan string)
	s.ToastService = NewToastService(s.ToastCh)

	// 初始化钉钉消息
	s.DingService = NewDdNoticeService(s.ToastService)

	// 初始化ssh会话
	s.CmdService = NewCommandExecService(s.sshConfObj, s.ToastService)

	// 初始化scp会话
	s.fileService = NewFileTransferService(s.sshConfObj, s.CmdService, s.ToastService)

	// 初始化监控服务
	s.MonitorService = NewMonitorDirService(s.sshConfObj, s.monConfObj, s.fileService, s.ToastService, 500*time.Millisecond)

	return s, nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			// 正常关闭 listener 时会触发
			if errors.Is(err, net.ErrClosed) {
				return
			}

			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) Start() error {

	// 启动后台服务（不阻塞）
	s.ToastService.Run()
	s.DingService.Run()

	// 启动监听
	addr := fmt.Sprintf("%s:%d", s.ipAddr, s.bindPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s failed: %w", addr, err)
	}
	s.ln = ln
	fmt.Printf("Listening on %s\n", addr)

	// 启动Accept
	go s.acceptLoop()
	go s.MonitorService.Run()

	return nil
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
	case "exec_cmd":
		err = s.CmdService.HandleCommand(cmdJson)
		break
	case "send_file":
		err = s.fileService.HandleCommand(cmdJson)
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
	case "notify_dd":
		s.DingService.HandleCommand(cmdJson)
		fmt.Printf("process finished\n")
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
	/*
		if s.ln != nil {
			s.ln.Close()
		}

		s.MonitorService.Stop()
		s.ToastService.Stop()
		s.DingService.Stop()
	*/
}
