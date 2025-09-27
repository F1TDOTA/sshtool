package session

type Session interface {
	OpenSess(isReconnect bool) error
	CloseSess()
	CheckAlive() bool
	UpdateLastCheckTime()
}
