package session

import (
	"BuildRun/pkg/conf"
	"fmt"
	"sync"
	"time"
)

const (
	SessTypeSSH = "ssh"
	SessTypeSCP = "scp"
)

type SessMgr struct {
	mu       sync.RWMutex
	AllSess  map[string]Session
	stopChan chan struct{}
}

func NewSessMgr() *SessMgr {
	sessMgr := &SessMgr{
		AllSess:  make(map[string]Session),
		stopChan: make(chan struct{}),
	}

	go sessMgr.TimeTaskCheckAllSess()
	return sessMgr
}

func (m *SessMgr) get(strIp string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.AllSess[strIp]
	return sess, ok
}

func (m *SessMgr) add(strIp string, sess Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AllSess[strIp] = sess
}

func (m *SessMgr) newSession(sessType string, conf *conf.SshHost) Session {
	var newSess Session
	switch sessType {
	case SessTypeSSH:
		newSess = NewSshSess(conf)
	case SessTypeSCP:
		newSess = NewScpSess(conf)
	default:
		newSess = nil
	}

	err := newSess.OpenSess(false)
	if err != nil {
		return nil
	}

	return newSess
}

func (m *SessMgr) GetOneSess(sessType string, conf *conf.SshHost, strIp string) Session {
	sess, ok := m.get(strIp)
	if ok {
		return sess
	}

	sessNew := m.newSession(sessType, conf)
	if sessNew == nil {
		return nil
	}

	m.add(strIp, sessNew)

	sess, ok = m.get(strIp)
	if ok {
		return sess
	}

	return nil
}

func (m *SessMgr) DestroyAllSession() {
	close(m.stopChan)

	m.mu.Lock()
	defer m.mu.Unlock()
	for ip, sess := range m.AllSess {
		sess.CloseSess()
		delete(m.AllSess, ip)
	}
}

func (m *SessMgr) PrintAllSess() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fmt.Printf("total sess num: %d\n", len(m.AllSess))
	for ip, sess := range m.AllSess {
		fmt.Printf("sess ip: %s, session: %+v\n", ip, sess)
	}
}

func (m *SessMgr) checkAllSess(strIp string, sess Session) {
	// 检查当前是否存活
	online := sess.CheckAlive()
	if online {
		sess.UpdateLastCheckTime()
		return
	}

	sess.CloseSess()
	delete(m.AllSess, strIp)

	// 存活失败，尝试重连
	fmt.Printf("ssh time check, ip: %s reconnect\n", strIp)
	err := sess.OpenSess(true)
	if err != nil {
		fmt.Printf("TimeTaskCheck ip: %s ReOpenSess error, ignore\n", strIp)
		return
	}

	fmt.Printf("ssh time check, ip: %s reconnect success\n", strIp)
}

func (m *SessMgr) TimeTaskCheckAllSess() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.RLock()
			for ip, sess := range m.AllSess {
				m.checkAllSess(ip, sess)
			}
			m.mu.RUnlock()

		case <-m.stopChan:
			fmt.Println("recv stop signal, TimeTaskCheckAllSess stop.")
			return
		}
	}
}
