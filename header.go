package lha

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"syscall"

	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

var (
	getPtr        int
	putPtr        int
	storageSize   int
	startPtr      int
	ptr           *[]byte
	convertCase   bool
	genericFormat bool
)

type LzHeaderList struct {
	next *LzHeaderList
	hdr  *LzHeader
}

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
	unixLastModifiedStamp int64
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

func getByte() byte {
	v := (*ptr)[getPtr] & 0xff
	getPtr++
	return v
}

func skipBytes(len int) {
	getPtr += len
}

func dumpGetByte() byte {
	return getByte()
}

func dumpSkipBytes(len int) {
	if len == 0 {
		return
	} else {
		skipBytes(len)
	}
	return
}

func getWord() int {
	var b0, b1, w int
	b0 = int(getByte())
	b1 = int(getByte())
	w = (b1 << 8) + b0
	return w
}

func putByte(c byte) {
	(*ptr)[putPtr] = c
	putPtr++
}

func putWord(v int) {
	putByte(byte(v))
	putByte(byte(v >> 8))
}

func getLongword() int {
	var l int
	b0 := getByte()
	b1 := getByte()
	b2 := getByte()
	b3 := getByte()

	l = int((b3 << 24) + (b2 << 16) + (b1 << 8) + b0)
	return l
}

func putLongWord(v int) {
	putByte(byte(v))
	putByte(byte(v >> 8))
	putByte(byte(v >> 16))
	putByte(byte(v >> 24))
}

func getBytes(buf *[]byte, len, size int) int {
	var i int
	for i = 0; i < len && i < size; i++ {
		(*buf)[i] = (*ptr)[i]
	}
	getPtr += len
	return i
}

func putBytes(buf []byte, len int) {
	for i := 0; i < len; i++ {
		putByte(buf[i])
	}
}

func subbits(n, off, len int) int {
	return (((n) >> (off)) & ((1 << (len)) - 1))
}

func genericToUnixStamp(t int64) int64 {
	return time.Unix(t, 0).Unix()
}

func setupGet(p *[]byte, index, size int) {
	ptr = p
	getPtr = index
	startPtr = index
	storageSize = size
}

func setupPut(p *[]byte, index int) {
	ptr = p
	putPtr = index
}

func unixToGenericStamp(t int64) int {
	tm := time.Unix(t, 0)
	var us int
	us = ((tm.Year() - 80) << 25) +
		(int(tm.Month()) << 21) +
		(tm.Day() << 16) +
		(tm.Hour() << 11) +
		(tm.Minute() << 5) +
		(tm.Second() / 2)
	return us
}

func wintimeToUnixStamp() uint64 {
	var wintime [8]uint64
	epoch := [8]uint64{0x01, 0x9d, 0xb1, 0xde, 0xd5, 0x3e, 0x80, 0x00}
	/* 1970-01-01 00:00:00 (UTC) */
	/* wintime -= epoch */
	var borrow uint64 = 0
	for i := 7; i >= 0; i-- {
		wintime[i] = uint64(getByte()) - epoch[i] - borrow
		if wintime[i] > 0xff {
			borrow = 1
		} else {
			borrow = 0
		}
		wintime[i] &= 0xff
	}
	/* q = wintime / 10000000 */
	var x uint64 = 10000000 /* x: 24bit */
	var q, t uint64
	for i := 0; i < 8; i++ {
		t = (t << 8) + wintime[i] /* 24bit + 8bit. t must be 32bit variable */
		q <<= 8                   /* q must be 32bit (time_t) */
		q += t / x
		t %= x /* 24bit */

	}
	return q
}

/*
 * extended header
 *
 *             size  field name
 *  --------------------------------
 *  base header:         :
 *           2 or 4  next-header size  [*1]
 *  --------------------------------------
 *  ext header:   1  ext-type            ^
 *                ?  contents            | [*1] next-header size
 *           2 or 4  next-header size    v
 *  --------------------------------------
 *
 *  on level 1, 2 header:
 *    size field is 2 bytes
 *  on level 3 header:
 *    size field is 4 bytes
 */

