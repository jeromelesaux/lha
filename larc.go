package lha

import "io"

/* ------------------------------------------------------------------------ */
/* LHa for UNIX                                                             */
/*              larc.c -- extra *.lzs                                       */
/*                                                                          */
/*      Modified                Nobutaka Watazaki                           */
/*                                                                          */
/*  Ver. 1.14   Source All chagned              1995.01.14  N.Watazaki      */
/* ------------------------------------------------------------------------ */

/* ------------------------------------------------------------------------ */
var ()

/* ------------------------------------------------------------------------ */
/* lzs */

func (l *Lha) decodeCLzs( /*void*/ ) uint16 {
	if l.getbits(1) != 0 {
		return l.getbits(8)
	} else {
		l.matchpos = int(l.getbits(11))
		return l.getbits(4) + 0x100
	}
}

/* ------------------------------------------------------------------------ */
/* lzs */

func (l *Lha) decodePLzs( /*void*/ ) uint16 {
	return (l.loc - uint16(l.matchpos) - uint16(magic0)) & 0x7ff
}

/* ------------------------------------------------------------------------ */
/* lzs */
func (l *Lha) decodeStartLzs( /*void*/ ) {
	_ = l.initGetbits()
	initCodeCache()
}

/* ------------------------------------------------------------------------ */
/* lz5 */
func (l *Lha) decodeCLz5( /*void*/ ) uint16 {
	var c int
	b := make([]byte, 1)
	if l.flagcnt == 0 {
		l.flagcnt = 8
		_, err := l.infile.Read(b)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				return 0
			}
		}
		l.mFlag = int(b[0])
	}
	l.flagcnt--
	_, _ = l.infile.Read(b)
	c = int(b[0])
	if (l.mFlag & 1) == 0 {
		l.matchpos = c
		_, err := l.infile.Read(b)
		if err != nil {
			if err != io.ErrUnexpectedEOF {
				return 0
			}
		}
		c = int(b[0]) //@TODO add EOF handler
		l.matchpos += (c & 0xf0) << 4
		c &= 0x0f
		c += 0x100
	}
	l.mFlag >>= 1
	return uint16(c)
}

/* ------------------------------------------------------------------------ */
/* lz5 */
func (l *Lha) decodePLz5( /*void*/ ) uint16 {
	return (l.loc - uint16(l.matchpos) - uint16(magic5)) & 0xfff
}

/* ------------------------------------------------------------------------ */
/* lz5 */
func (l *Lha) decodeStartLz5( /*void*/ ) {
	var i int

	l.flagcnt = 0
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
