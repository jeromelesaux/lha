package lha

import "unsafe"

const (
	Np_      = (8 * 1024 / 64)
	Np2_     = (Np_*2 - 1)
	lenfield = 4 /* bit size of length field for tree output */
)

/* ------------------------------------------------------------------------ */
var (
	np uint
)
var fixed [2][16]int = [2][16]int{
	{3, 0x01, 0x04, 0x0c, 0x18, 0x30, 0},             /* old compatible */
	{2, 0x01, 0x01, 0x03, 0x06, 0x0D, 0x1F, 0x4E, 0}} /* 8K buf */

/* ------------------------------------------------------------------------ */
/* lh3 */
func decodeStartSt0( /*void*/ ) {
	nMax = 286
	maxmatch = Maxmatch
	initGetbits()
	initCodeCache()
	np = 1 << (lzhuff3Dicbit - 6)
}

/* ------------------------------------------------------------------------ */
func encodePSt0(j uint16) {
	var i uint16

	i = j >> 6
	putcode(ptLen[i], ptCode[i])
	putbits(6, j&0x3f)
}

/* ------------------------------------------------------------------------ */
func readyMade(method int) {
	var i, j int
	var code, weight uint
	var tbl int

	//tbl = fixed[method];
	//j = *tbl++;
	j = fixed[method][tbl]
	tbl++
	weight = 1 << (16 - j)
	code = 0
	for i = 0; i < int(np); i++ {

		for fixed[method][tbl] == i {
			j++
			tbl++
			weight >>= 1
		}
		ptLen[i] = byte(j)
		ptCode[i] = uint16(code)
		code += weight
	}
}

/* ------------------------------------------------------------------------ */
/* lh1 */
func encodeStartFix( /*void*/ ) {
	nMax = 314
	maxmatch = 60
	np = 1 << (12 - 6)
	initPutbits()
	initCodeCache()
	startCDyn()
	readyMade(0)
}

/* ------------------------------------------------------------------------ */
func readTreeC( /*void*/ ) { /* read tree from file */
	var i, c int

	i = 0
	for i < n1 {
		if getbits(1) != 0 {
			cLen[i] = byte(getbits(byte(lenfield))) + 1
		} else {
			cLen[i] = 0
		}
		i++
		if i == 3 && cLen[0] == 1 && cLen[1] == 1 && cLen[2] == 1 {
			c = int(getbits(cbit))
			for i = 0; i < n1; i++ {
				cLen[i] = 0
			}
			for i = 0; i < 4096; i++ {
				cTable[i] = uint16(c)
			}
			return
		}
	}
	makeTable(int16(n1), (*[]byte)(unsafe.Pointer(&cLen)), 12, (*[]uint16)(unsafe.Pointer(&cTable)))
}

/* ------------------------------------------------------------------------ */
func readTreeP( /*void*/ ) { /* read tree from file */
	var i, c int

	i = 0
	for i < Np_ {
		ptLen[i] = byte(getbits(byte(lenfield)))
		i++
		if i == 3 && ptLen[0] == 1 && ptLen[1] == 1 && ptLen[2] == 1 {
			c = int(getbits(byte(lzhuff3Dicbit) - 6))
			for i = 0; i < Np_; i++ {
				ptLen[i] = 0
			}
			for i = 0; i < 256; i++ {
				ptTable[i] = uint16(c)
			}
			return
		}
	}
}

/* ------------------------------------------------------------------------ */
/* lh1 */
func decodeStartFix( /*void*/ ) {
	nMax = 314
	maxmatch = 60
	initGetbits()
	initCodeCache()
	np = 1 << (lzhuff1Dicbit - 6)
	startCDyn()
	readyMade(0)
	makeTable(int16(np), (*[]byte)(unsafe.Pointer(&ptLen)), 8, (*[]uint16)(unsafe.Pointer(&ptTable)))
}

/* ------------------------------------------------------------------------ */
/* lh3 */
var blocksize uint16 = 0

func decodeCSt0( /*void*/ ) uint16 {
	var (
		i, j int
	)
	if blocksize == 0 { /* read block head */
		blocksize = getbits(byte(bufbits)) /* read block blocksize */
		readTreeC()
		if getbits(1) != 0 {
			readTreeP()
		} else {
			readyMade(1)
		}
		makeTable(int16(Np), (*[]byte)(unsafe.Pointer(&ptLen)), 8, (*[]uint16)(unsafe.Pointer(&ptTable)))
	}
	blocksize--
	j = int(cTable[peekbits(12)])
	if j < n1 {
		fillbuf(cLen[j])
	} else {
		fillbuf(12)
		i = int(bitbuf)
		for {
			if i < 0 {
				j = int(right[j])
			} else {
				j = int(left[j])
			}
			i <<= 1
			if j < n1 {
				break
			}
		} //while (j >= N1);
		fillbuf(cLen[j] - 12)
	}
	if j == n1-1 {
		j += int(getbits(byte(extrabits)))
	}
	return uint16(j)
}

/* ------------------------------------------------------------------------ */
/* lh1, 3 */
func decodePSt0( /*void*/ ) uint16 {
	var (
		i, j int
	)

	j = int(ptTable[peekbits(8)])
	if j < int(np) {
		fillbuf(ptLen[j])
	} else {
		fillbuf(8)
		i = int(bitbuf)
		for {
			if i < 0 {
				j = int(right[j])
			} else {
				j = int(left[j])
			}
			i <<= 1
			if j < int(np) {
				break
			}
		}
		fillbuf(ptLen[j] - 8)
	}
	return uint16((j << 6) + int(getbits(6)))
}
