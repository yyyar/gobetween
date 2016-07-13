/**
 * proxy.go - proxy utils
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package server

import (
	"../core"
	"../logging"
	"io"
	"net"
	"time"
)

const (

	/* Buffer size to handle data from socket */
	BUFFER_SIZE = 16 * 1024

	/* Interval of pushing aggregated read/write stats */
	PROXY_STATS_PUSH_INTERVAL = 1 * time.Second
)

/**
 * Perform copy/proxy data from 'from' to 'to' socket, counting r/w stats and
 * dropping connection if timeout exceeded
 */
func proxy(to net.Conn, from net.Conn, timeout time.Duration) <-chan core.ReadWriteCount {

	log := logging.For("proxy")

	stats := make(chan core.ReadWriteCount)
	outStats := make(chan core.ReadWriteCount)

	rwcBuffer := core.ReadWriteCount{}
	ticker := time.NewTicker(PROXY_STATS_PUSH_INTERVAL)
	flushed := false

	// Stats collecting goroutine
	go func() {

		if timeout > 0 {
			to.SetReadDeadline(time.Now().Add(timeout))
		}

		for {
			select {
			case <-ticker.C:
				outStats <- rwcBuffer
				flushed = true
			case rwc, ok := <-stats:

				if !ok {
					ticker.Stop()
					if !flushed {
						outStats <- rwcBuffer
					}
					close(outStats)
					return
				}

				if timeout > 0 && rwc.CountRead > 0 {
					to.SetReadDeadline(time.Now().Add(timeout))
				}

				// Remove non blocking
				if flushed {
					rwcBuffer = rwc
				} else {
					rwcBuffer.CountWrite += rwc.CountWrite
					rwcBuffer.CountRead = rwc.CountRead
				}

				flushed = false
			}
		}
	}()

	// Run proxy copier
	go func() {
		err := Copy(to, from, stats)
		// hack to determine normal close. TODO: fix when it will be exposed in golang
		if err != nil && err.(*net.OpError).Err.Error() != "use of closed network connection" {
			log.Warn(err)
		}

		to.Close()
		from.Close()

		// Stop stats collecting goroutine
		close(stats)
	}()

	return outStats
}

/**
 * It's build by analogy of io.Copy
 */
func Copy(to io.Writer, from io.Reader, ch chan<- core.ReadWriteCount) error {

	buf := make([]byte, BUFFER_SIZE)
	var err error = nil

	for {
		readN, readErr := from.Read(buf)

		if readN > 0 {

			writeN, writeErr := to.Write(buf[0:readN])

			// non-blocking stats send
			// may produce innacurate counters because receiving
			// part may miss them. NOTE. Remove non-blocking if will be needed
			//select {
			//case ch <- core.ReadWriteCount{CountRead: readN, CountWrite: writeN}:
			//default:
			//	}

			ch <- core.ReadWriteCount{CountRead: readN, CountWrite: writeN}

			if writeErr != nil {
				err = writeErr
				break
			}

			if readN != writeN {
				err = io.ErrShortWrite
				break
			}
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			err = readErr
			break
		}
	}

	return err
}
