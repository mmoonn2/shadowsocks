package shadowsocks

import (
	"io"
	"net"

	ss "shadowsocks/shadowsocks/shadowsocks"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-units"
	"github.com/juju/ratelimit"
)

// RateLimit bindwidth limit
func RateLimit(sizeStr string, srcConn *ss.Conn) (dstConn *ss.Conn, err error) {

	var size int64
	if size, err = units.RAMInBytes(sizeStr); err != nil {
		return
	}

	log.Debugf("RateLimit ===>>>>1 rate limit:%s is %d byte", sizeStr, size)
	// Destination
	// var dst = &bytes.Buffer{}

	// raw := ioutil.NopCloser(srcConn)

	c1, c2 := net.Pipe()
	ss.NewConn(c2, srcConn.Cipher.Copy())

	// Bucket adding size byte every second, holding max size byte
	bucket := ratelimit.NewBucketWithRate(float64(size), size)

	// Copy source to destination, but wrap our reader with rate limited one
	go func() {
		if _, err = io.Copy(c1, ratelimit.Reader(srcConn, bucket)); err != nil {
			log.Errorln("RateLimit ===>>>>3 err", err)
		}
	}()

	log.Debugf("RateLimit ===>>>>2 rate limit:%s is %d byte", sizeStr, size)
	_, err = io.Copy(srcConn, ratelimit.Reader(c1, bucket))

	return
}
