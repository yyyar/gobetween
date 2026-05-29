package core

/**
 * target.go - backend target
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

/**
 * Target host and port
 */
type Target struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

/**
 * Compare to other target
 */
func (t *Target) EqualTo(other Target) bool {
	return t.Host == other.Host &&
		t.Port == other.Port
}

/**
 * Get target full address
 * host:port
 */
func (this *Target) Address() string {
	return this.Host + ":" + this.Port
}

/**
 * To String conversion
 */
func (this *Target) String() string {
	return this.Address()
}
