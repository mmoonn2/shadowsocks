package shadowsocks

import (
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"net"
	ss "shadowsocks/shadowsocks/shadowsocks"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

var (
	errAddrType      = errors.New("socks addr type not supported")
	errVer           = errors.New("socks version not supported")
	errMethod        = errors.New("socks only support 1 method now")
	errAuthExtraData = errors.New("socks authentication get extra data")
	errReqExtraData  = errors.New("socks request get extra data")
	errCmd           = errors.New("socks command not supported")
)

// ServerCipher Server Cipher
type ServerCipher struct {
	server string
	cipher *ss.Cipher
}

type client struct {
	config    *ss.Config
	srvCipher []*ServerCipher
	failCnt   []int // failed connection count
}

func (c *client) Run() (err error) {
	var (
		ln         net.Listener
		listenAddr = net.JoinHostPort(c.config.LocalAddr, strconv.Itoa(c.config.LocalPort))
	)

	if ln, err = net.Listen("tcp", listenAddr); err != nil {
		return
	}

	log.Infof("starting local socks5 server at %v ...\n", listenAddr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go c.handleConnection(conn)
	}
}

func (c *client) handleConnection(conn net.Conn) {
	log.Debugf("socks connect from %s\n", conn.RemoteAddr().String())
	closed := false
	defer func() {
		if !closed {
			conn.Close()
		}
	}()

	var err error
	if err = c.handShake(conn); err != nil {
		log.Errorln("socks handshake:", err)
		return
	}

	rawaddr, addr, err := c.getRequest(conn)
	if err != nil {
		log.Println("error getting request:", err)
		return
	}
	// Sending connection established message immediately to client.
	// This some round trip time for creating socks connection with the client.
	// But if connection failed, the client will get connection reset error.
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		log.Errorln("send connection confirmation:", err)
		return
	}

	remote, err := c.createServerConn(rawaddr, addr)
	if err != nil {
		if len(c.srvCipher) > 1 {
			log.Errorln("Failed connect to all avaiable shadowsocks server")
		}
		return
	}
	defer func() {
		if !closed {
			remote.Close()
		}
	}()

	go ss.PipeThenClose(conn, remote)
	ss.PipeThenClose(remote, conn)
	closed = true
	log.Debugln("closed connection to", addr)
}

// Connection to the server in the order specified in the config. On
// connection failure, try the next server. A failed server will be tried with
// some probability according to its fail count, so we can discover recovered
// servers.
func (c *client) createServerConn(rawaddr []byte, addr string) (remote *ss.Conn, err error) {
	const baseFailCnt = 20
	n := len(c.srvCipher)
	var skipped []int
	for i := 0; i < n; i++ {
		// skip failed server, but try it with some probability
		if c.failCnt[i] > 0 && rand.Intn(c.failCnt[i]+baseFailCnt) != 0 {
			skipped = append(skipped, i)
			continue
		}
		remote, err = c.connectToServer(i, rawaddr, addr)
		if err == nil {
			return
		}
	}
	// last resort, try skipped servers, not likely to succeed
	for _, i := range skipped {
		remote, err = c.connectToServer(i, rawaddr, addr)
		if err == nil {
			return
		}
	}
	return nil, err
}

func (c *client) connectToServer(serverID int, rawaddr []byte, addr string) (remote *ss.Conn, err error) {
	se := c.srvCipher[serverID]
	remote, err = ss.DialWithRawAddr(rawaddr, se.server, se.cipher.Copy())
	if err != nil {
		log.Errorln("error connecting to shadowsocks server:", err)
		const maxFailCnt = 30
		if c.failCnt[serverID] < maxFailCnt {
			c.failCnt[serverID]++
		}
		return nil, err
	}
	log.Debugf("connected to %s via %s", addr, se.server)
	c.failCnt[serverID] = 0
	return
}

