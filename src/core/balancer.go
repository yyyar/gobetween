package core

/**
 * Balancer interface
 */
type Balancer interface {

	/**
	 * Elect backend based on Balancer implementation
	 */
	Elect(Context, []*Backend) (*Backend, error)
}