func (l *LzHeader) getExtendedHeader(fp *io.Reader, headerSize int, hcrc *uint) (error, int) {
	var data [lzheaderStorage]byte
	var nameLength int
	var dirname [filenameLength]byte
	var dirLength, i int
	wholeSize := headerSize
	var extType int
	n := i + l.sizeFieldLength /* `ext-type' + `next-header size' */
	if l.headerLevel == 0 {
		return nil, 0
	}
	nameLength = len(l.name)
	for l.headerSize != 0 {
		setupGet((*[]byte)(unsafe.Pointer(&data)), 0, len(data))
		if len(data) < l.headerSize {
			return fmt.Errorf("header size (%d) too large.", l.headerSize), 0
		}
		nb, err := (*fp).Read(data[:l.headerSize])
		if err != nil || nb == 0 {
			return fmt.Errorf("Invalid header (LHA file ?)"), 0
		}
		extType = int(getByte())
		switch extType {
		case 0:
			l.headerCrc = uint(getWord())
			/* header crc (CRC-16) */
			/* clear buffer for CRC calculation. */
			data[1] = 0
			data[2] = 0
			skipBytes(l.headerSize - n - 2)
		case 1:
			/* filename */
			nameLength = getBytes((*[]byte)(unsafe.Pointer(&l.name)), l.headerSize-n, len(l.name)-1)
			l.name[nameLength] = 0
		case 2:
			dirLength = getBytes((*[]byte)(unsafe.Pointer(&dirname)), headerSize-n, len(dirname)-1)
			dirname[dirLength] = 0
		case 0x40:
			/* MS-DOS attribute */
			l.attribute = byte(getWord())
		case 0x41:
			/* Windows time stamp (FILETIME structure) */
			/* it is time in 100 nano seconds since 1601-01-01 00:00:00 */

			skipBytes(8) /* create time is ignored */

			/* set last modified time */
			if l.headerLevel >= 2 {
				skipBytes(8) /* time_t has been already set */
			} else {
				l.unixLastModifiedStamp = int64(wintimeToUnixStamp())
			}
			skipBytes(8) /* last access time is ignored */
		case 0x42:
			skipBytes(8)
			skipBytes(8)
		case 0x50:
			/* UNIX permission */
			l.unixMode = uint16(getWord())
		case 0x51:
			/* UNIX group name */
			i = getBytes((*[]byte)(unsafe.Pointer(&l.group)), headerSize-n, len(l.group)-1)
			l.group[i] = 0
		case 0x53:
			/* UNIX user name */
			i = getBytes((*[]byte)(unsafe.Pointer(&l.user)), headerSize-n, len(l.user)-1)
			l.user[i] = 0
		case 0x54:
			/* UNIX last modified time */
			l.unixLastModifiedStamp = int64(getLongword())
		default:
			/* other headers */
			/* 0x39: multi-disk header
			   0x3f: uncompressed comment
			   0x42: 64bit large file size
			   0x48-0x4f(?): reserved for authenticity verification
			   0x7d: encapsulation
			   0x7e: extended attribute - platform information
			   0x7f: extended attribute - permission, owner-id and timestamp
			         (level 3 on OS/2)
			   0xc4: compressed comment (dict size: 4096)
			   0xc5: compressed comment (dict size: 8192)
			   0xc6: compressed comment (dict size: 16384)
			   0xc7: compressed comment (dict size: 32768)
			   0xc8: compressed comment (dict size: 65536)
			   0xd0-0xdf(?): operating systemm specific information
			   0xfc: encapsulation (another opinion)
			   0xfe: extended attribute - platform information(another opinion)
			   0xff: extended attribute - permission, owner-id and timestamp
			         (level 3 on UNLHA32) */
			skipBytes(headerSize - n)
		}
		if *hcrc != 0 {
			*hcrc = calcCrc(*hcrc, (*[]byte)(unsafe.Pointer(&data)), uint(getPtr), uint(headerSize))
		}

		if l.sizeFieldLength == 2 {
			headerSize = getWord()
			wholeSize += headerSize
		} else {
			headerSize = getLongword()
			wholeSize += headerSize
		}
	}

	return nil, wholeSize
}

const (
	iHeaderSize     = 0  /* level 0,1,2   */
	iHeaderChecksum = 1  /* level 0,1     */
	iMethod         = 2  /* level 0,1,2,3 */
	iPackedSize     = 7  /* level 0,1,2,3 */
	iAttribute      = 19 /* level 0,1,2,3 */
	iHeaderLevel    = 20 /* level 0,1,2,3 */

	commonHeaderSize = 21 /* size of common part */

	iGenericHeaderSize = 24 /* + nameLength */
	iLevel0HeaderSize  = 36 /* + nameLength (unix extended) */
	iLevel1HeaderSize  = 27 /* + nameLength */
	iLevel2HeaderSize  = 26 /* + padding */
	iLevel3HeaderSize  = 32
)

var (
	defaultSystemKanjiCode   = none
	optionalArchiveKanjiCode = none
	optionalSystemKanjiCode  = none
	optionalArchiveDelim     = ""
	optionalSystemDelim      = ""
	optionalFilenameCase     = none
)

/*
 * level 0 header
 *
 *
 * offset  size  field name
 * ----------------------------------
 *     0      1  header size    [*1]
 *     1      1  header sum
 *            ---------------------------------------
 *     2      5  method ID                         ^
 *     7      4  packed size    [*2]               |
 *    11      4  original size                     |
 *    15      2  time                              |
 *    17      2  date                              |
 *    19      1  attribute                         | [*1] header size (X+Y+22)
 *    20      1  level (0x00 fixed)                |
 *    21      1  name length                       |
 *    22      X  pathname                          |
 * X +22      2  file crc (CRC-16)                 |
 * X +24      Y  ext-header(old style)             v
 * -------------------------------------------------
 * X+Y+24        data                              ^
 *                 :                               | [*2] packed size
 *                 :                               v
 * -------------------------------------------------
 *
 * ext-header(old style)
 *     0      1  ext-type ('U')
 *     1      1  minor version
 *     2      4  UNIX time
 *     6      2  mode
 *     8      2  uid
 *    10      2  gid
 *
 * attribute (MS-DOS)
 *    bit1  read only
 *    bit2  hidden
 *    bit3  system
 *    bit4  volume label
 *    bit5  directory
 *    bit6  archive bit (need to backup)
 *
 */
