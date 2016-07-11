/**
 * misc.go
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

package core

/**
 * Next r/w operation data counters
 */
type ReadWriteCount struct {

	/* Read bytes count */
	CountRead int

	/* Write bytes count */
	CountWrite int

	Target Target
}
