package lha

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

var (
	dicsiz int
)

const (
	limit = 0x100
)

type matchdata struct {
	len int
	off uint
}

func (l *Lha) decodeMethodStart(method int) {
	switch method - 1 {
	case 0: /* lh1 */
		l.decodeStartFix()
	case 1: /* lh2 */
		l.decodeStartDyn()
	case 2: /* lh3 */
		l.decodeStartSt0()
	case 3: /* lh4 */
		l.decodeStartSt1()
	case 4: /* lh5 */
		l.decodeStartSt1()
	case 5: /* lh6 */
		l.decodeStartSt1()
	case 6: /* lh7 */
		l.decodeStartSt1()
	case 7: /* lzs */
		l.decodeStartLzs()
	case 8: /* lz5 */
		l.decodeStartLz5()
	case 12: /* pm2 */
		l.decodeStartPm2()
	default:
		return
	}
}

func (l *Lha) decodeMethodP(method int) uint16 {
	switch method - 1 {
	case 0: /* lh1 */
		return l.decodePSt0()
	case 1: /* lh2 */
		return l.decodePDyn()
	case 2: /* lh3 */
		return l.decodePSt0()
	case 3: /* lh4 */
		return l.decodePSt1()
	case 4: /* lh5 */
		return l.decodePSt1()
	case 5: /* lh6 */
		return l.decodePSt1()
	case 6: /* lh7 */
		return l.decodePSt1()
	case 7: /* lzs */
		return l.decodePLzs()
	case 8: /* lz5 */
		return l.decodePLz5()
	case 12: /* pm2 */
		return l.decodePPm2()
	default:
		return 0
	}
}

func (l *Lha) decodeMethodC(method int) uint16 {
	switch method - 1 {
	case 0: /* lh1 */
		return l.decodeCDyn()
	case 1: /* lh2 */
		return l.decodeCDyn()
	case 2: /* lh3 */
		return l.decodeCSt0()
	case 3: /* lh4 */
		return l.decodeCSt1()
	case 4: /* lh5 */
		return l.decodeCSt1()
	case 5: /* lh6 */
		return l.decodeCSt1()
	case 6: /* lh7 */
		return l.decodeCSt1()
	case 7: /* lzs */
		return l.decodeCLzs()
	case 8: /* lz5 */
		return l.decodeCLz5()
	case 12: /* pm2 */
		return l.decodeCPm2()
	default:
		return 0
	}
}

func (l *Lha) decode(inter *interfacing) uint {
	var (
		i, c            uint
		dicsiz1, adjust uint
		crc             uint
	)

	l.infile = inter.infile
	l.outfile = inter.outfile
	dicbit = uint16(inter.dicbit)
	l.origsize = inter.original
	l.compsize = inter.packed

	initializeCrc(&crc)
	dicsiz = int(1) << dicbit
	dtext = make([]byte, dicsiz)

	if !l.ExtractBrokenArchive {
		for i := 0; i < len(dtext); i++ {
			dtext[i] = ' '
		}
	}

	/* LHa for UNIX (autoconf) had a fatal bug since version
	   1.14i-ac20030713 (slide.c revision 1.20).

	   This bug is possible to make a broken archive, proper LHA
	   cannot extract it (probably it report CRC error).

	   If the option "--extract-broken-archive" specified, extract
	   the broken archive made by old LHa for UNIX. */

	l.decodeMethodStart(inter.method)
	dicsiz1 = uint(dicsiz) - 1
	adjust = 256 - uint(threshold)
	if (inter.method == LarcMethodNum) || (inter.method == Pmarc2MethodNum) {
		adjust = 256 - 2
	}
	decodeCount = 0
	l.loc = 0
	for decodeCount < l.origsize {
		c = uint(l.decodeMethodC(inter.method))
		if c < 256 {
			if l.DumpLzss {
				b := c
				if !strconv.IsPrint(rune(c)) {
					b = '?'
				}
				fmt.Printf("%d %02x(%c)\n", decodeCount, c, b)

			}
			dtext[l.loc] = byte(c)
			l.loc++
			if int(l.loc) == dicsiz {
				l.fwriteCrc(&crc, dtext, dicsiz, &l.outfile)
				l.loc = 0
			}
			decodeCount++
		} else {
			var match matchdata
			var matchpos uint

			match.len = int(c - adjust)
			match.off = uint(l.decodeMethodP(inter.method)) + 1
			matchpos = (uint(l.loc) - match.off) & dicsiz1
			if l.DumpLzss {
				fmt.Printf("%d <%d %d>\n",
					decodeCount, match.len, decodeCount-int(match.off))
			}

			decodeCount += match.len
			for i = 0; i < uint(match.len); i++ {
				c = uint(dtext[(matchpos+i)&dicsiz1])
				dtext[l.loc] = byte(c)
				l.loc++
				if int(l.loc) == dicsiz {
					l.fwriteCrc(&crc, dtext, dicsiz, &l.outfile)
					l.loc = 0
				}
			}
		}
	}
	if l.loc != 0 {
		l.fwriteCrc(&crc, dtext, int(l.loc), &l.outfile)
	}
	/* usually read size is interface->packed */
	inter.readSize = inter.packed - l.compsize

	return crc
}

