package main

func calcSum(p *[]byte, start, len int) int {
	sum := 0
	for len != 0 {
		sum += int((*p)[start])
		len--
		start++
	}

	return sum & 0xff
}
