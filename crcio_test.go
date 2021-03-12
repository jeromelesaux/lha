package lha

import "testing"

func TestCalcCrc(t *testing.T) {

	MakeCrcTable()
	var crc uint = 39177
	var c byte = 246

	crc2 := updateCrc(crc, c)

	if crc2 != 16601 {
		t.Fatal()
	}
}
