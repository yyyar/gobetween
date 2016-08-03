package server

import "../config"

type Server interface{
	Start() error
	Cfg() config.Server
	Stop()
}
