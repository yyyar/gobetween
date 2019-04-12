package core

/**
 * Service is a global facility that could be Enabled or Disabled for a number
 * of core.Server instances, depending on their configration. See services/registry
 * for exact examples.
 */
type Service interface {
	/**
	 * Enable service for Server
	 */
	Enable(Server) error

	/**
	 * Disable service for Server
	 */
	Disable(Server) error
}
