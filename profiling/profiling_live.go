// +build profiling_live

package profiling

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

const profileAddr = "127.0.0.1:7070"

func Start() Stopper {
	fmt.Println()
	fmt.Println("LIVE PROFILING BUILD! LIVE PROFILING BUILD! LIVE PROFILING BUILD!")
	fmt.Println("LIVE PROFILING BUILD! LIVE PROFILING BUILD! LIVE PROFILING BUILD!")
	fmt.Println("LIVE PROFILING BUILD! LIVE PROFILING BUILD! LIVE PROFILING BUILD!")
	fmt.Println("running at http://" + profileAddr + "/debug/pprof/")
	fmt.Println()

	go func() {
		fmt.Println(http.ListenAndServe(profileAddr, nil))
	}()

	return NopStopper{}
}
