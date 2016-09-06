/**
 * session.go - udp "session"
 *
 * @author Illarion Kovalchuk
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package udp

import (
	"../../core"
	"../../logging"
	"../../stats"
	"../scheduler"
	"net"
	"time"
)

/**
 * Emulates UDP "session"
 */
type session struct {

	/* client address */
	clientAddr *net.UDPAddr

	statsHandler *stats.Handler
	scheduler    *scheduler.Scheduler
	backend      *core.Backend

	/* connection to previously elected backend */
	backendConn *net.UDPConn

	lastUpdated time.Time

	updC  chan bool
	stopC chan bool
}

/**
 * Start session
 */
func (c *session) start(serverConn *net.UDPConn, sessionManager *sessionManager, timeout time.Duration, maxPackets *int) {

	log := logging.For("udp/session")

	go func() {

		ticker := time.NewTicker(timeout)
		for {
			select {
			case <-c.stopC:
				ticker.Stop()
				c.scheduler.DecrementConnection(*c.backend)
				c.backendConn.Close()
				sessionManager.remove(c)
			case <-c.updC:
				c.lastUpdated = time.Now()
			case now := <-ticker.C:
				if c.lastUpdated.Add(timeout).Before(now) {
					c.stop()
				}
			}
		}
	}()

	go func() {
		var buf = make([]byte, UDP_PACKET_SIZE)
		var packets = 0
		for {
			n, _, err := c.backendConn.ReadFromUDP(buf)
			packets++

			if err != nil {
				log.Debug("Closing client ", c.clientAddr.String())
				break
			}
			c.markUpdated()
			c.scheduler.IncrementRx(*c.backend, uint(n))
			serverConn.WriteToUDP(buf[0:n], c.clientAddr)
			if maxPackets != nil && packets >= *maxPackets {
				c.stop()
			}
		}
	}()
}

/**
 * Touches session
 */
func (c *session) markUpdated() {
	go func() {
		c.updC <- true
	}()
}

func (c *session) sendToBackend(buf []byte) {
	go func() {
		c.backendConn.Write(buf)
		c.markUpdated()
		n := len(buf)
		c.scheduler.IncrementTx(*c.backend, uint(n))
	}()
}

/**
 * Stops session
 */
func (c *session) stop() {
	go func() {
		c.stopC <- true
	}()
}
