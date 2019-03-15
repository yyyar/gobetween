package stats

/**
 * store.go - stats storage and getter
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"sync"
)

/**
 * Handlers Store
 */
var Store = struct {
	sync.RWMutex
	handlers map[string]*Handler
}{handlers: make(map[string]*Handler)}

/**
 * Get stats for the server
 */
func GetStats(name string) interface{} {

	Store.RLock()
	defer Store.RUnlock()

	handler, ok := Store.handlers[name]
	if !ok {
		return nil
	}
	return handler.latestStats // TODO: syncronize?
}