func (l *LzHeader) getHeaderLevel0(fp *io.Reader, data []byte) (error, bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var checksum int
	var nameLength int
	var i int

	l.sizeFieldLength = 2 /* in bytes */
	l.headerSize = int(getByte())
	headerSize = l.headerSize
	checksum = int(getByte())

	/* The data variable has been already read as COMMON_HEADER_SIZE bytes.
	So we must read the remaining header size by the headerSize. */
	remainSize = headerSize + 2 - commonHeaderSize
	if remainSize <= 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false

	}
	nb, err := (*fp).Read(data[commonHeaderSize : commonHeaderSize+remainSize])

	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	if calcSum((*[]byte)(unsafe.Pointer(&data)), iMethod, headerSize) != checksum {
		return fmt.Errorf("Checksum error (LHarc file?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) // sizeof
	l.packedSize = getLongword()
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = genericToUnixStamp(int64(getLongword()))
	l.attribute = getByte() /* MS-DOS attribute */
	l.headerLevel = getByte()
	nameLength = int(getByte())
	i = getBytes((*[]byte)(unsafe.Pointer(&l.name)), nameLength, 2-1) // sizeof l.name
	l.name[i] = 0

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	extendSize = headerSize + 2 - nameLength - 24

	if extendSize < 0 {
		if extendSize == -2 {
			/* CRC field is not given */
			l.extendType = extendGeneric
			l.hasCrc = false

			return nil, true
		}

		return fmt.Errorf("Unkonwn header (lha file?)"), false
	}

	l.hasCrc = true
	l.crc = uint(getWord())

	if extendSize == 0 {
		return nil, true
	}

	l.extendType = getByte()
	extendSize--

	if l.extendType == extendUnix {
		if extendSize >= 11 {
			l.minorVersion = getByte()
			l.unixLastModifiedStamp = int64(getLongword())
			l.unixMode = uint16(getWord())
			l.unixUID = uint16(getWord())
			l.unixGid = uint16(getWord())
			extendSize -= 11
		} else {
			l.extendType = extendGeneric
		}
	}
	if extendSize > 0 {
		skipBytes(extendSize)
	}

	l.headerSize += 2
	return nil, true
}

/*
 * level 1 header
 *
 *
 * offset   size  field name
 * -----------------------------------
 *     0       1  header size   [*1]
 *     1       1  header sum
 *             -------------------------------------
 *     2       5  method ID                        ^
 *     7       4  skip size     [*2]               |
 *    11       4  original size                    |
 *    15       2  time                             |
 *    17       2  date                             |
 *    19       1  attribute (0x20 fixed)           | [*1] header size (X+Y+25)
 *    20       1  level (0x01 fixed)               |
 *    21       1  name length                      |
 *    22       X  filename                         |
 * X+ 22       2  file crc (CRC-16)                |
 * X+ 24       1  OS ID                            |
 * X +25       Y  ???                              |
 * X+Y+25      2  next-header size                 v
 * -------------------------------------------------
 * X+Y+27      Z  ext-header                       ^
 *                 :                               |
 * -----------------------------------             | [*2] skip size
 * X+Y+Z+27       data                             |
 *                 :                               v
 * -------------------------------------------------
 *
 */
func (l *LzHeader) getHeaderLevel1(fp *io.Reader, data []byte) (err error, ok bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var checksum int
	var nameLength int
	var i, dummy int

	l.sizeFieldLength = 2 /* in bytes */
	l.headerSize = int(getByte())
	headerSize = l.headerSize
	checksum = int(getByte())

	/* The data variable has been already read as COMMON_HEADER_SIZE bytes.
	So we must read the remaining header size by the headerSize. */
	remainSize = headerSize + 2 - commonHeaderSize
	if remainSize <= 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false
	}
	nb, err := (*fp).Read(data[commonHeaderSize : commonHeaderSize+remainSize])
	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	if calcSum((*[]byte)(unsafe.Pointer(&data)), iMethod, headerSize) != checksum {
		return fmt.Errorf("Checksum error (LHarc file?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) //sizeof(l.method)
	l.packedSize = getLongword()                         /* skip size */
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = genericToUnixStamp(int64(getLongword()))
	l.attribute = getByte() /* 0x20 fixed */
	l.headerLevel = getByte()

	nameLength = int(getByte())
	i = getBytes((*[]byte)(unsafe.Pointer(&l.name)), nameLength, 2-1) // sizeof(l.name)
	l.name[i] = 0

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	l.hasCrc = true
	l.crc = uint(getWord())
	l.extendType = getByte()

	dummy = headerSize + 2 - nameLength - iLevel1HeaderSize
	if dummy > 0 {
		skipBytes(dummy) /* skip old style extend header */
	}

	extendSize = getWord()
	var trash uint
	err, extendSize = l.getExtendedHeader(fp, extendSize, &trash)
	if err != nil || extendSize == -1 {
		return err, false
	}

	/* On level 1 header, size fields should be adjusted. */
	/* the `packed_size' field contains the extended header size. */
	/* the `headerSize' field does not. */
	l.packedSize -= extendSize
	l.headerSize += extendSize + 2

	return nil, true
}

/*
 * level 2 header
 *
 *
 * offset   size  field name
 * --------------------------------------------------
 *     0       2  total header size [*1]           ^
 *             -----------------------             |
 *     2       5  method ID                        |
 *     7       4  packed size       [*2]           |
 *    11       4  original size                    |
 *    15       4  time                             |
 *    19       1  RESERVED (0x20 fixed)            | [*1] total header size
 *    20       1  level (0x02 fixed)               |      (X+26+(1))
 *    21       2  file crc (CRC-16)                |
 *    23       1  OS ID                            |
 *    24       2  next-header size                 |
 * -----------------------------------             |
 *    26       X  ext-header                       |
 *                 :                               |
 * -----------------------------------             |
 * X +26      (1) padding                          v
 * -------------------------------------------------
 * X +26+(1)      data                             ^
 *                 :                               | [*2] packed size
 *                 :                               v
 * -------------------------------------------------
 *
 */
func (l *LzHeader) getHeaderLevel2(fp *io.Reader, data []byte) (error, bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var padding int
	var hcrc uint

	l.sizeFieldLength = 2 /* in bytes */
	l.headerSize = getWord()
	headerSize = l.headerSize

	/* The data variable has been already read as COMMON_HEADER_SIZE bytes.
	So we must read the remaining header size without ext-header. */
	remainSize = headerSize - iLevel2HeaderSize
	if remainSize < 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false
	}
	n, err := (*fp).Read(data[commonHeaderSize : commonHeaderSize+(iLevel2HeaderSize-commonHeaderSize)])
	if err != nil || n == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) // sizeof(l.method)
	l.packedSize = getLongword()
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = int64(getLongword())
	l.attribute = getByte() /* reserved */
	l.headerLevel = getByte()

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	l.hasCrc = true
	l.crc = uint(getWord())
	l.extendType = getByte()
	extendSize = getWord()

	initializeCrc(&hcrc)

	hcrc = calcCrc(hcrc, (*[]byte)(unsafe.Pointer(&data)), 0, uint(n-len(data)))
	err, extendSize = l.getExtendedHeader(fp, extendSize, &hcrc)
	if err != nil || extendSize == -1 {
		return err, false
	}

	padding = headerSize - iLevel2HeaderSize - extendSize
	/* padding should be 0 or 1 */
	if padding != 0 && padding != 1 {
		return fmt.Errorf("Invalid header size (padding: %d)", padding), false
	}
	padding--
	for padding != 0 {
		buf := make([]byte, 1)
		_, err := (*fp).Read(buf)
		if err != nil {
			return err, false
		}
		hcrc = updateCrc(&hcrc, uint(buf[0]))
	}

	if l.headerCrc != hcrc {
		return fmt.Errorf("header CRC error"), false
	}

	return nil, true
}

