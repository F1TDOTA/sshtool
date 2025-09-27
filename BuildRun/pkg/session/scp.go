package session

import (
	"BuildRun/pkg/conf"
	"context"
	"fmt"
	"os"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

type ScpSess struct {
	hostConf      *conf.SshHost
	sessClient    *ssh.Client
	createTime    string // 创建时间
	lastCheckTime string // 上一次检测时间
	onlineStatus  int    // 在线状态
}

func (s *ScpSess) Reconnect() {
	online := s.CheckAlive()
	if online {
		s.lastCheckTime = time.Now().Format("2006-01-02 15:04:05")
		return
	}

	// 重新建立连接
	s.sessClient.Close()

	// 认证配置
	clientConf, err := auth.PasswordKey(s.hostConf.User, s.hostConf.Pass, ssh.InsecureIgnoreHostKey())
	if err != nil {
		fmt.Printf("ip: %s auth.PasswordKey error, ignore\n", s.hostConf.Host)
		return
	}

	// 建立SSH会话
	ipAddr := fmt.Sprintf("%s:%s", s.hostConf.Host, s.hostConf.Port)
	sshClient, err := ssh.Dial("tcp", ipAddr, &clientConf)
	if err != nil {
		fmt.Printf("reconnect ip: %s ssh.Dial error, ignore\n", s.hostConf.Host)
		return
	}

	s.sessClient = sshClient
	s.lastCheckTime = time.Now().Format("2006-01-02 15:04:05")
}

func (s *ScpSess) CheckAlive() bool {
	sess, err := s.sessClient.NewSession()
	if err != nil {
		return false
	}

	defer sess.Close()
	return sess.Run("true") == nil
}

func (s *ScpSess) UploadFileToRemote(strLocalPath string, strRemotePath string) error {
	scpClient, err := scp.NewClientBySSH(s.sessClient)
	if err != nil {
		fmt.Printf("ip: %s scp.NewClientBySSH error, ignore\n", s.hostConf.Host)
		return err
	}

	f, err := os.Open(strLocalPath)
	if err != nil {
		fmt.Printf("Error opening file %s: %s\n", strLocalPath, err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = scpClient.CopyFile(ctx, f, strRemotePath, "0755")
	if err != nil {
		fmt.Printf("Error copying file %s: %s\n", strLocalPath, err)
		return err
	}

	fmt.Printf("Successfully upload file %s => %s\n", strLocalPath, strRemotePath)
	return nil
}

func (s *ScpSess) DownFileToLocal(strRemotePath string, strLocalFile string) error {
	scpClient, err := scp.NewClientBySSH(s.sessClient)
	if err != nil {
		fmt.Printf("ip: %s scp.NewClientBySSH error, ignore\n", s.hostConf.Host)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 打开本地文件
	f, err := os.Create(strLocalFile)
	if err != nil {
		fmt.Printf("Error creating file %s: %s\n", strLocalFile, err)
		return err
	}
	defer f.Close()

	// 下载文件
	err = scpClient.CopyFromRemote(ctx, f, strRemotePath)
	if err != nil {
		fmt.Printf("Error down file %s: %s\n", strRemotePath, err)
		return err
	}

	fmt.Printf("Successfully down file %s => %s\n", strRemotePath, strLocalFile)
	return nil
}

type ScpSessMgr struct {
	allSess map[string]*ScpSess
}

func NewScpSessMgr() *ScpSessMgr {
	return &ScpSessMgr{
		allSess: make(map[string]*ScpSess),
	}
}

func (m *ScpSessMgr) createOne(hostConf *conf.SshHost, strIp string) {
	// 认证配置
	clientConf, err := auth.PasswordKey(hostConf.User, hostConf.Pass, ssh.InsecureIgnoreHostKey())
	if err != nil {
		fmt.Printf("ip: %s auth.PasswordKey error, ignore\n", strIp)
		return
	}

	// 建立SSH会话
	ipAddr := fmt.Sprintf("%s:%s", hostConf.Host, hostConf.Port)
	sshClient, err := ssh.Dial("tcp", ipAddr, &clientConf)
	if err != nil {
		fmt.Printf("ip: %s ssh.Dial error, ignore\n", strIp)
		return
	}

	// 放入会话池
	tm := time.Now()
	oneSess := &ScpSess{
		hostConf:      hostConf,
		sessClient:    sshClient,
		createTime:    tm.Format("2006-01-02 15:04:05"),
		lastCheckTime: tm.Format("2006-01-02 15:04:05"),
		onlineStatus:  1,
	}

	fmt.Printf("ip: %s new scp session success, time: %s\n", strIp, oneSess.createTime)
	m.allSess[strIp] = oneSess
}

func (m *ScpSessMgr) GetSession(c *conf.SshHost, strIp string) *ScpSess {
	data, ok := m.allSess[strIp]
	if ok {
		return data
	}

	// 新建一个
	m.createOne(c, strIp)
	data, ok = m.allSess[strIp]
	if ok {
		return data
	}

	return nil
}

func (m *ScpSessMgr) DestroyAllSession() {
	for _, v := range m.allSess {
		v.sessClient.Close()
	}
}

func (m *ScpSessMgr) PrintAllSess() {
	fmt.Println("======scp sess start=================")
	fmt.Printf("scp sess num: %d\n", len(m.allSess))
	for ip, oneSess := range m.allSess {
		fmt.Println("")
		fmt.Printf("IP: %s\n", ip)
		fmt.Printf("Status: %d\n", oneSess.onlineStatus)
		fmt.Printf("CreateTime: %s\n", oneSess.createTime)
		fmt.Printf("CheckTime: %s\n", oneSess.lastCheckTime)
	}
	fmt.Println("======scp sess end===================")
	fmt.Println("")
}
