package lha

import (
	"fmt"
	"os"
	"unsafe"
)

var (
	left   [2*Nc - 1]uint16
	right  [2*Nc - 1]uint16
	cCode  [Nc]uint16  /* encode */
	ptCode [Npt]uint16 /* encode */

	cTable  [4096]uint16 /* decode */
	ptTable [256]uint16  /* decode */

	cFreq [2*Nc - 1]uint16 /* encode */
	pFreq [2*Np - 1]uint16 /* encode */
	tFreq [2*Nt - 1]uint16 /* encode */

	cLen  [Nc]byte
	ptLen [Npt]byte
	buf   []byte

	outputPos  uint16
	outputMask uint16
	bufsiz     uint16
)

const (
	ushrtMax = ((1 << (2*8 - 1)) - 1)
)

/* ------------------------------------------------------------------------ */
/*                              Encording                                   */
/* ------------------------------------------------------------------------ */
func countTFreq( /*void*/ ) {
	var i, k, n, count uint16

	for i = 0; i < Nt; i++ {
		tFreq[i] = 0
	}
	n = Nc
	for n > 0 && cLen[n-1] == 0 {
		n--
	}
	i = 0
	for i < n {
		k = uint16(cLen[i])
		i++
		if k == 0 {
			count = 1
			for i < n && cLen[i] == 0 {
				i++
				count++
			}
			if count <= 2 {
				tFreq[0] += count
			} else {
				if count <= 18 {
					tFreq[1]++
				} else {
					if count == 19 {
						tFreq[0]++
						tFreq[1]++
					} else {
						tFreq[2]++
					}
				}
			}
		} else {
			tFreq[k+2]++
		}
	}
}

/* ------------------------------------------------------------------------ */
func writePtLen(n, nbit, i_special int16) {
	var i, k int16

	for n > 0 && ptLen[n-1] == 0 {
		n--
	}
	putbits(byte(nbit), uint16(n))
	i = 0
	for i < n {
		k = int16(ptLen[i])
		i++
		if k <= 6 {
			putbits(3, uint16(k))
		} else {
			/* k=7 -> 1110  k=8 -> 11110  k=9 -> 111110 ... */
			putbits(byte(k)-3, ushrtMax<<1)
		}
		if i == i_special {
			for i < 6 && ptLen[i] == 0 {
				i++
			}
			putbits(2, uint16(i)-3)
		}
	}
}

/* ------------------------------------------------------------------------ */
func writeCLen( /*void*/ ) {
	var i, k, n, count uint16

	n = Nc
	for n > 0 && cLen[n-1] == 0 {
		n--
	}
	putbits(cbit, n)
	i = 0
	for i < n {
		k = uint16(cLen[i])
		i++
		if k == 0 {
			count = 1
			for i < n && cLen[i] == 0 {
				i++
				count++
			}
			if count <= 2 {
				for k = 0; k < count; k++ {
					putcode(ptLen[0], ptCode[0])
				}
			} else if count <= 18 {
				putcode(ptLen[1], ptCode[1])
				putbits(4, count-3)
			} else {
				if count == 19 {
					putcode(ptLen[0], ptCode[0])
					putcode(ptLen[1], ptCode[1])
					putbits(4, 15)
				} else {
					putcode(ptLen[2], ptCode[2])
					putbits(cbit, count-20)
				}
			}
		} else {
			putcode(ptLen[k+2], ptCode[k+2])
		}
	}
}

/* ------------------------------------------------------------------------ */
func encodeC(c int16) {
	putcode(cLen[c], cCode[c])
}

/* ------------------------------------------------------------------------ */
func encodeP(p uint16) {
	var c, q uint16

	c = 0
	q = p
	for q != 0 {
		q >>= 1
		c++
	}
	putcode(ptLen[c], ptCode[c])
	if c > 1 {
		putbits(byte(c)-1, p)
	}
}

