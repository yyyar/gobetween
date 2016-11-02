/**
 * session.go - udp "session"
 *
 * @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
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

	/* stats handler */
	statsHandler *stats.Handler

	/* scheduler */
	scheduler *scheduler.Scheduler

	/* Session backend */
	backend *core.Backend

	/* connection to previously elected backend */
	backendConn *net.UDPConn

	/* Time where session was touched last time */
	lastUpdated time.Time

	/* ----- channels ----- */

	/* touch channel */
	touchC chan bool

	/* stop channel */
	stopC chan bool
}

/**
 * Start session
 */
func (c *session) start(serverConn *net.UDPConn, sessionManager *sessionManager, timeout time.Duration, maxResponses *int) {

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
			case <-c.touchC:
				c.lastUpdated = time.Now()
			case now := <-ticker.C:
				if c.lastUpdated.Add(timeout).Before(now) {
					c.stop()
				}
			}
		}
	}()

	/**
	 * Proxy data from backend to client
	 */
	go func() {
		var buf = make([]byte, UDP_PACKET_SIZE)
		var responses = 0
		for {
			n, _, err := c.backendConn.ReadFromUDP(buf)
			responses++

			if err != nil {
				log.Debug("Closing client ", c.clientAddr.String())
				break
			}

			c.touch()
			c.scheduler.IncrementRx(*c.backend, uint(n))
			serverConn.WriteToUDP(buf[0:n], c.clientAddr)
			if maxResponses != nil && responses >= *maxResponses {
				c.stop()
			}
		}
	}()
}

/**
 * Writes data to session backend
 */
func (c *session) send(buf []byte) {
	go func() {
		c.backendConn.Write(buf)
		c.touch()
		n := len(buf)
		c.scheduler.IncrementTx(*c.backend, uint(n))
	}()
}

/**
 * Touches session
 */
func (c *session) touch() {
	go func() {
		c.touchC <- true
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
