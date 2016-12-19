// +build !profiling_live
// +build !profiling_cpu
// +build !profiling_mem
// +build !profiling_block

package profiling

// Start does nothing in this case, as no profiler is enabled.
func Start() Stopper {
	return NopStopper{}
}
