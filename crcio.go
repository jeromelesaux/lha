package main

func calcCrc(crc uint, p *[]byte, pIndex, n uint) uint {
	for n > 0 {
		crc = updateCrc(&crc, uint((*p)[int(pIndex)]))
		pIndex++
		n--
	}
	return crc
}
