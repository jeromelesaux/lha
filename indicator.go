package lha

import "fmt"

var (
	reading_size int
	quiet        bool
)

func startIndicator(name string, size int, msg []byte, def_indicator_threshold int) {

	if quiet {
		return
	}

	fmt.Printf("%s\t- ", name)

	reading_size = 0
}

func finishIndicator(name, msg string) {
	if quiet {
		return
	}
	fmt.Printf("%s\n", msg)

}