/* ------------------------------------------------------------------------ */
func sendBlock( /* void */ ) {
	var flags byte
	var i, k, root, pos, size uint16

	root = uint16(makeTree(int(Nc), (*[]uint16)(unsafe.Pointer(&cFreq)), (*[]byte)(unsafe.Pointer(&cLen)), (*[]uint16)(unsafe.Pointer(&cCode))))
	size = cFreq[root]
	putbits(16, size)
	if root >= Nc {
		countTFreq()
		root = uint16(makeTree(int(Nt), (*[]uint16)(unsafe.Pointer(&tFreq)), (*[]byte)(unsafe.Pointer(&ptLen)), (*[]uint16)(unsafe.Pointer(&ptCode))))
		if root >= Nt {
			writePtLen(int16(Nt), int16(tbit), 3)
		} else {
			putbits(tbit, 0)
			putbits(tbit, root)
		}
		writeCLen()
	} else {
		putbits(tbit, 0)
		putbits(tbit, 0)
		putbits(cbit, 0)
		putbits(cbit, root)
	}
	root = uint16(makeTree(int(np), (*[]uint16)(unsafe.Pointer(&pFreq)), (*[]byte)(unsafe.Pointer(&ptLen)), (*[]uint16)(unsafe.Pointer(&ptCode))))
	if root >= uint16(np) {
		writePtLen(int16(np), int16(pbit), -1)
	} else {
		putbits(pbit, 0)
		putbits(pbit, root)
	}
	pos = 0
	for i = 0; i < size; i++ {
		if i%uint16(charBit) == 0 {
			flags = buf[pos]
			pos++
		} else {
			flags <<= 1
		}
		if flags&(1<<(charBit-1)) != 0 {
			encodeC(int16(buf[pos]) + int16(1<<charBit))
			pos++
			k = uint16(buf[pos]) << charBit
			pos++
			k += uint16(buf[pos])
			pos++
			encodeP(k)
		} else {
			encodeC(int16(buf[pos]))
			pos++
		}
		if unpackable {
			return
		}
	}
	for i = 0; i < Nc; i++ {
		cFreq[i] = 0
	}
	for i = 0; i < uint16(np); i++ {
		pFreq[i] = 0
	}
}

/* ------------------------------------------------------------------------ */
/* lh4, 5, 6, 7 */
var cpos uint16

func outputSt1(c, p uint16) {

	outputMask >>= 1
	if outputMask == 0 {
		outputMask = 1 << (charBit - 1)
		if outputPos >= bufsiz-3*uint16(charBit) {
			sendBlock()
			if unpackable {
				return
			}
			outputPos = 0
		}
		cpos = outputPos
		outputPos++
		buf[cpos] = 0
	}
	buf[outputPos] = byte(c)
	outputPos++
	cFreq[c]++
	if c >= (1 << charBit) {
		buf[cpos] |= byte(outputMask)
		buf[outputPos] = byte(p >> charBit)
		outputPos++
		buf[outputPos] = byte(p)
		outputPos++
		c = 0
		for p != 0 {
			p >>= 1
			c++
		}
		pFreq[c]++
	}
}

/* ------------------------------------------------------------------------ */
func allocBuf( /* void */ ) []byte {
	bufsiz = 16 * 1024 * 2 /* 65408U; */ /* t.okamoto */
	/* for ((buf = (unsigned char *) malloc(bufsiz)) == NULL) {
	    bufsiz = (bufsiz / 10) * 9;
	    if (bufsiz < 4 * 1024)
	        fatal_error("Not enough memory");
	}*/
	return buf
}

/* ------------------------------------------------------------------------ */
/* lh4, 5, 6, 7 */
func encodeStartSt1( /* void */ ) {
	var i int

	switch int(dicbit) {
	case lzhuff4Dicbit:
	case lzhuff5Dicbit:
		pbit = 4
		np = uint(lzhuff5Dicbit) + 1
	case lzhuff6Dicbit:
		pbit = 5
		np = uint(lzhuff6Dicbit) + 1
	case lzhuff7Dicbit:
		pbit = 5
		np = uint(lzhuff7Dicbit) + 1
	default:
		fmt.Fprintf(os.Stderr, "Cannot use %d bytes dictionary", 1<<dicbit)
	}

	for i = 0; i < int(Nc); i++ {
		cFreq[i] = 0
	}
	for i = 0; i < Np; i++ {
		pFreq[i] = 0
	}

	outputPos = 0
	outputMask = 0
	initPutbits()
	initCodeCache()
	buf[0] = 0
}

