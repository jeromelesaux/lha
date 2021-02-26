package lha

import (
	"fmt"
	"strconv"
)

var (
	extractBrokenArchive, dumpLzss bool
	dicsiz                         int
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
	case 4: /* lh5 */
	case 5: /* lh6 */
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
	case 4: /* lh5 */
	case 5: /* lh6 */
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
	case 1: /* lh2 */
		return decodeCDyn()
	case 2: /* lh3 */
		return decodeCSt0()
	case 3: /* lh4 */
	case 4: /* lh5 */
	case 5: /* lh6 */
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

	if !extractBrokenArchive {
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
	if (inter.method == larcMethodNum) || (inter.method == pmarc2MethodNum) {
		adjust = 256 - 2
	}
	decodeCount = 0
	loc = 0
	for decodeCount < origsize {
		c = uint(decodeMethodC(inter.method))
		if c < 256 {
			if dumpLzss {
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
			if dumpLzss {
				fmt.Printf("%d <%u %d>\n",
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
