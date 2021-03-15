package lha

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
func (l *Lha) decodeStartSt0( /*void*/ ) {
	nMax = 286
	maxmatch = Maxmatch
	l.initGetbits()
	initCodeCache()
	np = 1 << (lzhuff3Dicbit - 6)
}

/* ------------------------------------------------------------------------ */
func (l *Lha) encodePSt0(j uint16) {
	var i uint16

	i = j >> 6
	l.putcode(ptLen[i], ptCode[i])
	l.putbits(6, j&0x3f)
}

/* ------------------------------------------------------------------------ */
func (l *Lha) readyMade(method int) {
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
func (l *Lha) encodeStartFix( /*void*/ ) {
	nMax = 314
	maxmatch = 60
	np = 1 << (12 - 6)
	l.initPutbits()
	initCodeCache()
	l.startCDyn()
	l.readyMade(0)
}

/* ------------------------------------------------------------------------ */
func (l *Lha) readTreeC( /*void*/ ) { /* read tree from file */
	var i, c int

	i = 0
	for i < n1 {
		if l.getbits(1) != 0 {
			cLen[i] = byte(l.getbits(byte(lenfield))) + 1
		} else {
			cLen[i] = 0
		}
		i++
		if i == 3 && cLen[0] == 1 && cLen[1] == 1 && cLen[2] == 1 {
			c = int(l.getbits(cbit))
			for i = 0; i < n1; i++ {
				cLen[i] = 0
			}
			for i = 0; i < 4096; i++ {
				cTable[i] = uint16(c)
			}
			return
		}
	}
	makeTable(int16(n1), &cLen, 12, &cTable)
}

/* ------------------------------------------------------------------------ */
func (l *Lha) readTreeP( /*void*/ ) { /* read tree from file */
	var i, c int

	i = 0
	for i < Np_ {
		ptLen[i] = byte(l.getbits(byte(lenfield)))
		i++
		if i == 3 && ptLen[0] == 1 && ptLen[1] == 1 && ptLen[2] == 1 {
			c = int(l.getbits(byte(lzhuff3Dicbit) - 6))
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
func (l *Lha) decodeStartFix( /*void*/ ) {
	nMax = 314
	maxmatch = 60
	l.initGetbits()
	initCodeCache()
	np = 1 << (lzhuff1Dicbit - 6)
	l.startCDyn()
	l.readyMade(0)
	makeTable(int16(np), &ptLen, 8, &ptTable)
}

/* ------------------------------------------------------------------------ */
/* lh3 */
var blocksize uint16 = 0

func (l *Lha) decodeCSt0( /*void*/ ) uint16 {
	var (
		i, j int
	)
	if blocksize == 0 { /* read block head */
		blocksize = l.getbits(byte(bufbits)) /* read block blocksize */
		l.readTreeC()
		if l.getbits(1) != 0 {
			l.readTreeP()
		} else {
			l.readyMade(1)
		}
		makeTable(int16(Np), &ptLen, 8, &ptTable)
	}
	blocksize--
	j = int(cTable[l.peekbits(12)])
	if j < n1 {
		l.fillbuf(cLen[j])
	} else {
		l.fillbuf(12)
		i = int(l.bitbuf)
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
		l.fillbuf(cLen[j] - 12)
	}
	if j == n1-1 {
		j += int(l.getbits(byte(extrabits)))
	}
	return uint16(j)
}

/* ------------------------------------------------------------------------ */
/* lh1, 3 */
func (l *Lha) decodePSt0( /*void*/ ) uint16 {
	var (
		i, j int
	)

	j = int(ptTable[l.peekbits(8)])
	if j < int(np) {
		l.fillbuf(ptLen[j])
	} else {
		l.fillbuf(8)
		i = int(l.bitbuf)
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
		l.fillbuf(ptLen[j] - 8)
	}
	return uint16((j << 6) + int(l.getbits(6)))
}
