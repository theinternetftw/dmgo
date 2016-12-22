// +build profiling_cpu

package profiling

import (
	"fmt"

	"github.com/pkg/profile"
)

func Start() Stopper {
	fmt.Println()
	fmt.Println("CPU PROFILING BUILD! CPU PROFILING BUILD! CPU PROFILING BUILD!")
	fmt.Println("CPU PROFILING BUILD! CPU PROFILING BUILD! CPU PROFILING BUILD!")
	fmt.Println("CPU PROFILING BUILD! CPU PROFILING BUILD! CPU PROFILING BUILD!")
	fmt.Println()

	return profile.Start(
		profile.CPUProfile,
		profile.ProfilePath("."),
	)
}