func (l *Lha) encodeStart(method int) {
	switch method {
	case 0:
		l.encodeStartFix()
	case 1:
		l.encodeStartSt1()
	case 2:
		l.encodeStartSt1()
	case 3:
		l.encodeStartSt1()
	case 4:
		l.encodeStartSt1()
	case 5:
		l.encodeStartSt1()
	case 6:
		l.encodeStartSt1()
	case 7:
		l.encodeStartSt1()
	}
}

func (l *Lha) encodeEnd(method int) {
	switch method {
	case 0:
		l.encodeEndDyn()
	case 1:
		l.encodeEndSt1()
	case 2:
		l.encodeEndSt1()
	case 3:
		l.encodeEndSt1()
	case 4:
		l.encodeEndSt1()
	case 5:
		l.encodeEndSt1()
	case 6:
		l.encodeEndSt1()
	case 7:
		l.encodeEndSt1()
	}
}

func (l *Lha) encodeOuput(method int, code, pos uint16) {
	switch method {
	case 0:
		l.outputDyn(int(code), int(pos))
	case 1:
		l.outputSt1(code, pos)
	case 2:
		l.outputSt1(code, pos)
	case 3:
		l.outputSt1(code, pos)
	case 4:
		l.outputSt1(code, pos)
	case 5:
		l.outputSt1(code, pos)
	case 6:
		l.outputSt1(code, pos)
	case 7:
		l.outputSt1(code, pos)
	}
}

func insertHash(token, pos uint) { /* associate position with token */

	prevs[pos&uint(dicsiz-1)] = hash[token].pos /* chain the previous pos. */
	hash[token].pos = pos
}

func initHash(pos uint) uint {
	return uint((((uint16(text[pos]) << 5) ^ uint16(text[pos+1])) << 5) ^ uint16(text[pos+2])&uint16(hshsiz-1))
}

func nextHash(token, pos uint) uint {
	return uint((uint16(token)<<5 ^ uint16(text[pos+2])) & uint16(hshsiz-1))
}

func (l *Lha) updateDict(pos *uint, crc *uint) { /* update dictionary */
	var i, j uint

	copy(text[0:], text[dicsiz:txtsiz])

	n, err := l.freadCrc(crc, &text, uint(txtsiz-dicsiz), dicsiz, l.infile)
	if err != nil && io.EOF != err {
		fmt.Fprintf(os.Stderr, "error while freadCrc : %v\n", err)
	}

	l.remainder += uint(n)

	*pos -= uint(dicsiz)
	for i = 0; i < hshsiz; i++ {
		j = hash[i].pos
		hash[i].pos = 0
		if j > uint(dicsiz) {
			hash[i].pos = j - uint(dicsiz)
		}
		hash[i].tooFlag = 0
	}
	for i = 0; i < uint(dicsiz); i++ {
		j = prevs[i]
		prevs[i] = 0
		if j > uint(dicsiz) {
			prevs[i] = j - uint(dicsiz)
		}
	}
}

func (l *Lha) nextToken(token *uint, pos *uint, crc *uint) {
	l.remainder--
	*pos++
	if *pos >= uint(txtsiz)-uint(maxmatch) {
		l.updateDict(pos, crc)
	}
	*token = nextHash(*token, *pos)
}

