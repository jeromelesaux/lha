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

func DecodeLzhuf(infp io.Reader, outfp io.Writer, original_size int, packed_size int, name string, method int, read_sizep *int) int {
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
	case Lzhuff0MethodNum: /* -lh0- */
		inter.dicbit = lzhuff0Dicbit

	case Lzhuff1MethodNum: /* -lh1- */
		inter.dicbit = lzhuff1Dicbit

	case Lzhuff2MethodNum: /* -lh2- */
		inter.dicbit = lzhuff2Dicbit

	case Lzhuff3MethodNum: /* -lh2- */
		inter.dicbit = lzhuff3Dicbit

	case Lzhuff4MethodNum: /* -lh4- */
		inter.dicbit = lzhuff4Dicbit

	case Lzhuff5MethodNum: /* -lh5- */
		inter.dicbit = lzhuff5Dicbit

	case Lzhuff6MethodNum: /* -lh6- */
		inter.dicbit = lzhuff6Dicbit

	case Lzhuff7MethodNum: /* -lh7- */
		inter.dicbit = lzhuff7Dicbit

	case LarcMethodNum: /* -lzs- */
		inter.dicbit = larcDicbit

	case Larc5MethodNum: /* -lz5- */
		inter.dicbit = larc5Dicbit

	case Larc4MethodNum: /* -lz4- */
		inter.dicbit = larc4Dicbit

	case Pmarc0MethodNum: /* -pm0- */
		inter.dicbit = pmarc0Dicbit

	case Pmarc2MethodNum: /* -pm2- */
		inter.dicbit = pmarc2Dicbit

	default:
		fmt.Fprintf(os.Stdout, "unknown method %d", method)
		inter.dicbit = lzhuff5Dicbit /* for backward compatibility */
	}

	if inter.dicbit == 0 { /* LZHUFF0_DICBIT or LARC4_DICBIT or PMARC0_DICBIT*/
		mode := "Melting "
		if VerifyMode {
			mode = "Testing "
		}
		startIndicator(name, original_size, []byte(mode), 2048)

		if DumpLzss {
			fmt.Printf("no use slide\n")
		}

		*read_sizep, _ = copyfile(&infp, &outfp, original_size, 2, &crc)
	} else {
		mode := "Melting "
		if VerifyMode {
			mode = "Testing "
		}
		startIndicator(name, original_size, []byte(mode), 1<<inter.dicbit)
		if DumpLzss {
			fmt.Printf("\n")
		}

		crc = decode(&inter)
		*read_sizep = inter.readSize
	}
	mode := "Melted  "
	if VerifyMode {
		mode = "Tested  "
	}
	finishIndicator(name, mode)

	return int(crc)
}
