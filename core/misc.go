package core

/**
 * misc.go
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

/**
 * Next r/w operation data counters
 */
type ReadWriteCount struct {

	/* Read bytes count */
	CountRead uint

	/* Write bytes count */
	CountWrite uint

	Target Target
}

func (this ReadWriteCount) IsZero() bool {
	return this.CountRead == 0 && this.CountWrite == 0
}
