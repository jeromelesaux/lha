package lha

/* ------------------------------------------------------------------------ */
/* LHa for UNIX                                                             */
/*              larc.c -- extra *.lzs                                       */
/*                                                                          */
/*      Modified                Nobutaka Watazaki                           */
/*                                                                          */
/*  Ver. 1.14   Source All chagned              1995.01.14  N.Watazaki      */
/* ------------------------------------------------------------------------ */

/* ------------------------------------------------------------------------ */
var (
	mFlag, flagcnt, matchpos int
	loc                      uint16
	ExtractDirectory         string
)

/* ------------------------------------------------------------------------ */
/* lzs */

func decodeCLzs( /*void*/ ) uint16 {
	if getbits(1) != 0 {
		return getbits(8)
	} else {
		matchpos = int(getbits(11))
		return getbits(4) + 0x100
	}
}

/* ------------------------------------------------------------------------ */
/* lzs */

func decodePLzs( /*void*/ ) uint16 {
	return (loc - uint16(matchpos) - uint16(magic0)) & 0x7ff
}

/* ------------------------------------------------------------------------ */
/* lzs */
func decodeStartLzs( /*void*/ ) {
	initGetbits()
	initCodeCache()
}

/* ------------------------------------------------------------------------ */
/* lz5 */
func decodeCLz5( /*void*/ ) uint16 {
	var c int
	b := make([]byte, 1)
	if flagcnt == 0 {
		flagcnt = 8
		infile.Read(b)
		mFlag = int(b[0])
	}
	flagcnt--
	infile.Read(b)
	c = int(b[0])
	if (mFlag & 1) == 0 {
		matchpos = c
		infile.Read(b)
		c = int(b[0])
		matchpos += (c & 0xf0) << 4
		c &= 0x0f
		c += 0x100
	}
	mFlag >>= 1
	return uint16(c)
}

/* ------------------------------------------------------------------------ */
/* lz5 */
func decodePLz5( /*void*/ ) uint16 {
	return (loc - uint16(matchpos) - uint16(magic5)) & 0xfff
}

/* ------------------------------------------------------------------------ */
/* lz5 */
func decodeStartLz5( /*void*/ ) {
	var i int

	flagcnt = 0
	for i = 0; i < 256; i++ {
		for j := 0; j < 13; j++ {
			dtext[i*13+18+j] = byte(i)
		}
		//	memset(&dtext[i*13+18], i, 13)
	}
	for i = 0; i < 256; i++ {
		dtext[256*13+18+i] = byte(i)
	}
	for i = 0; i < 256; i++ {
		dtext[256*13+256+18+i] = byte(255 - i)
	}
	for j := 0; j < 128; j++ {
		dtext[256*13+512+18+j] = 0
	}

	for j := 0; j < (128 - 18); j++ {
		dtext[256*13+512+128+18] = ' '
	}

	//memset(&dtext[256*13+512+18], 0, 128)
	//memset(&dtext[256*13+512+128+18], ' ', 128-18)
}
