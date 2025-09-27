package session

import (
	"BuildRun/pkg/conf"
	"fmt"
	"time"
)

type ScpMgr struct {
	AllSess  map[string]*ScpSess
	stopChan chan struct{}
}

func NewScpSessMgr() *ScpMgr {
	return &ScpMgr{
		AllSess: make(map[string]*ScpSess),
	}
}

func (m *ScpMgr) GetOneSess(c *conf.SshHost, strIp string) *ScpSess {
	data, ok := m.AllSess[strIp]
	if ok {
		return data
	}

	sess := NewScpSess(c)
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

func (m *ScpMgr) DestroyAllSession() {
	close(m.stopChan)
	for ip, v := range m.AllSess {
		v.CloseSess()
		delete(m.AllSess, ip)
	}
}

func (m *ScpMgr) PrintAllSess() {
	fmt.Println("======scp sess start=================")
	fmt.Printf("scp sess num: %d\n", len(m.AllSess))
	for ip, oneSess := range m.AllSess {
		fmt.Println("")
		fmt.Printf("IP: %s\n", ip)
		fmt.Printf("Status: %d\n", oneSess.onlineStatus)
		fmt.Printf("CreateTime: %s\n", oneSess.createTime)
		fmt.Printf("CheckTime: %s\n", oneSess.lastCheckTime)
	}
	fmt.Println("======scp sess end===================")
	fmt.Println("")
}

func (m *ScpMgr) checkAllSess(s *ScpSess) {
	// 检查当前是否存活
	online := s.CheckAlive()
	if online {
		s.lastCheckTime = time.Now().Format("2006-01-02 15:04:05")
		return
	}

	s.CloseSess()
	delete(m.AllSess, s.hostConf.Host)

	// 存活失败，尝试重连
	fmt.Printf("scp time check, ip: %s reconnect\n", s.hostConf.Host)
	err := s.OpenSess(true)
	if err != nil {
		fmt.Printf("scp TimeTaskCheck ip: %s ReOpenSess error, ignore\n", s.hostConf.Host)
		return
	}

	fmt.Printf("scp time check, ip: %s reconnect success\n", s.hostConf.Host)
}

func (m *ScpMgr) TimeTaskCheckAllSess() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, sess := range m.AllSess {
				m.checkAllSess(sess)
			}
		case <-m.stopChan:
			fmt.Println("recv stop signal, scp TimeTaskCheckAllSess stop.")
			return
		}
	}
}
