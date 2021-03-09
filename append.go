package lha

import (
	"fmt"
	"io"
)

type Hash struct {
	pos     uint
	tooFlag int
}

var (
	hash   []Hash
	txtsiz = (maxDicsiz*2 + int(Maxmatch))
	text   []byte
	prevs  []uint
)

const (
	hshsiz = uint(1) << 15
)

func EncodeLzhuf(infp *io.Reader, outfp *io.Writer, size int, original_size_var *int, packed_size_var *int, name []byte, hdr_method []byte) (uint, error) {
	var method = -1
	var crc uint
	inter := &interfacing{}

	if method < 0 {
		method = CompressMethod
		if method > 0 {
			method, _ = encodeAlloc(method)
		}
	}

	inter.method = method

	if inter.method > 0 {
		inter.infile = *infp
		inter.outfile = *outfp
		inter.original = size
		startIndicator(string(name), size, []byte("Freezing"), 1<<dicbit)
		crc, _ = encode(inter)
		*packed_size_var = inter.packed
		*original_size_var = inter.original
	} else {
		*original_size_var, _ = copyfile(infp, outfp, size, 0, &crc)
		*packed_size_var = *original_size_var
	}
	copy(hdr_method, "-lh -")
	hdr_method[3] = byte(inter.method) + '0'

	finishIndicator2(string(name), "Frozen", (int)((*packed_size_var*100) / *original_size_var))
	return crc, nil
}

func encodeAlloc(method int) (int, error) {

	switch method {
	case Lzhuff1MethodNum:
		maxmatch = 60
		dicbit = uint16(lzhuff1Dicbit) /* 12 bits  Changed N.Watazaki */

	case Lzhuff5MethodNum:
		maxmatch = Maxmatch
		dicbit = uint16(lzhuff5Dicbit) /* 13 bits */

	case Lzhuff6MethodNum:
		maxmatch = Maxmatch
		dicbit = uint16(lzhuff6Dicbit) /* 15 bits */

	case Lzhuff7MethodNum:
		maxmatch = Maxmatch
		dicbit = uint16(lzhuff7Dicbit) /* 16 bits */

	default:
		return 0, fmt.Errorf("unknown method %d", method)

	}

	dicsiz = int(uint(1) << uint(dicbit))
	txtsiz = dicsiz*2 + int(maxmatch)

	if len(hash) != 0 {
		return method, nil
	}

	allocBuf()

	hash = make([]Hash, hshsiz)
	prevs = make([]uint, maxDicsiz)
	text = make([]byte, txtsiz)

	return method, nil
}
