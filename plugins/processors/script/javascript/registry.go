package javascript

var sessionHooks = map[string]SessionHook{}

// SessionHook is a function that get invoked when each new Session is created.
type SessionHook func(s Session)

// AddSessionHook registers a SessionHook that gets invoked for each new Session.
func AddSessionHook(name string, mod SessionHook) {
	sessionHooks[name] = mod
}