/*
 * level 3 header
 *
 *
 * offset   size  field name
 * --------------------------------------------------
 *     0       2  size field length (4 fixed)      ^
 *     2       5  method ID                        |
 *     7       4  packed size       [*2]           |
 *    11       4  original size                    |
 *    15       4  time                             |
 *    19       1  RESERVED (0x20 fixed)            | [*1] total header size
 *    20       1  level (0x03 fixed)               |      (X+32)
 *    21       2  file crc (CRC-16)                |
 *    23       1  OS ID                            |
 *    24       4  total header size [*1]           |
 *    28       4  next-header size                 |
 * -----------------------------------             |
 *    32       X  ext-header                       |
 *                 :                               v
 * -------------------------------------------------
 * X +32          data                             ^
 *                 :                               | [*2] packed size
 *                 :                               v
 * -------------------------------------------------
 *
 */
func (l *LzHeader) getHeaderLevel3(fp *io.Reader, data []byte) (error, bool) {
	var headerSize int
	var remainSize int
	var extendSize int
	var padding int
	var hcrc uint

	l.sizeFieldLength = getWord()
	nb, err := (*fp).Read(data[commonHeaderSize : commonHeaderSize+iLevel3HeaderSize-commonHeaderSize])
	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	getBytes((*[]byte)(unsafe.Pointer(&l.method)), 5, 2) //sizeof(l.method)
	l.packedSize = getLongword()
	l.originalSize = getLongword()
	l.unixLastModifiedStamp = int64(getLongword())
	l.attribute = getByte() /* reserved */
	l.headerLevel = getByte()

	/* defaults for other type */
	l.unixMode = uint16(unixFileRegular) | uint16(unixRwRwRw)
	l.unixGid = 0
	l.unixUID = 0

	l.hasCrc = true
	l.crc = uint(getWord())
	l.extendType = getByte()
	l.headerSize = getLongword()
	headerSize = l.headerSize
	remainSize = headerSize - iLevel3HeaderSize
	if remainSize < 0 {
		return fmt.Errorf("Invalid header size (LHarc file ?)"), false
	}
	extendSize = getLongword()

	initializeCrc(&hcrc)
	hcrc = calcCrc(hcrc, (*[]byte)(unsafe.Pointer(&data)), 0, uint(nb-len(data)))

	err, extendSize = l.getExtendedHeader(fp, extendSize, &hcrc)
	if err != nil || extendSize == -1 {
		return err, false
	}

	padding = remainSize - extendSize
	/* padding should be 0 */
	if padding != 0 {
		return fmt.Errorf("Invalid header size (padding: %d)", padding), false
	}

	if l.headerCrc != hcrc {
		return fmt.Errorf("header CRC error"), false
	}

	return nil, true
}

