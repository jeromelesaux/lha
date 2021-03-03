package lha

import (
	"fmt"
	"io"
	"strings"
)

/* ------------------------------------------------------------------------ */
/* LHa for UNIX                                                             */
/*              util.c -- LHarc Util                                        */
/*                                                                          */
/*      Modified                Nobutaka Watazaki                           */
/*                                                                          */
/*  Ver. 1.14   Source All chagned              1995.01.14  N.Watazaki      */
/*  Ver. 1.14e  Support for sfx archives        1999.05.28  T.Okamoto       */
/* ------------------------------------------------------------------------ */
/*
 * util.c - part of LHa for UNIX Feb 26 1992 modified by Masaru Oki Mar  4
 * 1992 modified by Masaru Oki #ifndef USESTRCASECMP added. Mar 31 1992
 * modified by Masaru Oki #ifdef NOMEMSET added.
 */

func copyfile(f1 *io.Reader, f2 *io.Writer, size, text_flg int, crcp *uint) (int, error) { /* return: size of source file */
	/* 0: binary, 1: read text, 2: write text */
	var xsize int
	var buf []byte = make([]byte, buffersize)
	var rsize int = 0

	if !TextMode {
		text_flg = 0
	}
	if *crcp != 0 {
		initializeCrc(crcp)
	}
	if text_flg != 0 {
		initCodeCache()
	}
	for size > 0 {
		/* read */
		var err error
		xsize, err = (*f1).Read(buf)

		if xsize == 0 {
			break
		}
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("file read error :%v", err.Error())
		}

		/* write */
		_, err = (*f2).Write(buf)
		if err != nil {
			return 0, fmt.Errorf("file write error :%v", err.Error())
		}

		/* calculate crc */
		if *crcp != 0 {
			*crcp = calcCrc(*crcp, &buf, 0, uint(xsize))
		}
		rsize += xsize
	}
	return rsize, nil
}

func encodeStoredCrc(ifp *io.Reader, ofp *io.Writer, size int, original_size_var *int, write_size_var *int) (uint, error) {
	var save_quiet bool
	var crc uint

	save_quiet = Quiet
	Quiet = true
	size, err := copyfile(ifp, ofp, size, 1, &crc)
	if err != nil {
		return 0, err
	}
	*original_size_var = size
	*write_size_var = size
	Quiet = save_quiet
	return crc, nil
}

/* If TRUE, archive file name is msdos SFX file name. */
func archiveIsMsdosSfx1(name []byte) bool {
	length := len(name)

	if length >= 4 {
		extension := strings.ToUpper(string(name[length : len(name)-4]))
		if extension == ".COM" || extension == ".EXE" {
			return true
		}
	}

	if length >= 2 {
		extension := strings.ToUpper(string(name[length : len(name)-2]))
		if extension == ".x" {
			return true
		}
	}
	return false
}