/* ------------------------------------------------------------------------ */
/* lh4, 5, 6, 7 */
func encodeEndSt1( /* void */ ) {
	if !unpackable {
		sendBlock()
		putbits(charBit-1, 0) /* flush remaining bits */
	}
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func min16(a, b int16) int16 {
	if a > b {
		return b
	}
	return a
}

func peekbits(n int) int {
	return int(bitbuf)>>(2)*8 - (n)
}

/* ------------------------------------------------------------------------ */
/*                              decoding                                    */
/* ------------------------------------------------------------------------ */
func readPtLen(nn, nbit, i_special int16) {

	var i, c, n int

	n = int(getbits(byte(nbit)))
	if n == 0 {
		c = int(getbits(byte(nbit)))
		for i = 0; i < int(nn); i++ {
			ptLen[i] = 0
		}
		for i = 0; i < 256; i++ {
			ptTable[i] = uint16(c)
		}
	} else {
		i = 0
		for i < min(n, int(Npt)) {
			c = peekbits(3)
			if c != 7 {
				fillbuf(3)
			} else {
				mask := 1 << (16 - 4)
				for mask&int(bitbuf) != 0 {
					mask >>= 1
					c++
				}
				fillbuf(byte(c) - 3)
			}

			ptLen[i] = byte(c)
			i++
			if i == int(i_special) {
				c = int(getbits(2))
				c--
				for c >= 0 && i < int(Npt) {
					ptLen[i] = 0
					c--
					i++
				}
			}
		}
		for i < int(nn) {
			ptLen[i] = 0
			i++
		}
		makeTable(nn, (*[]byte)(unsafe.Pointer(&ptLen)), 8, (*[]uint16)(unsafe.Pointer(&ptTable)))
	}
}

/* ------------------------------------------------------------------------ */
func readCLen( /* void */ ) {
	var i, c, n int16

	n = int16(getbits(cbit))
	if n == 0 {
		c = int16(getbits(cbit))
		for i = 0; i < int16(Nc); i++ {
			cLen[i] = 0
		}
		for i = 0; i < 4096; i++ {
			cTable[i] = uint16(c)
		}
	} else {
		i = 0
		for i < min16(n, int16(Nc)) {
			c = int16(ptTable[peekbits(8)])
			if c >= int16(Nt) {
				var mask uint16 = 1 << (16 - 9)
				for c >= int16(Nt) && (mask != 0 || c != int16(left[c])) {
					if (bitbuf & mask) != 0 {
						c = int16(right[c])
					} else {
						c = int16(left[c])
					}
					mask >>= 1
				} // for (c >= NT && (mask || c != left[c])); /* CVE-2006-4338 */
			}
			fillbuf(ptLen[c])
			if c <= 2 {
				if c == 0 {
					c = 1
				} else {
					if c == 1 {
						c = int16(getbits(4)) + 3
					} else {
						c = int16(getbits(cbit)) + 20
					}
				}
				c--
				for c >= 0 {
					cLen[i] = 0
					i++
					c--
				}
			} else {
				cLen[i] = byte(c) - 2
				i++
			}
		}
		for i < int16(Nc) {
			cLen[i] = 0
			i++
		}
		makeTable(int16(Nc), (*[]byte)(unsafe.Pointer(&cLen)), 12, (*[]uint16)(unsafe.Pointer(&cTable)))
	}
}

/* ------------------------------------------------------------------------ */
/* lh4, 5, 6, 7 */
func decodeCSt1( /*void*/ ) uint16 {
	var j, mask uint16

	if blocksize == 0 {
		blocksize = getbits(16)
		readPtLen(int16(Nt), int16(tbit), 3)
		readCLen()
		readPtLen(int16(np), int16(pbit), -1)
	}
	blocksize--
	j = cTable[peekbits(12)]
	if j < Nc {
		fillbuf(cLen[j])
	} else {
		fillbuf(12)
		mask = 1 << (16 - 1)
		for j >= Nc && (mask != 0 || j != left[j]) {
			if (bitbuf & mask) != 0 {
				j = right[j]
			} else {
				j = left[j]
			}
			mask >>= 1
			//for (j >= NC && (mask || j != left[j])); /* CVE-2006-4338 */
		} //for (j >= NC && (mask || j != left[j])); /* CVE-2006-4338 */
		fillbuf(cLen[j] - 12)
	}
	return j
}

/* ------------------------------------------------------------------------ */
/* lh4, 5, 6, 7 */
func decodePSt1( /* void */ ) uint16 {
	var j, mask uint16

	j = ptTable[peekbits(8)]
	if j < uint16(np) {
		fillbuf(ptLen[j])
	} else {
		fillbuf(8)
		mask = 1 << (16 - 1)
		for uint(j) >= np && (mask != 0 || j != left[j]) {
			if (bitbuf & mask) != 0 {
				j = right[j]
			} else {
				j = left[j]
			}
			mask >>= 1
			//for (j >= np && (mask || j != left[j])); /* CVE-2006-4338 */
		} //for (j >= np && (mask || j != left[j])); /* CVE-2006-4338 */
		fillbuf(ptLen[j] - 8)
	}
	if j != 0 {
		j = (1 << (j - 1)) + getbits(byte(j)-1)
	}
	return j
}

/* ------------------------------------------------------------------------ */
/* lh4, 5, 6, 7 */
func decodeStartSt1( /* void */ ) {
	switch int(dicbit) {
	case lzhuff4Dicbit:
	case lzhuff5Dicbit:
		pbit = 4
		np = uint(lzhuff5Dicbit) + 1
		break
	case lzhuff6Dicbit:
		pbit = 5
		np = uint(lzhuff6Dicbit) + 1
		break
	case lzhuff7Dicbit:
		pbit = 5
		np = uint(lzhuff7Dicbit) + 1
		break
	default:
		fmt.Fprintf(os.Stderr, "Cannot use %d bytes dictionary", 1<<dicbit)
	}

	initGetbits()
	initCodeCache()
	blocksize = 0
}