func (c *client) getRequest(conn net.Conn) (rawaddr []byte, host string, err error) {
	const (
		idVer   = 0
		idCmd   = 1
		idType  = 3 // address type index
		idIP0   = 4 // ip addres start index
		idDmLen = 4 // domain address length index
		idDm0   = 5 // domain address start index

		typeIPv4 = 1 // type is ipv4 address
		typeDm   = 3 // type is domain address
		typeIPv6 = 4 // type is ipv6 address

		lenIPv4   = 3 + 1 + net.IPv4len + 2 // 3(ver+cmd+rsv) + 1addrType + ipv4 + 2port
		lenIPv6   = 3 + 1 + net.IPv6len + 2 // 3(ver+cmd+rsv) + 1addrType + ipv6 + 2port
		lenDmBase = 3 + 1 + 1 + 2           // 3 + 1addrType + 1addrLen + 2port, plus addrLen
	)
	// refer to getRequest in server.go for why set buffer size to 263
	buf := make([]byte, 263)
	var n int
	ss.SetReadTimeout(conn)
	// read till we get possible domain length field
	if n, err = io.ReadAtLeast(conn, buf, idDmLen+1); err != nil {
		return
	}
	// check version and cmd
	if buf[idVer] != socksVer5 {
		err = errVer
		return
	}
	if buf[idCmd] != socksCmdConnect {
		err = errCmd
		return
	}

	reqLen := -1
	switch buf[idType] {
	case typeIPv4:
		reqLen = lenIPv4
		host = net.IP(buf[idIP0 : idIP0+net.IPv4len]).String()
	case typeIPv6:
		reqLen = lenIPv6
		host = net.IP(buf[idIP0 : idIP0+net.IPv6len]).String()
	case typeDm:
		reqLen = int(buf[idDmLen]) + lenDmBase
		host = string(buf[idDm0 : idDm0+buf[idDmLen]])
	default:
		err = errAddrType
		return
	}

	if n == reqLen {
		// common case, do nothing
	} else if n < reqLen { // rare case
		if _, err = io.ReadFull(conn, buf[n:reqLen]); err != nil {
			return
		}
	} else {
		err = errReqExtraData
		return
	}

	rawaddr = buf[idType:reqLen]
	port := binary.BigEndian.Uint16(buf[reqLen-2 : reqLen])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))

	return
}

func (c *client) handShake(conn net.Conn) (err error) {
	const (
		idVer     = 0
		idNmethod = 1
	)
	// version identification and method selection message in theory can have
	// at most 256 methods, plus version and nmethod field in total 258 bytes
	// the current rfc defines only 3 authentication methods (plus 2 reserved),
	// so it won't be such long in practice

	buf := make([]byte, 258)

	var n int
	ss.SetReadTimeout(conn)
	// make sure we get the nmethod field
	if n, err = io.ReadAtLeast(conn, buf, idNmethod+1); err != nil {
		return
	}
	if buf[idVer] != socksVer5 {
		return errVer
	}
	nmethod := int(buf[idNmethod])
	msgLen := nmethod + 2
	if n == msgLen { // handshake done, common case
		// do nothing, jump directly to send confirmation
	} else if n < msgLen { // has more methods to read, rare case
		if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
			return
		}
	} else { // error, should not get extra data
		return errAuthExtraData
	}
	// send confirmation: version 5, no authentication required
	_, err = conn.Write([]byte{socksVer5, 0})
	return
}

func (c *client) ParseServerConfig(cfgFile string) (err error) {
	if strings.TrimSpace(cfgFile) == "" {
		err = errors.New("config must be set")
		return
	}
	var conf *ss.Config
	if conf, err = ss.ParseConfig(cfgFile); err != nil {
		return
	}

	c.parseServerConfig(conf)
	return
}

func (c *client) parseServerConfig(config *ss.Config) {
	hasPort := func(s string) bool {
		_, port, err := net.SplitHostPort(s)
		if err != nil {
			return false
		}
		return port != ""
	}

	if len(config.ServerPassword) == 0 {
		method := config.Method
		if config.Auth {
			method += "-auth"
		}
		// only one encryption table
		cipher, err := ss.NewCipher(method, config.Password)
		if err != nil {
			log.Fatal("Failed generating ciphers:", err)
		}
		srvPort := strconv.Itoa(config.ServerPort)
		srvArr := config.GetServerArray()
		n := len(srvArr)
		c.srvCipher = make([]*ServerCipher, n)

		for i, s := range srvArr {
			if hasPort(s) {
				log.Println("ignore server_port option for server", s)
				c.srvCipher[i] = &ServerCipher{s, cipher}
			} else {
				c.srvCipher[i] = &ServerCipher{net.JoinHostPort(s, srvPort), cipher}
			}
		}
	} else {
		// multiple servers
		n := len(config.ServerPassword)
		c.srvCipher = make([]*ServerCipher, n)

		cipherCache := make(map[string]*ss.Cipher)
		i := 0
		for _, serverInfo := range config.ServerPassword {
			if len(serverInfo) < 2 || len(serverInfo) > 3 {
				log.Fatalf("server %v syntax error\n", serverInfo)
			}
			server := serverInfo[0]
			passwd := serverInfo[1]
			encmethod := ""
			if len(serverInfo) == 3 {
				encmethod = serverInfo[2]
			}
			if !hasPort(server) {
				log.Fatalf("no port for server %s\n", server)
			}
			// Using "|" as delimiter is safe here, since no encryption
			// method contains it in the name.
			cacheKey := encmethod + "|" + passwd
			cipher, ok := cipherCache[cacheKey]
			if !ok {
				var err error
				cipher, err = ss.NewCipher(encmethod, passwd)
				if err != nil {
					log.Fatal("Failed generating ciphers:", err)
				}
				cipherCache[cacheKey] = cipher
			}
			c.srvCipher[i] = &ServerCipher{server, cipher}
			i++
		}
	}
	c.failCnt = make([]int, len(c.srvCipher))
	for _, se := range c.srvCipher {
		log.Println("available remote server", se.server)
	}
	c.config = config
	return
}
