package gai

// ServiceProvider defines the contract for registering services into the
// application container, mirroring Laravel's service provider pattern.
type ServiceProvider interface {
	// Register binds things into the container. Called before Boot.
	Register(app *Application)

	// Boot is called after all providers have been registered, allowing
	// providers to do work that depends on other bindings.
	Boot(app *Application)
}
