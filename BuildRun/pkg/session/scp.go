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

func NewScpSess(hostConf *conf.SshHost) *ScpSess {
	return &ScpSess{
		hostConf:     hostConf,
		sessClient:   nil,
		onlineStatus: 0,
	}
}

func (s *ScpSess) OpenSess(isReconnect bool) error {
	// 认证配置
	clientConf, err := auth.PasswordKey(s.hostConf.User, s.hostConf.Pass, ssh.InsecureIgnoreHostKey())
	if err != nil {
		return fmt.Errorf("ip: %s auth.PasswordKey error, ignore\n", s.hostConf.Host)
	}

	// 建立SSH会话
	ipAddr := fmt.Sprintf("%s:%s", s.hostConf.Host, s.hostConf.Port)
	sshClient, err := ssh.Dial("tcp", ipAddr, &clientConf)
	if err != nil {
		return fmt.Errorf("ip: %s ssh.Dial error, ignore\n", s.hostConf.Host)
	}

	tm := time.Now()
	s.sessClient = sshClient
	s.lastCheckTime = tm.Format("2006-01-02 15:04:05")
	s.onlineStatus = 1
	if !isReconnect {
		s.createTime = tm.Format("2006-01-02 15:04:05")
	}

	fmt.Printf("ip: %s new scp session success, time: %s\n", s.hostConf.Host, s.createTime)
	return nil
}

func (s *ScpSess) CloseSess() {
	s.sessClient.Close()
	s.sessClient = nil
	s.createTime = ""
	s.lastCheckTime = ""
}

func (s *ScpSess) CheckAlive() bool {
	if s.sessClient == nil {
		return false
	}

	sess, err := s.sessClient.NewSession()
	if err != nil {
		return false
	}

	defer sess.Close()
	return sess.Run("true") == nil
}

func (s *ScpSess) PrintStatus() {
	fmt.Printf("scp ip: %s, session: %+v", s.hostConf.Host, s)
}

func (s *ScpSess) UpdateLastCheckTime() {
	s.lastCheckTime = time.Now().Format("2006-01-02 15:04:05")
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
