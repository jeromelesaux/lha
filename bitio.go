package lha

import (
	"fmt"
	"io"
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

var (
	subbitbuf  byte
	bitcount   byte
	infile     io.Reader
	outfile    io.Writer
	unpackable bool
	compsize   int
	origsize   int
	bitbuf     uint16
)

func fillbuf(n byte) error { /* Shift bitbuf n bits left, read n bits */

	for n > bitcount {
		n -= bitcount
		bitbuf = (bitbuf << uint16(bitcount)) + uint16(subbitbuf>>(charBit-bitcount))
		if compsize != 0 {
			compsize--
			c := make([]byte, 1)
			_, err := infile.Read(c)
			if err != nil {
				return fmt.Errorf("cannot read stream")
			}
			subbitbuf = c[0]
		} else {
			subbitbuf = 0
		}
		bitcount = charBit
	}
	bitcount -= n
	bitbuf = (bitbuf << uint16(n)) + uint16(subbitbuf>>(charBit-n))
	subbitbuf <<= n
	return nil
}

func getbits(n byte) uint16 {
	var x uint16

	x = bitbuf >> (2*charBit - n)
	err := fillbuf(n)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fillbuf error :%v\n", err.Error())
	}
	return x
}

func putcode(n byte, x uint16) error { /* Write leftmost n bits of x */

	for n >= bitcount {
		n -= bitcount
		subbitbuf += byte(x) >> (ushrtBit - bitcount)
		x <<= bitcount
		if compsize < origsize {
			var b []byte
			b = append(b, subbitbuf)
			_, err := outfile.Write(b)
			if err != nil {
				return fmt.Errorf("Write error in bitio.c(putcode)")
			}
			compsize++
		} else {
			unpackable = true
		}
		subbitbuf = 0
		bitcount = charBit
	}
	subbitbuf += byte(x) >> (ushrtBit - bitcount)
	bitcount -= n
	return nil
}

func putbits(n byte, x uint16) error { /* Write rightmost n bits of x */
	x <<= ushrtBit - n
	return putcode(n, x)
}

func initGetbits( /* void */ ) error {
	bitbuf = 0
	subbitbuf = 0
	bitcount = 0
	return fillbuf(2 * charBit)
}

func initPutbits( /* void */ ) {
	bitcount = charBit
	subbitbuf = 0
}
