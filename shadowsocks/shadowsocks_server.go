package shadowsocks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	ss "shadowsocks/shadowsocks/shadowsocks"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

const logCntDelta int64 = 100

var nextLogConnCnt = logCntDelta

// PortListener listen port
type PortListener struct {
	password string
	listener net.Listener
}

type passwdManager struct {
	sync.Mutex
	config       *ss.Config
	configFile   string
	connCnt      int64
	portListener map[string]*PortListener
}

func (pm *passwdManager) UpdatePortPasswd(port, password string, auth bool) {
	pl, ok := pm.get(port)
	if !ok {
		log.Infof("new port %s added\n", port)
	} else {
		if pl.password == password {
			return
		}
		log.Infof("closing port %s to update password\n", port)
		pl.listener.Close()
	}
	// run will add the new port listener to passwdManager.
	// So there maybe concurrent access to passwdManager and we need lock to protect it.
	go pm.Run(port, password, auth)
}

func (pm *passwdManager) Reload() {
	log.Infoln("updating password")
	newconfig, err := ss.ParseConfig(pm.configFile)
	if err != nil {
		log.Errorf("error parsing config file %s to update password: %v\n", pm.configFile, err)
		return
	}
	oldconfig := pm.config
	pm.config = newconfig

	if err = pm.unifyPortPassword(); err != nil {
		return
	}
	for port, passwd := range pm.config.PortPassword {
		pm.UpdatePortPasswd(port, passwd, pm.config.Auth)
		if oldconfig.PortPassword != nil {
			delete(oldconfig.PortPassword, port)
		}
	}
	// port password still left in the old config should be closed
	for port := range oldconfig.PortPassword {
		log.Infof("closing port %s as it's deleted\n", port)
		pm.del(port)
	}
	log.Infoln("password updated")
}

func (pm *passwdManager) Run(port, password string, auth bool) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("listening port %v: %v", port, err)
	}

	pm.add(port, password, ln)
	var cipher *ss.Cipher
	log.Infof("server listening port %v ...", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			// listener maybe closed to update password
			log.Debugf("accept error: %v", err)
			return
		}

		// Creating cipher upon first connection.
		if cipher == nil {
			log.Infoln("creating cipher for port:", port)
			cipher, err = ss.NewCipher(pm.config.Method, password)
			if err != nil {
				log.Errorf("Error generating cipher for port: %s %v\n", port, err)
				conn.Close()
				continue
			}
		}

		go pm.handleConnection(ss.NewConn(conn, cipher.Copy()), auth)
	}
}

func (pm *passwdManager) handleConnection(conn *ss.Conn, auth bool) {
	atomic.AddInt64(&pm.connCnt, 1)
	if pm.connCnt-nextLogConnCnt >= 0 {
		log.Warnf("Number of client connections reaches %d\n", nextLogConnCnt)
		nextLogConnCnt += logCntDelta
	}

	log.Debugf("new client %s->%s\n", conn.RemoteAddr().String(), conn.LocalAddr())
	var (
		host   string
		closed = false
	)

	defer func() {
		log.Debugf("closed pipe %s<->%s\n", conn.RemoteAddr(), host)
		atomic.AddInt64(&pm.connCnt, ^int64(0))

		if !closed {
			conn.Close()
		}
	}()

	host, ota, err := getRequest(conn, auth)
	if err != nil {
		log.Errorln("error getting request", conn.RemoteAddr(), conn.LocalAddr(), err)
		return
	}
	log.Debugln("connecting", host)
	remote, err := net.Dial("tcp", host)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
			// log too many open file error
			// EMFILE is process reaches open file limits, ENFILE is system limit
			log.Errorln("dial error:", err)
		} else {
			log.Errorln("error connecting to:", host, err)
		}
		return
	}
	defer func() {
		if !closed {
			remote.Close()
		}
	}()
	log.Debugf("piping %s<->%s ota=%v connOta=%v", conn.RemoteAddr(), host, ota, conn.IsOta())

	if ota {
		go ss.PipeThenCloseOta(conn, remote)
	} else {
		go ss.PipeThenClose(conn, remote)
	}
	ss.PipeThenClose(remote, conn)
	closed = true
	return
}

