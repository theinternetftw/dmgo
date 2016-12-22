// +build profiling_block

package profiling

import (
	"fmt"

	"github.com/pkg/profile"
)

func Start() Stopper {
	fmt.Println()
	fmt.Println("BLOCK PROFILING BUILD! BLOCK PROFILING BUILD! BLOCK PROFILING BUILD!")
	fmt.Println("BLOCK PROFILING BUILD! BLOCK PROFILING BUILD! BLOCK PROFILING BUILD!")
	fmt.Println("BLOCK PROFILING BUILD! BLOCK PROFILING BUILD! BLOCK PROFILING BUILD!")
	fmt.Println()

	return profile.Start(
		profile.BlockProfile,
		profile.ProfilePath("."),
	)
}
