package lha

import (
	"fmt"
	"io"
	"os"
)

/* ------------------------------------------------------------------------ */
/* LHa for UNIX                                                             */
/*              extract.c -- extrcat from archive                           */
/*                                                                          */
/*      Modified                Nobutaka Watazaki                           */
/*                                                                          */
/*  Ver. 1.14   Source All chagned              1995.01.14  N.Watazaki      */
/* ------------------------------------------------------------------------ */

func decodeLzhuf(infp io.Reader, outfp io.Writer, original_size int, packed_size int, name string, method int, read_sizep *int) int {
	var (
		crc   uint
		inter interfacing
	)
	inter.method = method
	inter.infile = infp
	inter.outfile = outfp
	inter.original = original_size
	inter.packed = packed_size
	inter.readSize = 0

	switch method {
	case lzhuff0MethodNum: /* -lh0- */
		inter.dicbit = lzhuff0Dicbit

	case lzhuff1MethodNum: /* -lh1- */
		inter.dicbit = lzhuff1Dicbit

	case lzhuff2MethodNum: /* -lh2- */
		inter.dicbit = lzhuff2Dicbit

	case lzhuff3MethodNum: /* -lh2- */
		inter.dicbit = lzhuff3Dicbit

	case lzhuff4MethodNum: /* -lh4- */
		inter.dicbit = lzhuff4Dicbit

	case lzhuff5MethodNum: /* -lh5- */
		inter.dicbit = lzhuff5Dicbit

	case lzhuff6MethodNum: /* -lh6- */
		inter.dicbit = lzhuff6Dicbit

	case lzhuff7MethodNum: /* -lh7- */
		inter.dicbit = lzhuff7Dicbit

	case larcMethodNum: /* -lzs- */
		inter.dicbit = larcDicbit

	case larc5MethodNum: /* -lz5- */
		inter.dicbit = larc5Dicbit

	case larc4MethodNum: /* -lz4- */
		inter.dicbit = larc4Dicbit

	case pmarc0MethodNum: /* -pm0- */
		inter.dicbit = pmarc0Dicbit

	case pmarc2MethodNum: /* -pm2- */
		inter.dicbit = pmarc2Dicbit

	default:
		fmt.Fprintf(os.Stdout, "unknown method %d", method)
		inter.dicbit = lzhuff5Dicbit /* for backward compatibility */
	}

	if inter.dicbit == 0 { /* LZHUFF0_DICBIT or LARC4_DICBIT or PMARC0_DICBIT*/
		mode := "Melting "
		if verifyMode {
			mode = "Testing "
		}
		startIndicator(name, original_size, []byte(mode), 2048)

		if dumpLzss {
			fmt.Printf("no use slide\n")
		}

		*read_sizep, _ = copyfile(&infp, &outfp, original_size, 2, &crc)
	} else {
		mode := "Melting "
		if verifyMode {
			mode = "Testing "
		}
		startIndicator(name, original_size, []byte(mode), 1<<inter.dicbit)
		if dumpLzss {
			fmt.Printf("\n")
		}

		crc = decode(&inter)
		*read_sizep = inter.readSize
	}
	mode := "Melted  "
	if verifyMode {
		mode = "Tested  "
	}
	finishIndicator(name, mode)

	return int(crc)
}