func (l *LzHeader) getHeader(fp *io.Reader) (error, bool) {
	var data [lzheaderStorage]byte

	archiveKanjiCode := codeSJIS
	systemKanjiCode := defaultSystemKanjiCode
	var archiveDelim string = "\377\\" /* `\' is for level 0 header and
	   broken archive. */
	var systemDelim string = "//"
	var filenameCase int = none
	var endMark byte

	setupGet((*[]byte)(unsafe.Pointer(&data)), 0, len(data))
	buf := make([]byte, 1)
	nb, err := (*fp).Read(buf)

	endMark = buf[0]
	if err != nil || endMark == 0 {
		return err, false /* finish */
	}
	data[0] = endMark

	nb, err = (*fp).Read(data[1:commonHeaderSize])
	if err != nil || nb == 0 {
		return fmt.Errorf("Invalid header (LHarc file ?)"), false
	}

	switch data[iHeaderLevel] {
	case 0:
		err, header := l.getHeaderLevel0(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	case 1:
		err, header := l.getHeaderLevel1(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	case 2:
		err, header := l.getHeaderLevel2(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	case 3:
		err, header := l.getHeaderLevel3(fp, data[:])
		if err != nil || header == false {
			return err, false
		}
	default:
		return fmt.Errorf("Unknown level header (level %d)", data[iHeaderLevel]), false
	}

	/* filename conversion */
	switch l.extendType {
	case extendMsdos:
		filenameCase = none
		if convertCase {
			filenameCase = toLower
		}
	case extendHuman:
	case extendOs68k:
	case extendXosk:
	case extendUnix:
	case extendJava:
		filenameCase = none

	case extendMacos:
		archiveDelim = "\377/:\\"
		/* `\' is for level 0 header and broken archive. */
		systemDelim = "/://"
		filenameCase = none

	case extendAmiga:
		{
			/* workaround */
			lenName := len(l.name)
			if lenName > 0 && l.name[lenName-1] == lhaPathsep && string(l.method[:]) == lzhuff0Method {
				/* replace with "-lhd-" */
				copy(l.method[:], lzhdirsMethod)
			}
		}
	default:
		filenameCase = none
		if convertCase {
			filenameCase = toLower
		}
	}

	if optionalArchiveKanjiCode != none {
		archiveKanjiCode = optionalArchiveKanjiCode
	}
	if optionalSystemKanjiCode != none {
		systemKanjiCode = optionalSystemKanjiCode
	}
	if optionalArchiveDelim != "" {
		archiveDelim = optionalArchiveDelim
	}
	if optionalSystemDelim != "" {
		systemDelim = optionalSystemDelim
	}
	if optionalFilenameCase != none {
		filenameCase = optionalFilenameCase
	}

	/* kanji code and delimiter conversion */
	convertFilename((*[]byte)(unsafe.Pointer(&l.name)), len(l.name), 1, archiveKanjiCode, systemKanjiCode, archiveDelim, systemDelim, filenameCase)

	if l.unixMode&uint16(unixFileSymlink) == uint16(unixFileSymlink) {

		/* split symbolic link */
		p := strings.Index(string(l.name[:]), "|")

		if p != -1 {
			/* l.name is symbolic link name */
			/* l.realname is real name */
			copy(l.realname[:], l.name[p:len(l.name)-1])
			/* ok */
		} else {
			return fmt.Errorf("unknown symlink name \"%s\"", l.name), false
		}
	}

	return nil, true
}

func convertFilename(name *[]byte, len, size int, fromCode, toCode int, fromDelim, toDelim string, caseTo int) {
	var i int
	key := make([]byte, 1)
	if fromCode == codeSJIS && caseTo == toLower {
		for i = 0; i < len; i++ {
			key[0] = (*name)[i]
			r, _ := utf8.DecodeRune(key)
			if unicode.IsLower(r) {
				caseTo = none
				break
			}
		}
	}
	for i = 0; i < len; i++ {
		index := strings.Index(fromDelim, string((*name)[i]))
		if index != -1 {
			//s := len(fromDelim)
			(*name)[i] = toDelim[index] //name[i] = to_delim[ptr - from_delim];
			continue
		}
		key[0] = (*name)[i]
		r, _ := utf8.DecodeRune(key)
		if caseTo == toUpper && unicode.IsLower(r) {
			(*name)[i] = byte(unicode.ToUpper(r))
			continue
		}
		if caseTo == toLower && unicode.IsUpper(r) {
			(*name)[i] = byte(unicode.ToLower(r))
			continue
		}
	}

}

/* skip SFX header */
func (l *LzHeader) seekLhaHeader(fp *io.Reader) (error, int) {
	buffer := make([]byte, 64*1024) /* max seek size */
	var p [64 * 1024]byte
	var n int
	n, err := (*fp).Read(buffer)
	if err != nil {
		return err, 0
	}

	for i := 0; i < n; i++ {
		copy(p[:], buffer[i:len(buffer)-1-i])

		if !(p[iMethod] == '-' && (p[iMethod+1] == 'l' || p[iMethod+1] == 'p') && p[iMethod+4] == '-') {
			continue
		}
		/* found "-[lp]??-" keyword (as METHOD type string) */

		/* level 0 or 1 header */
		var calcSum byte //calcSum(p+2, p[I_HEADER_SIZE])
		if (p[iHeaderLevel] == 0 || p[iHeaderLevel] == 1) && p[iHeaderSize] > 20 && p[iHeaderChecksum] == calcSum {
			//	if fseeko(fp, (p-buffer)-n, SEEK_CUR) == -1 {
			//		return fmt.Errorf("cannot seek header"), 1
			//	}
			return nil, 0
		}

		/* level 2 header */
		if p[iHeaderLevel] == 2 && p[iHeaderSize] >= 24 && p[iAttribute] == 0x20 {
			//	if fseeko(fp, (p-buffer)-n, SEEK_CUR) == -1 {
			//		return fmt.Errorf("cannot seek header"), 1
			//	}
			return nil, 0
		}
	}

	//	if fseeko(fp, -n, SEEK_CUR) == -1 {
	//		return fmt.Errorf("cannot seek header"), 1
	//	}
	return nil, -1
}

func removeLeadingDots(p string) string {
	return path.Clean(p)
}

func copyPathElement(dst, src *[]byte, size int) int {
	if size < 1 {
		return 0
	}
	var i int
	for i = 0; i < size; i++ {
		(*dst)[i] = (*src)[i]
		if (*dst)[i] == 0 {
			return i
		}
		if (*dst)[i] == '/' {
			i++
			(*dst)[i] = 0
			return i
		}
	}
	i--
	(*dst)[i] = 0
	return i
}

func canonPath(newpath, path *[]byte, size int) int {
	p := removeLeadingDots(string(*path))
	(*newpath) = []byte(p)
	return len(*newpath) - len(*path)
}

func (l LzHeader) initHeader(name string, headerLevel byte, fileinfo os.FileInfo) {
	l.packedSize = 0
	l.originalSize = int(fileinfo.Size())
	l.attribute = genericAttribute
	l.headerLevel = headerLevel
	length := canonPath((*[]byte)(unsafe.Pointer(&l.name)), (*[]byte)(unsafe.Pointer(&name)), len(name))
	l.crc = 0x0000
	l.extendType = extendUnix
	l.unixLastModifiedStamp = fileinfo.ModTime().Local().Unix()
	/* since 00:00:00 JAN.1.1970 */
	l.unixMode = uint16(fileinfo.Mode())
	if stat, ok := fileinfo.Sys().(*syscall.Stat_t); ok {
		l.unixUID = uint16(stat.Uid)
		l.unixGid = uint16(stat.Gid)
	}
	if fileinfo.IsDir() {
		copy(l.method[:], lzhdirsMethod)
		l.attribute = genericDirectoryAttribute
		l.originalSize = 0
		if length > 0 && l.name[length-1] != '/' {
			if length < len(l.name)-1 {
				length++
				l.name[length] = '/'
			} else {
				fmt.Fprintf(os.Stderr, "the length of dirname \"%s\" is too long.", l.name)
			}
		}
	}
	if fileinfo.Mode()&os.ModeSymlink == os.ModeSymlink {
		copy(l.method[:], lzhdirsMethod)
		l.attribute = genericDirectoryAttribute
		l.originalSize = 0
		max := 1024
		if len(fileinfo.Name()) < 1024 {
			max = len(fileinfo.Name()) - 1
		}
		copy(l.realname[:], fileinfo.Name()[0:max])
	}
}

func writeUnixInfo(l *LzHeader) {
	/* UNIX specific informations */

	putWord(5)    /* size */
	putByte(0x50) /* permission */
	putWord(int(l.unixMode))

	putWord(7)    /* size */
	putByte(0x51) /* gid and uid */
	putWord(int(l.unixGid))
	putWord(int(l.unixUID))

	if l.group[0] != 0 {
		length := len(l.group)
		putWord(length + 3) /* size */
		putByte(0x52)       /* group name */
		putBytes(l.group[:], length)
	}

	if l.user[0] != 0 {
		length := len(l.user)
		putWord(length + 3) /* size */
		putByte(0x53)       /* user name */
		putBytes(l.user[:], length)
	}

	if l.headerLevel == 1 {
		putWord(7)    /* size */
		putByte(0x54) /* time stamp */
		putLongWord(int(l.unixLastModifiedStamp))
	}
}

func (l *LzHeader) writeHeaderLevel0(data []byte, pathname []byte) int {
	var limit int
	var nameLength int
	var headerSize int

	setupPut((*[]byte)(unsafe.Pointer(&data)), 0)

	putByte(0x00) /* header size */
	putByte(0x00) /* check sum */
	putBytes(l.method[:], 5)
	putLongWord(l.packedSize)
	putLongWord(l.originalSize)
	//putLongWord(unixToGenericStamp(l.unixLastModifiedStamp))
	putByte(l.attribute)
	putByte(l.headerLevel) /* level 0 */

	/* write pathname (level 0 header contains the directory part) */
	nameLength = len(pathname)
	if genericFormat {
		limit = 255 - iGenericHeaderSize + 2
	} else {
		limit = 255 - iLevel0HeaderSize + 2
	}

	if nameLength > limit {
		fmt.Fprintf(os.Stderr, "the length of pathname \"%s\" is too long.", pathname)
		nameLength = limit
	}
	putByte(byte(nameLength))
	putBytes(pathname, nameLength)
	putWord(int(l.crc))

	if genericFormat {
		headerSize = iGenericHeaderSize + nameLength - 2
		data[iHeaderSize] = byte(headerSize)
		data[iHeaderChecksum] = byte(calcSum((*[]byte)(unsafe.Pointer(&data)), iMethod, headerSize))
	} else {
		/* write old-style extend header */
		putByte(extendUnix)
		putByte(currentUnixMinorVersion)
		putLongWord(int(l.unixLastModifiedStamp))
		putWord(int(l.unixMode))
		putWord(int(l.unixUID))
		putWord(int(l.unixGid))

		/* size of extended header is 12 */
		headerSize = iLevel0HeaderSize + nameLength - 2
		data[iHeaderSize] = byte(headerSize)
		data[iHeaderChecksum] = byte(calcSum((*[]byte)(unsafe.Pointer(&data)), iMethod, headerSize))
	}

	return headerSize + 2
}

func (l *LzHeader) writeHeaderLevel1(data []byte, pathname []byte) int {
	var nameLength, dirLength, limit int
	var basename, dirname []byte
	var headerSize int
	var extendHeaderTop []byte
	var extendHeaderSize int
	index := strings.Index(string(pathname), string(lhaPathsep))

	if index != 0 {
		basename = pathname[0:index]
		index++
		nameLength = len(basename)
		dirname = pathname
		dirLength = index - len(dirname)
	} else {
		basename = pathname
		nameLength = len(basename)
		dirname = []byte("")
		dirLength = 0
	}

	setupPut((*[]byte)(unsafe.Pointer(&data)), 0)

	putByte(0x00) /* header size */
	putByte(0x00) /* check sum */
	putBytes(l.method[:], 5)
	putLongWord(l.packedSize)
	putLongWord(l.originalSize)
	putLongWord(unixToGenericStamp(l.unixLastModifiedStamp))
	putByte(0x20)
	putByte(l.headerLevel) /* level 1 */

	/* level 1 header: write filename (basename only) */
	limit = 255 - iLevel1HeaderSize + 2
	if nameLength > limit {
		putByte(0) /* name length */
	} else {
		putByte(byte(nameLength))
		putBytes(basename, nameLength)
	}

	putWord(int(l.crc))

	if genericFormat {
		putByte(0x00)
	} else {
		putByte(extendUnix)
	}

	/* write extend header from here. */
	copy(extendHeaderTop[:], data[putPtr+2:putPtr+2+len(data)])
	//extendHeaderTop = putPtr + 2 /* +2 for the field `next header size' */
	headerSize = len(extendHeaderTop) // - data - 2

	/* write filename and dirname */

	if nameLength > limit {
		putWord(nameLength + 3) /* size */
		putByte(0x01)           /* filename */
		putBytes(basename, nameLength)
	}

	if dirLength > 0 {
		putWord(dirLength + 3) /* size */
		putByte(0x02)          /* dirname */
		putBytes(dirname, dirLength)
	}

	if !genericFormat {
		writeUnixInfo(l)
	}

	putWord(0x0000) /* next header size */

	extendHeaderSize = len(extendHeaderTop) - len(data[putPtr:putPtr+len(data)])
	/* On level 1 header, the packed size field is contains the ext-header */
	l.packedSize += len(extendHeaderTop) - len(data[putPtr:putPtr+len(data)])
	/* put `skip size' */
	setupPut((*[]byte)(unsafe.Pointer(&data)), iPackedSize)
	putLongWord(l.packedSize)

	data[iHeaderSize] = byte(headerSize)
	data[iHeaderChecksum] = byte(calcSum((*[]byte)(unsafe.Pointer(&data)), iMethod, headerSize))

	return headerSize + extendHeaderSize + 2
}

func (l *LzHeader) writeHeaderLevel2(data []byte, pathname []byte) int {
	var nameLength, dirLength int
	var basename, dirname []byte
	var headerSize int
	var extendHeaderTop []byte
	var headercrcPtr []byte
	var hcrc uint
	index := strings.Index(string(pathname), string(lhaPathsep))

	if index != 0 {
		basename = pathname[0:index]
		index++
		nameLength = len(basename)
		dirname = pathname
		dirLength = index - len(dirname)
	} else {
		basename = pathname
		nameLength = len(basename)
		dirname = []byte("")
		dirLength = 0
	}
	setupPut((*[]byte)(unsafe.Pointer(&data)), 0)

	putWord(0x0000) /* header size */
	putBytes(l.method[:], 5)
	putLongWord(l.packedSize)
	putLongWord(l.originalSize)
	putLongWord(int(l.unixLastModifiedStamp))
	putByte(0x20)
	putByte(l.headerLevel) /* level 2 */

	putWord(int(l.crc))

	if genericFormat {
		putByte(0x00)
	} else {
		putByte(extendUnix)
	}

	/* write extend header from here. */
	/* write extend header from here. */
	copy(extendHeaderTop[:], data[putPtr+2:putPtr+2+len(data)])
	//extendHeaderTop = putPtr + 2 /* +2 for the field `next header size' */
	headerSize = len(extendHeaderTop) // - data - 2
	//extendHeaderTop = putPtr + 2 /* +2 for the field `next header size' */

	/* write common header */
	putWord(5)
	putByte(0x00)
	copy(headercrcPtr[:], data[putPtr:putPtr+len(data)])
	//headercrcPtr = len(data[putPtr:len(data)])
	putWord(0x0000) /* header CRC */

	/* write filename and dirname */
	/* must have this header, even if the nameLength is 0. */
	putWord(nameLength + 3) /* size */
	putByte(0x01)           /* filename */
	putBytes(basename, nameLength)

	if dirLength > 0 {
		putWord(dirLength + 3) /* size */
		putByte(0x02)          /* dirname */
		putBytes(dirname, dirLength)
	}

	if !genericFormat {
		writeUnixInfo(l)
	}

	putWord(0x0000) /* next header size */

	headerSize = len(data) - len(data[putPtr:putPtr-len(data)]) //- data
	if (headerSize & 0xff) == 0 {
		/* cannot put zero at the first byte on level 2 header. */
		/* adjust header size. */
		putByte(0) /* padding */
		headerSize++
	}

	/* put header size */
	setupPut((*[]byte)(unsafe.Pointer(&data)), iHeaderSize)
	putWord(headerSize)

	/* put header CRC in extended header */
	initializeCrc(&hcrc)
	hcrc = calcCrc(hcrc, (*[]byte)(unsafe.Pointer(&data)), 0, uint(headerSize))
	setupPut((*[]byte)(unsafe.Pointer(&headercrcPtr)), 0)
	putWord(int(hcrc))

	return headerSize
}

func (l *LzHeader) writeHeader(fp *io.Writer) int {
	var headerSize int
	var data [lzheaderStorage]byte

	archiveKanjiCode := codeSJIS
	systemKanjiCode := defaultSystemKanjiCode
	var archiveDelim []byte = []byte("\377")
	var systemDelim []byte = []byte("/")
	filenameCase := none
	var pathname [filenameLength]byte

	if optionalArchiveKanjiCode != none {
		archiveKanjiCode = optionalArchiveKanjiCode
	}
	if optionalSystemKanjiCode != none {
		systemKanjiCode = optionalSystemKanjiCode
	}

	if genericFormat && convertCase {
		filenameCase = toUpper
	}

	if l.headerLevel == 0 {
		archiveDelim = []byte("\\")
	}

	if (l.unixMode & uint16(unixFileSymlink)) == uint16(unixFileSymlink) {
		var p int
		p = strings.Index(string(l.name[:]), "|")

		if p != -1 {
			fmt.Fprintf(os.Stderr, "symlink name \"%s\" contains '|' char. change it into '_'", l.name)
			l.name[p] = '_'
		}
		buf := make([]byte, 1024)
		buf = append(buf, l.name[:]...)
		buf = append(buf, '|')
		buf = append(buf, l.realname[:]...)
		copy(pathname[:], buf)
	} else {
		copy(pathname[:], l.name[:])
		pathname[len(pathname)-1] = 0
	}

	convertFilename((*[]byte)(unsafe.Pointer(&pathname)),
		len(pathname),
		2,
		systemKanjiCode,
		archiveKanjiCode,
		string(systemDelim),
		string(archiveDelim), filenameCase)

	switch l.headerLevel {
	case 0:
		headerSize = l.writeHeaderLevel0(data[:], pathname[:])
	case 1:
		headerSize = l.writeHeaderLevel1(data[:], pathname[:])
	case 2:
		headerSize = l.writeHeaderLevel2(data[:], pathname[:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown level header (level %d)", l.headerLevel)
		os.Exit(1)
	}
	n, err := (*fp).Write(data[:])
	if n == 0 || err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write to temporary file")
		os.Exit(-1)
	}
	return headerSize
}