func (l *Lha) searchDict(token, pos uint, min int, m *matchdata) {
	/* search token */
	/* position of token */
	/* min. length of matching string */

	var off, tok, max uint

	if min < threshold-1 {
		min = threshold - 1
	}
	max = uint(maxmatch)
	m.off = 0
	m.len = min

	off = 0
	for tok = token; hash[tok].tooFlag != 0 && off < uint(maxmatch-uint16(threshold)); {
		/* If matching position is too many, The search key is
		   changed into following token from `off' (for speed). */
		off++
		tok = nextHash(tok, pos+off)
	}
	if off == uint(maxmatch-uint16(threshold)) {
		off = 0
		tok = token
	}

	l.searchDict1(tok, pos, off, max, m)

	if off > 0 && m.len < int(off)+3 {
		/* re-search */
		l.searchDict1(token, pos, 0, off+2, m)
	}

	if m.len > int(l.remainder) {
		m.len = int(l.remainder)
	}
}

func (l *Lha) searchDict1(token, pos, off, max uint, m *matchdata) {
	/* max. length of matching string */

	var chain uint
	var scanPos int = int(hash[token].pos)
	var scanBeg int = scanPos - int(off)
	var scanEnd = int(pos) - dicsiz
	var le uint

	for scanBeg > 0 && scanBeg > scanEnd {
		chain++
		if text[int(scanBeg)+m.len] == text[int(pos)+m.len] {
			{
				var a, b int

				/* collate token */
				a = scanBeg
				b = int(pos)

				for le = 0; le < max && text[a] == text[b]; le++ {
					a++
					b++
				}
			}

			if int(le) > m.len {
				m.off = pos - uint(scanBeg)
				m.len = int(le)
				if m.len == int(max) {
					break
				}
			}
		}
		scanPos = int(prevs[scanPos&(dicsiz-1)])
		scanBeg = scanPos - int(off)
	}

	if chain >= limit {
		hash[token].tooFlag = 1
	}
}

func (l *Lha) encode(inter *interfacing) (uint, error) {
	var token, pos, crc uint
	var count int
	var match, last matchdata

	l.infile = inter.infile
	l.outfile = inter.outfile
	l.origsize = inter.original
	l.compsize = 0
	count = 0
	l.unpackable = false

	initializeCrc(&crc)

	initSlide()

	l.encodeStart(inter.method)

	for i := 0; i < (txtsiz + 1); i++ {
		text[i] = ' '
	}
	var err error
	var v int

	v, err = l.freadCrc(&crc, &text, uint(dicsiz), txtsiz-dicsiz, l.infile)
	if err != nil {
		return 0, err
	}
	l.remainder = uint(v)

	match.len = threshold - 1
	match.off = 0
	if match.len > int(l.remainder) {
		match.len = int(l.remainder)
	}

	pos = uint(dicsiz)
	token = initHash(pos)
	insertHash(token, pos) /* associate token and pos */

	for l.remainder > 0 && !l.unpackable {
		last = match

		l.nextToken(&token, &pos, &crc)
		l.searchDict(token, pos, last.len-1, &match)
		insertHash(token, pos)

		if match.len > last.len || last.len < threshold {
			/* output a letter */
			l.encodeOuput(inter.method, uint16(text[pos-1]), 0)
			count++
		} else {
			/* output length and offset */
			l.encodeOuput(inter.method,
				uint16(last.len+(256-threshold)),
				uint16((last.off-1)&uint(dicsiz-1)))

			count += last.len

			last.len -= 2
			for last.len > 0 {
				l.nextToken(&token, &pos, &crc)
				insertHash(token, pos)
				last.len--
			}
			l.nextToken(&token, &pos, &crc)
			l.searchDict(token, pos, threshold-1, &match)
			insertHash(token, pos)
		}
	}
	l.encodeEnd(inter.method)

	inter.packed = l.compsize
	inter.original = count

	return crc, nil
}

func initSlide() {
	for i := 0; i < int(hshsiz); i++ {
		hash[i].pos = 0
		hash[i].tooFlag = 0
	}
}
