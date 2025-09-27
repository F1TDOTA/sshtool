package session

import (
	"BuildRun/pkg/conf"
	"fmt"
	"time"
)

type SshMgr struct {
	AllSess  map[string]*SshSess
	stopChan chan struct{}
}

func NewSshMgr() *SshMgr {
	s := &SshMgr{
		AllSess: make(map[string]*SshSess),
	}

	go s.TimeTaskCheckAllSess()
	return s
}

func (m *SshMgr) GetOneSess(c *conf.SshHost, strIp string) *SshSess {
	data, ok := m.AllSess[strIp]
	if ok {
		return data
	}

	// 创建会话
	sess := NewSshSess(c)
	err := sess.OpenSess(false)
	if err != nil {
		return nil
	}
	m.AllSess[strIp] = sess

	data, ok = m.AllSess[strIp]
	if ok {
		return data
	}

	return nil
}

func (m *SshMgr) CloseSshMgr(strIp string) {
	close(m.stopChan)
	if sess, ok := m.AllSess[strIp]; ok {
		sess.CloseSess()
		delete(m.AllSess, strIp)
	}
}

func (m *SshMgr) PrintAllSess() {
	fmt.Println("======ssh sess start=========")
	fmt.Printf("ssh sess num: %d\n", len(m.AllSess))
	for ip, sess := range m.AllSess {
		fmt.Println("")
		fmt.Printf("IP: %s\n", ip)
		fmt.Printf("CreateTime: %s\n", sess.createTime)
		fmt.Printf("CheckTime: %s\n", sess.lastCheckTime)
	}
	fmt.Println("======ssh sess end===================")
	fmt.Println("")
}

func (m *SshMgr) checkAllSess(s *SshSess) {
	// 检查当前是否存活
	online := s.CheckAlive()
	if online {
		s.lastCheckTime = time.Now().Format("2006-01-02 15:04:05")
		return
	}

	s.CloseSess()
	delete(m.AllSess, s.hostConf.Host)

	// 存活失败，尝试重连
	fmt.Printf("ssh time check, ip: %s reconnect\n", s.hostConf.Host)
	err := s.OpenSess(true)
	if err != nil {
		fmt.Printf("TimeTaskCheck ip: %s ReOpenSess error, ignore\n", s.hostConf.Host)
		return
	}

	fmt.Printf("ssh time check, ip: %s reconnect success\n", s.hostConf.Host)
}

func (m *SshMgr) TimeTaskCheckAllSess() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, sess := range m.AllSess {
				m.checkAllSess(sess)
			}
		case <-m.stopChan:
			fmt.Println("recv stop signal, ssh TimeTaskCheckAllSess stop.")
			return
		}
	}
}
