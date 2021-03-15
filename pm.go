package lha

import (
	"fmt"
	"os"
)

/***********************************************************
	pm2.c -- extract pmext2 coding
***********************************************************/
/*
  Copyright (c) 1999 Maarten ter Huurne

  Permission is hereby granted, free of charge, to any person
  obtaining a copy of this software and associated documentation files
  (the "Software"), to deal in the Software without restriction,
  including without limitation the rights to use, copy, modify, merge,
  publish, distribute, sublicense, and/or sell copies of the Software,
  and to permit persons to whom the Software is furnished to do so,
  subject to the following conditions:

  The above copyright notice and this permission notice shall be
  included in all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
  EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
  MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
  NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
  BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
  ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
  CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
  SOFTWARE.
*/
var (
	lastupdate, dicsiz1 uint
	gettree1            byte

/* repeated from slide.c */

)

const (
	offset = (0x100 - 2)
)

func (l *Lha) decodeStartPm2() {
	dicsiz1 = (1 << dicbit) - 1
	l.initGetbits()
	histInit()
	nextcount = 0
	lastupdate = 0
	l.getbits(1) /* discard bit */
}

var (
	historyBits [8]int = [8]int{3, 3, 4, 5, 5, 5, 6, 6}
	historyBase [8]int = [8]int{0, 8, 16, 32, 64, 96, 128, 192}
	repeatBits  [6]int = [6]int{3, 3, 5, 6, 7, 0}
	repeatBase  [6]int = [6]int{17, 25, 33, 65, 129, 256}
)

func (l *Lha) decodeCPm2() uint16 {
	/* various admin: */
	for lastupdate != uint(l.loc) {
		histUpdate(dtext[lastupdate])
		lastupdate = (lastupdate + 1) & dicsiz1
	}

	for decodeCount >= int(nextcount) {
		/* Actually it will never loop, because decode_count doesn't grow that fast.
		   However, this is the way LHA does it.
		   Probably other encoding methods can have repeats larger than 256 bytes.
		   Note: LHA puts this code in decode_p...
		*/

		switch nextcount {
		case 0x0000:
			l.maketree1()
			l.maketree2(5)
			nextcount = 0x0400

		case 0x0400:
			l.maketree2(6)
			nextcount = 0x0800

		case 0x0800:
			l.maketree2(7)
			nextcount = 0x1000

		case 0x1000:
			if l.getbits(1) != 0 {
				l.maketree1()
			}
			l.maketree2(8)
			nextcount = 0x2000

		default: /* 0x2000, 0x3000, 0x4000, ... */
			if l.getbits(1) != 0 {
				l.maketree1()
				l.maketree2(8)
			}
			nextcount += 0x1000

		}
	}
	gettree1 = byte(l.tree1Get()) /* value preserved for decode_p */
	if gettree1 >= 29 {
		fmt.Fprintf(os.Stderr, "Bad table")
		return 0
	}

	/* direct value (ret <= UCHAR_MAX) */
	if gettree1 < 8 {
		return uint16(histLookup(historyBase[gettree1] +
			int(l.getbits(byte(historyBits[gettree1])))))
	}
	/* repeats: (ret > UCHAR_MAX) */
	if gettree1 < 23 {
		return offset + 2 + uint16(gettree1-8)
	}

	return offset + uint16(repeatBase[gettree1-23]) +
		l.getbits(byte(repeatBits[gettree1-23]))
}

func (l *Lha) decodePPm2() uint16 {
	/* gettree1 value preserved from decode_c */
	var nbits, delta, gettree2 int
	if gettree1 == 8 { /* 2-byte repeat with offset 0..63 */
		nbits = 6
		delta = 0
	} else {
		if gettree1 < 28 { /* n-byte repeat with offset 0..8191 */
			gettree2 = l.tree2Get()
			if gettree2 == 0 {
				nbits = 6
				delta = 0
			} else { /* 1..7 */
				nbits = 5 + gettree2
				delta = 1 << nbits
			}
		} else { /* 256 bytes repeat with offset 0 */
			nbits = 0
			delta = 0
		}
	}

	return uint16(delta) + l.getbits(byte(nbits))
}
