package shadowsocks

// Server for shadowsocks multiuser
type Server interface {
	// UpdatePortPasswd port password would first close a port and restart listening on that
	// port. A different approach would be directly change the password used by
	// that port, but that requires **sharing** password between the port listener
	// and password manager.
	UpdatePortPasswd(port, password string, auth bool)
	Run(port, password string, auth bool)
	Reload()
	Start() error
}

// Client shadowsocks client
type Client interface {
	Run() error
	ParseServerConfig(string) error
}

// NewServer create instance for PasswdManager
func NewServer(cfgFile string) Server {
	return &passwdManager{
		portListener: map[string]*PortListener{},
		configFile:   cfgFile,
	}
}

// NewClient create instance for client
func NewClient() Client {
	return &client{}
}
