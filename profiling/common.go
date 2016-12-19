package profiling

// Stopper provides a Stop() method
type Stopper interface {
	Stop()
}

// NopStopper does nothing
type NopStopper struct{}

// Stop implements Stopper
func (n NopStopper) Stop() {}
