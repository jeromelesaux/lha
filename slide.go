package lha

import (
	"fmt"
	"strconv"
)

var (
	ExtractBrokenArchive, DumpLzss bool
	dicsiz                         int
	remainder                      uint
)

const (
	limit = 0x100
)

type matchdata struct {
	len int
	off uint
}

func decodeMethodStart(method int) {
	switch method - 1 {
	case 0: /* lh1 */
		decodeStartFix()
	case 1: /* lh2 */
		decodeStartDyn()
	case 2: /* lh3 */
		decodeStartSt0()
	case 3: /* lh4 */
		decodeStartSt1()
	case 4: /* lh5 */
		decodeStartSt1()
	case 5: /* lh6 */
		decodeStartSt1()
	case 6: /* lh7 */
		decodeStartSt1()
	case 7: /* lzs */
		decodeStartLzs()
	case 8: /* lz5 */
		decodeStartLz5()
	case 12: /* pm2 */
		decodeStartPm2()
	default:
		return
	}
}

func decodeMethodP(method int) uint16 {
	switch method - 1 {
	case 0: /* lh1 */
		return decodePSt0()
	case 1: /* lh2 */
		return decodePDyn()
	case 2: /* lh3 */
		return decodePSt0()
	case 3: /* lh4 */
		return decodePSt1()
	case 4: /* lh5 */
		return decodePSt1()
	case 5: /* lh6 */
		return decodePSt1()
	case 6: /* lh7 */
		return decodePSt1()
	case 7: /* lzs */
		return decodePLzs()
	case 8: /* lz5 */
		return decodePLz5()
	case 12: /* pm2 */
		return decodePPm2()
	default:
		return 0
	}
	return 0
}

func decodeMethodC(method int) uint16 {
	switch method - 1 {
	case 0: /* lh1 */
		return decodeCDyn()
	case 1: /* lh2 */
		return decodeCDyn()
	case 2: /* lh3 */
		return decodeCSt0()
	case 3: /* lh4 */
		return decodeCSt1()
	case 4: /* lh5 */
		return decodeCSt1()
	case 5: /* lh6 */
		return decodeCSt1()
	case 6: /* lh7 */
		return decodeCSt1()
	case 7: /* lzs */
		return decodeCLzs()
	case 8: /* lz5 */
		return decodeCLz5()
	case 12: /* pm2 */
		return decodeCPm2()
	default:
		return 0
	}
	return 0
}

