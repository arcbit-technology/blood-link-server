package controller

type Manager struct{}

var M *Manager

func NewManager() *Manager {
	return &Manager{}
}