func getRequest(conn *ss.Conn, auth bool) (host string, ota bool, err error) {
	ss.SetReadTimeout(conn)

	// buf size should at least have the same size with the largest possible
	// request size (when addrType is 3, domain name has at most 256 bytes)
	// 1(addrType) + 1(lenByte) + 256(max length address) + 2(port) + 10(hmac-sha1)
	buf := make([]byte, 270)
	// read till we get possible domain length field
	if _, err = io.ReadFull(conn, buf[:idType+1]); err != nil {
		return
	}

	var reqStart, reqEnd int
	addrType := buf[idType]
	switch addrType & ss.AddrMask {
	case typeIPv4:
		reqStart, reqEnd = idIP0, idIP0+lenIPv4
	case typeIPv6:
		reqStart, reqEnd = idIP0, idIP0+lenIPv6
	case typeDm:
		if _, err = io.ReadFull(conn, buf[idType+1:idDmLen+1]); err != nil {
			return
		}
		reqStart, reqEnd = idDm0, int(idDm0+buf[idDmLen]+lenDmBase)
	default:
		err = fmt.Errorf("addr type %d not supported", addrType&ss.AddrMask)
		return
	}

	if _, err = io.ReadFull(conn, buf[reqStart:reqEnd]); err != nil {
		return
	}

	// Return string for typeIP is not most efficient, but browsers (Chrome,
	// Safari, Firefox) all seems using typeDm exclusively. So this is not a
	// big problem.
	switch addrType & ss.AddrMask {
	case typeIPv4:
		host = net.IP(buf[idIP0 : idIP0+net.IPv4len]).String()
	case typeIPv6:
		host = net.IP(buf[idIP0 : idIP0+net.IPv6len]).String()
	case typeDm:
		host = string(buf[idDm0 : idDm0+buf[idDmLen]])
	}
	// parse port
	port := binary.BigEndian.Uint16(buf[reqEnd-2 : reqEnd])
	host = net.JoinHostPort(host, strconv.Itoa(int(port)))
	// if specified one time auth enabled, we should verify this
	if auth || addrType&ss.OneTimeAuthMask > 0 {
		ota = true
		if _, err = io.ReadFull(conn, buf[reqEnd:reqEnd+lenHmacSha1]); err != nil {
			return
		}
		iv := conn.GetIv()
		key := conn.GetKey()
		actualHmacSha1Buf := ss.HmacSha1(append(iv, key...), buf[:reqEnd])
		if !bytes.Equal(buf[reqEnd:reqEnd+lenHmacSha1], actualHmacSha1Buf) {
			err = fmt.Errorf("verify one time auth failed, iv=%v key=%v data=%v", iv, key, buf[:reqEnd])
			return
		}
	}
	return
}

func (pm *passwdManager) get(port string) (pl *PortListener, ok bool) {
	pm.Lock()
	pl, ok = pm.portListener[port]
	pm.Unlock()
	return
}

func (pm *passwdManager) add(port, password string, listener net.Listener) {
	pm.Lock()
	pm.portListener[port] = &PortListener{password, listener}
	pm.Unlock()
}

func (pm *passwdManager) del(port string) {
	pl, ok := pm.get(port)
	if !ok {
		return
	}
	pl.listener.Close()
	pm.Lock()
	delete(pm.portListener, port)
	pm.Unlock()
}

func (pm *passwdManager) Start() (err error) {
	if pm.config, err = ss.ParseConfig(pm.configFile); err != nil {
		return
	}

	if err = ss.CheckCipherMethod(pm.config.Method); err != nil {
		return
	}

	if err = pm.unifyPortPassword(); err != nil {
		return
	}

	for port, password := range pm.config.PortPassword {
		go pm.Run(port, password, pm.config.Auth)
	}
	return nil
}

func (pm *passwdManager) unifyPortPassword() (err error) {
	if len(pm.config.PortPassword) == 0 { // this handles both nil PortPassword and empty one
		if !pm.enoughOptions() {
			log.Errorln("must specify both port and password")
			return errors.New("not enough options")
		}
		port := strconv.Itoa(pm.config.ServerPort)
		pm.config.PortPassword = map[string]string{port: pm.config.Password}
	} else {
		if pm.config.Password != "" || pm.config.ServerPort != 0 {
			log.Warnln("given port_password, ignore server_port and password option")
		}
	}
	return
}

func (pm *passwdManager) enoughOptions() bool {
	return pm.config.ServerPort != 0 && pm.config.Password != ""
}
