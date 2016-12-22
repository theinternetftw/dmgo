// +build profiling_mem

package profiling

import (
	"fmt"

	"github.com/pkg/profile"
)

func Start() Stopper {
	fmt.Println()
	fmt.Println("MEM PROFILING BUILD! MEM PROFILING BUILD! MEM PROFILING BUILD!")
	fmt.Println("MEM PROFILING BUILD! MEM PROFILING BUILD! MEM PROFILING BUILD!")
	fmt.Println("MEM PROFILING BUILD! MEM PROFILING BUILD! MEM PROFILING BUILD!")
	fmt.Println()

	return profile.Start(
		profile.MemProfile,
		profile.ProfilePath("."),
	)
}
