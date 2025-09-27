package session

import "BuildRun/pkg/conf"

type Session interface {
	OpenSess(isReconnect bool) error
	CloseSess()
	CheckAlive() bool
	PrintStatus()
	UpdateLastCheckTime()
}

type SessionMgr interface {
	GetOneSess(c *conf.SshHost, strIp string) *Session
	DestroyAllSession()
	PrintAllSess()
	TimeTaskCheckAllSess()
}