func decode(inter *interfacing) uint {
	var (
		i, c            uint
		dicsiz1, adjust uint
		crc             uint
	)

	infile = inter.infile
	outfile = inter.outfile
	dicbit = uint16(inter.dicbit)
	origsize = inter.original
	compsize = inter.packed

	initializeCrc(&crc)
	dicsiz = int(1) << dicbit
	dtext = make([]byte, dicsiz)

	if !ExtractBrokenArchive {
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

	decodeMethodStart(inter.method)
	dicsiz1 = uint(dicsiz) - 1
	adjust = 256 - uint(threshold)
	if (inter.method == LarcMethodNum) || (inter.method == Pmarc2MethodNum) {
		adjust = 256 - 2
	}
	decodeCount = 0
	loc = 0
	for decodeCount < origsize {
		c = uint(decodeMethodC(inter.method))
		if c < 256 {
			if DumpLzss {
				b := c
				if !strconv.IsPrint(rune(c)) {
					b = '?'
				}
				fmt.Printf("%d %02x(%c)\n", decodeCount, c, b)

			}
			dtext[loc] = byte(c)
			loc++
			if int(loc) == dicsiz {
				fwriteCrc(&crc, dtext, dicsiz, &outfile)
				loc = 0
			}
			decodeCount++
		} else {
			var match matchdata
			var matchpos uint

			match.len = int(c - adjust)
			match.off = uint(decodeMethodP(inter.method)) + 1
			matchpos = (uint(loc) - match.off) & dicsiz1
			if DumpLzss {
				fmt.Printf("%d <%d %d>\n",
					decodeCount, match.len, decodeCount-int(match.off))
			}

			decodeCount += match.len
			for i = 0; i < uint(match.len); i++ {
				c = uint(dtext[(matchpos+i)&dicsiz1])
				dtext[loc] = byte(c)
				loc++
				if int(loc) == dicsiz {
					fwriteCrc(&crc, dtext, dicsiz, &outfile)
					loc = 0
				}
			}
		}
	}
	if loc != 0 {
		fwriteCrc(&crc, dtext, int(loc), &outfile)
	}
	/* usually read size is interface->packed */
	inter.readSize = inter.packed - compsize

	return crc
}

func encodeStart(method int) {
	switch method {
	case 0:
		encodeStartFix()
	case 1:
		encodeStartSt1()
	case 2:
		encodeStartSt1()
	case 3:
		encodeStartSt1()
	}
}

func encodeEnd(method int) {
	switch method {
	case 0:
		encodeEndDyn()
	case 1:
		encodeEndSt1()
	case 2:
		encodeEndSt1()
	case 3:
		encodeEndSt1()
	}
}

func encodeOuput(method int, code, pos uint16) {
	switch method {
	case 0:
		outputDyn(int(code), int(pos))
	case 1:
		outputSt1(code, pos)
	case 2:
		outputSt1(code, pos)
	case 3:
		outputSt1(code, pos)
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

func updateDict(pos *uint, crc *uint) { /* update dictionary */
	var i, j uint

	copy(text[0:], text[dicsiz:txtsiz-dicsiz])

	n, _ := freadCrc(crc, &text, uint(txtsiz-dicsiz), dicsiz, infile)

	remainder += uint(n)

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

func nextToken(token *uint, pos *uint, crc *uint) {
	remainder--
	*pos++
	if *pos >= uint(txtsiz)-uint(maxmatch) {
		updateDict(pos, crc)
		*pos++
	}
	*token = nextHash(*token, *pos)
}

func searchDict(token, pos uint, min int, m *matchdata) {
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

	searchDict1(tok, pos, off, max, m)

	if off > 0 && m.len < int(off)+3 {
		/* re-search */
		searchDict1(token, pos, 0, off+2, m)
	}

	if m.len > int(remainder) {
		m.len = int(remainder)
	}
}

func searchDict1(token, pos, off, max uint, m *matchdata) {
	/* max. length of matching string */

	var chain uint
	var scanPos = hash[token].pos
	var scanBeg = scanPos - off
	var scanEnd = pos - uint(dicsiz)
	var length uint

	for scanBeg > scanEnd {
		chain++

		if text[int(scanBeg)+m.len] == text[int(pos)+m.len] {
			{
				var a, b byte
				var aindex, bindex int
				/* collate token */
				a = text[scanBeg]
				b = text[pos]

				for length = 0; length < max && a == b; length++ {
					aindex++
					bindex++

					a = text[scanBeg+uint(aindex)]
					b = text[pos+uint(bindex)]
				}
			}

			if int(length) > m.len {
				m.off = pos - scanBeg
				m.len = int(length)
				if m.len == int(max) {

					break
				}
			}
		}
		scanPos = prevs[scanPos&uint(dicsiz-1)]
		scanBeg = scanPos - off
	}

	if chain >= limit {
		hash[token].tooFlag = 1
	}
}

func encode(inter *interfacing) (uint, error) {
	var token, pos, crc uint
	var count int
	var match, last matchdata

	infile = inter.infile
	outfile = inter.outfile
	origsize = inter.original
	compsize = 0
	count = 0
	unpackable = false

	initializeCrc(&crc)

	initSlide()

	encodeStart(inter.method)

	for i := 0; i < txtsiz; i++ {
		text[i] = ' '
	}
	var err error
	var v int

	v, err = freadCrc(&crc, &text, uint(dicsiz), txtsiz-dicsiz, infile)
	if err != nil {
		return 0, err
	}
	remainder = uint(v)

	match.len = threshold - 1
	match.off = 0
	if match.len > int(remainder) {
		match.len = int(remainder)
	}

	pos = uint(dicsiz)
	token = initHash(pos)
	insertHash(token, pos) /* associate token and pos */

	for remainder > 0 && !unpackable {
		last = match

		nextToken(&token, &pos, &crc)
		searchDict(token, pos, last.len-1, &match)
		insertHash(token, pos)

		if match.len > last.len || last.len < threshold {
			/* output a letter */
			encodeOuput(inter.method, uint16(text[pos-1]), 0)
			count++
		} else {
			/* output length and offset */
			encodeOuput(inter.method, uint16(last.len+(256-threshold)), uint16((last.off-1)&uint(dicsiz-1)))

			count += last.len

			last.len--
			last.len--
			for last.len > 0 {
				nextToken(&token, &pos, &crc)
				insertHash(token, pos)
				last.len--
			}
			nextToken(&token, &pos, &crc)
			searchDict(token, pos, threshold-1, &match)
			insertHash(token, pos)
		}
	}
	encodeEnd(inter.method)

	inter.packed = compsize
	inter.original = count

	return crc, nil
}

func initSlide() {
	for i := 0; i < int(hshsiz); i++ {
		hash[i].pos = 0
		hash[i].tooFlag = 0
	}
}
