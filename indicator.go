package lha

import "fmt"

var (
	reading_size int
	Quiet        bool
)

func startIndicator(name string, size int, msg []byte, def_indicator_threshold int) {

	if Quiet {
		return
	}

	fmt.Printf("%s\t- ", name)

	reading_size = 0
}

func finishIndicator(name, msg string) {
	if Quiet {
		return
	}
	fmt.Printf("%s\n", msg)

}

func finishIndicator2(name, msg string, pcnt int) {
	if Quiet {
		return
	}
	if pcnt > 100 {
		pcnt = 100
	}
	fmt.Printf("%s\n", msg)
	return
}
