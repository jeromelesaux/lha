package main

import (
	"fmt"
	"os"
	"time"
)

var (
	getPtr int
	putPtr int
)

type LzHeader struct {
	headerSize      int
	sizeFieldLength int
	method          [methodTypeStorage]byte
	packedSize      int
	originalSize    int
	attribute       byte
	headerLevel     byte
	name            [filenameLength]byte
	realname        [filenameLength]byte /* real name for symbolic link */
	crc             uint                 /* file CRC */
	hasCrc          bool                 /* file CRC */
	headerCrc       uint                 /* header CRC */
	extendType      byte
	minorVersion    byte

	/* extend_type == EXTEND_UNIX  and convert from other type. */
	unixLastModifiedStamp time.Time
	unixMode              uint16
	unixUID               uint16
	unixGid               uint16
	user                  [256]byte
	group                 [256]byte
}

func NewLzHeader() *LzHeader {
	return &LzHeader{}
}

func (l *LzHeader) InitHeader(fileSize int, headerLevel byte) {
	//var len int
	n := copy(l.method[:], []byte(lzhuff0Method))
	if n != methodTypeStorage {
		fmt.Fprintf(os.Stderr, "copy lzh method differs expected [%d] and [%d] bytes copied\n", methodTypeStorage, n)
	}
	l.packedSize = 0
	l.originalSize = fileSize
	l.attribute = genericAttribute
	l.headerLevel = headerLevel
}

func calcSum(p *[]byte, start, len int) int {
	sum := 0
	for len != 0 {
		sum += int((*p)[start])
		len--
		start++
	}

	return sum & 0xff
}

func getByte(ptr *[]byte) int {
	v := (*ptr)[getPtr] & 0xff
	getPtr++
	return int(v)
}

func skipBytes(len int) {
	getPtr += len
}

func dumpGetByte(ptr *[]byte) int {
	return getByte(ptr)
}

func dumpSkipBytes(ptr *[]byte, len int) {
	if len == 0 {
		return
	} else {
		skipBytes(len)
	}
	return
}

func getWord(ptr *[]byte) int {
	var b0, b1, w int
	b0 = getByte(ptr)
	b1 = getByte(ptr)
	w = (b1 << 8) + b0
	return w
}

func putByte(ptr *[]byte, c byte) {
	(*ptr)[putPtr] = c
	putPtr++
}

func putWord(ptr *[]byte, v int) {
	putByte(ptr, byte(v))
	putByte(ptr, byte(v>>8))
}

func getLongword(ptr *[]byte) int {
	b0 := getByte(ptr)
	b1 := getByte(ptr)
	b2 := getByte(ptr)
	b3 := getByte(ptr)
	l := (b3 << 24) + (b2 << 16) + (b1 << 8) + b0
	return l
}

func putLongWord(ptr *[]byte, v int) {
	putByte(ptr, byte(v))
	putByte(ptr, byte(v>>8))
	putByte(ptr, byte(v>>16))
	putByte(ptr, byte(v>>24))
}

func getBytes(buf *[]byte, ptr *[]byte, len, size int) int {
	var i int
	for i = 0; i < len && i < size; i++ {
		(*buf)[i] = (*ptr)[i]
	}
	getPtr += len
	return i
}

func putBytes(buf, ptr *[]byte, len int) {
	for i := 0; i < len; i++ {
		putByte(ptr, (*buf)[i])
	}
}

func subbits(n, off, len int) int {
	return (((n) >> (off)) & ((1 << (len)) - 1))
}

func genericToUnixStamp(t int64) time.Time {
	return time.Unix(t, 0)
}
