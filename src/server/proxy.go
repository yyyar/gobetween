/**
 * proxy.go - proxy utils
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package server

import (
	"../logging"
	"io"
	"math/big"
	"net"
	"time"
)

const (

	/* Buffer size to handle data from socket */
	BUFFER_SIZE = 16 * 1024
)

/**
 * Next r/w operation data counters
 */
type ReadWriteCount struct {

	/* Read bytes count */
	CountRead int

	/* Write bytes count */
	CountWrite int
}

/**
 * Perform copy/proxy data from 'from' to 'to' socket, counting r/w stats and
 * dropping connection if timeout exceeded
 */
func proxy(to net.Conn, from net.Conn, timeout time.Duration) <-chan *big.Int {

	log := logging.For("proxy")

	stats := make(chan ReadWriteCount)
	done := make(chan *big.Int)

	total := big.NewInt(0)

	// Stats collecting goroutine
	go func() {

		if timeout > 0 {
			to.SetReadDeadline(time.Now().Add(timeout))
		}

		for {
			select {
			case rwc, ok := <-stats:

				if !ok {
					done <- total
					return
				}

				if timeout > 0 && rwc.CountRead > 0 {
					to.SetReadDeadline(time.Now().Add(timeout))
				}

				total.Add(total, big.NewInt(int64(rwc.CountRead)))
			}
		}
	}()

	// Run proxy copier
	go func() {
		err := Copy(to, from, stats)
		if err != nil {
			log.Info(err)
		}

		to.Close()
		from.Close()

		// Stop stats collecting goroutine
		close(stats)
	}()

	return done
}

/**
 * It's build by analogy of io.Copy
 */
func Copy(to io.Writer, from io.Reader, ch chan<- ReadWriteCount) error {

	buf := make([]byte, BUFFER_SIZE)
	var err error = nil

	for {
		readN, readErr := from.Read(buf)

		if readN > 0 {

			// send read bytes count
			ch <- ReadWriteCount{CountRead: readN}

			writeN, writeErr := to.Write(buf[0:readN])
			if writeN > 0 {
				// send write bytes count
				ch <- ReadWriteCount{CountWrite: writeN}
			}

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
