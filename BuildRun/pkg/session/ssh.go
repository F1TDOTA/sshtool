package session

import (
	"BuildRun/pkg/conf"
	"fmt"
	"time"

	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

type SshSess struct {
	hostConf      *conf.SshHost
	sessClient    *ssh.Client
	createTime    string // 创建时间
	lastCheckTime string // 上一次检测时间
}

func NewSshSess(hostConf *conf.SshHost) *SshSess {
	sess := &SshSess{
		hostConf:   hostConf,
		sessClient: nil,
	}

	return sess
}

func (s *SshSess) OpenSess(isReconnect bool) error {
	// 认证配置
	clientConf, err := auth.PasswordKey(s.hostConf.User, s.hostConf.Pass, ssh.InsecureIgnoreHostKey())
	if err != nil {
		fmt.Printf("ssh ip: %s auth.PasswordKey error, ignore\n", s.hostConf.Host)
		return err
	}

	// 建立SSH会话
	ipAddr := fmt.Sprintf("%s:%s", s.hostConf.Host, s.hostConf.Port)
	sshClient, err := ssh.Dial("tcp", ipAddr, &clientConf)
	if err != nil {
		fmt.Printf("ip: %s ssh.Dial error, ignore\n", s.hostConf.Host)
		return err
	}

	tm := time.Now()
	s.sessClient = sshClient
	s.lastCheckTime = tm.Format("2006-01-02 15:04:05")
	if !isReconnect {
		s.createTime = tm.Format("2006-01-02 15:04:05")
	}

	return nil
}

func (s *SshSess) CloseSess() {
	s.sessClient.Close()
	s.sessClient = nil
	s.createTime = ""
	s.lastCheckTime = ""
}

func (s *SshSess) CheckAlive() bool {
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

func (s *SshSess) ExecCommand(cmdExec string) error {
	if s.sessClient == nil {
		return fmt.Errorf("ssh exec sess client is nil, try open session\n")
	}

	online := s.CheckAlive()
	if !online {
		return fmt.Errorf("ssh exec sess client is not online.\n")
	}

	sess, err := s.sessClient.NewSession()
	if err != nil {
		return fmt.Errorf("ssh ip: %s ssh.NewSession error, ignore\n", s.hostConf.Host)
	}
	defer sess.Close()

	output, err := sess.CombinedOutput(cmdExec)
	if err != nil {
		return fmt.Errorf("ssh ip: %s ssh.CombinedOutput error, ignore\n", s.hostConf.Host)
	}
	fmt.Println(string(output))

	return nil
}
