package lha

import (
	"fmt"
	"os"
)

/* ------------------------------------------------------------------------ */
/* LHa for UNIX                                                             */
/*              bitio.c -- bit stream                                       */
/*                                                                          */
/*      Modified                Nobutaka Watazaki                           */
/*                                                                          */
/*  Ver. 1.14   Source All chagned              1995.01.14  N.Watazaki      */
/*              Separated from crcio.c          2002.10.26  Koji Arai       */
/* ------------------------------------------------------------------------ */

func (l *Lha) fillbuf(n byte) error { /* Shift bitbuf n bits left, read n bits */

	for n > l.bitcount {
		n -= l.bitcount
		l.bitbuf = (l.bitbuf << uint16(l.bitcount)) + uint16(l.subbitbuf>>(charBit-l.bitcount))
		if l.compsize != 0 {
			l.compsize--
			c := make([]byte, 1)
			_, err := l.infile.Read(c)
			if err != nil {
				return fmt.Errorf("cannot read stream")
			}
			l.subbitbuf = c[0]
		} else {
			l.subbitbuf = 0
		}
		l.bitcount = charBit
	}
	l.bitcount -= n
	l.bitbuf = (l.bitbuf << uint16(n)) + uint16(l.subbitbuf>>(charBit-n))
	l.subbitbuf <<= n
	return nil
}

func (l *Lha) getbits(n byte) uint16 {
	var x uint16

	x = l.bitbuf >> (2*charBit - n)
	err := l.fillbuf(n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fillbuf error :%v\n", err.Error())
	}
	return x
}

func (l *Lha) putcode(n byte, x uint16) error { /* Write leftmost n bits of x */

	for n >= l.bitcount {

		n -= l.bitcount
		l.subbitbuf += byte(x >> (ushrtBit - l.bitcount))
		x <<= l.bitcount
		if l.compsize < l.origsize {
			var b []byte

			b = append(b, l.subbitbuf)
			_, err := l.outfile.Write(b)
			if err != nil {
				return fmt.Errorf("Write error in bitio.c(putcode)")
			}
			l.compsize++
		} else {
			l.unpackable = true
		}
		l.subbitbuf = 0
		l.bitcount = charBit
	}
	l.subbitbuf += byte(x >> (ushrtBit - l.bitcount))
	l.bitcount -= n
	return nil
}

func (l *Lha) putbits(n byte, x uint16) error { /* Write rightmost n bits of x */
	x <<= ushrtBit - n
	return l.putcode(n, x)
}

func (l *Lha) initGetbits( /* void */ ) error {
	l.bitbuf = 0
	l.subbitbuf = 0
	l.bitcount = 0
	return l.fillbuf(2 * charBit)
}

func (l *Lha) initPutbits( /* void */ ) {
	l.bitcount = charBit
	l.subbitbuf = 0
}
