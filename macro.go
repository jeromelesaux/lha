package lha

const (
	PackageName       = "LHa for UNIX"
	PackageVersion    = "1.0"
	PlatForm          = "_____"
	methodTypeStorage = 5
	FilenameLength    = 1024

	lzheaderStorage      = 4096
	ucharMax             = (0x7F*2 + 1)
	charBit         byte = 8
	/*      #if nt > np  npt nt #else  npt np #endif  */
	Npt byte = 0x80
	/* slide.c */
	Maxmatch uint16 = 256 /* formerly f (not more than uchar_max + 1) */
	Nc       uint16 = (ucharMax + Maxmatch + 2 - uint16(threshold))
	ushrtBit byte   = 16 /* (char_bit * sizeof(ushort)) */
	Nt       uint16 = uint16(ushrtBit + 3)
	/* Added N.Watazaki ..^ */

	lzhuff0Dicbit int = 0  /* no compress */
	lzhuff1Dicbit int = 12 /* 2^12 =  4kb sliding dictionary */
	lzhuff2Dicbit int = 13 /* 2^13 =  8kb sliding dictionary */
	lzhuff3Dicbit int = 13 /* 2^13 =  8kb sliding dictionary */
	lzhuff4Dicbit int = 12 /* 2^12 =  4kb sliding dictionary */
	lzhuff5Dicbit int = 13 /* 2^13 =  8kb sliding dictionary */
	lzhuff6Dicbit int = 15 /* 2^15 = 32kb sliding dictionary */
	lzhuff7Dicbit int = 16 /* 2^16 = 64kb sliding dictionary */
	larcDicbit    int = 11 /* 2^11 =  2kb sliding dictionary */
	larc5Dicbit   int = 12 /* 2^12 =  4kb sliding dictionary */
	larc4Dicbit   int = 0  /* no compress */
	pmarc0Dicbit  int = 0  /* no compress */
	pmarc2Dicbit  int = 13 /* 2^13 =  8kb sliding dictionary */

	maxDicbit int = lzhuff7Dicbit
	Np        int = (maxDicbit + 1)
)

var (
	crctable             [ucharMax + 1]uint
	archiveNameExtension string = ".lzh"
	// nolint: unused
	backupNameExtension string = ".bak"

	dtext []byte

	/* for filename conversion */
	none int = 0
	// nolint: unused
	codeEuc  int = 1
	codeSJIS int = 2
	// nolint: unused
	codeUTF8 int = 3
	// nolint: unused
	codeCAP int = 4 /* Columbia AppleTalk Program */
	toLower int = 1
	toUpper int = 2

	/* ------------------------------------------------------------------------ */
	/*  LHa File Definitions                                                    */
	/* ------------------------------------------------------------------------ */
	Lzhuff0Method string = "-lh0-"
	Lzhuff1Method string = "-lh1-"
	Lzhuff2Method string = "-lh2-"
	Lzhuff3Method string = "-lh3-"
	Lzhuff4Method string = "-lh4-"
	Lzhuff5Method string = "-lh5-"
	Lzhuff6Method string = "-lh6-"
	Lzhuff7Method string = "-lh7-"
	LarcMethod    string = "-lzs-"
	Larc5Method   string = "-lz5-"
	Larc4Method   string = "-lz4-"
	LzhdirsMethod string = "-lhd-"
	Pmarc0Method  string = "-pm0-"
	Pmarc2Method  string = "-pm2-"

	/* Added N.Watazaki ..V */
	Lzhuff0MethodNum int = 0
	Lzhuff1MethodNum int = 1
	Lzhuff2MethodNum int = 2
	Lzhuff3MethodNum int = 3
	Lzhuff4MethodNum int = 4
	Lzhuff5MethodNum int = 5
	Lzhuff6MethodNum int = 6
	Lzhuff7MethodNum int = 7
	LarcMethodNum    int = 8
	Larc5MethodNum   int = 9
	Larc4MethodNum   int = 10
	LzhdirsMethodNum int = 11
	Pmarc0MethodNum  int = 12
	Pmarc2MethodNum  int = 13

	maxDicsiz int = (int(1) << maxDicbit)

	ExtendGeneric byte = 0
	ExtendUnix    byte = 'U'
	ExtendMsdos   byte = 'M'
	ExtendMacos   byte = 'm'
	ExtendOs9     byte = '9'
	ExtendOs2     byte = '2'
	ExtendOs68k   byte = 'K'
	Extend0s386   byte = '3' /* OS-9000??? */
	ExtendHuman   byte = 'H'
	ExtendCpm     byte = 'C'
	ExtendFlex    byte = 'F'
	ExtendRunser  byte = 'R'
	ExtendAmiga   byte = 'A'
	/* this OS type is not official */

	ExtendTownsos byte = 'T'
	ExtendXosk    byte = 'X' /* OS-9 for X68000 (?) */
	ExtendJava    byte = 'J'

	/*---------------------------------------------------------------------------*/

	genericAttribute          byte = 0x20
	genericDirectoryAttribute byte = 0x10

	currentUnixMinorVersion byte = 0x00

	lhaPathsep byte = 0xff /* path separator of the
	   filename in lha header.
	   it should compare with
	   `unsigned char' or `int',
	   that is not '\xff', but 0xff. */
	// nolint: unused
	oskRwRwRw int = 0000033
	// nolint: unused
	oskFileRegular    int = 0000000
	oskDirectoryPerm  int = 0000200
	oskSharedPerm     int = 0000100
	oskOtherExecPerm  int = 0000040
	oskOtherWritePerm int = 0000020
	oskOtherReadPerm  int = 0000010
	oskOwnerExecPerm  int = 0000004
	oskOwnerWritePerm int = 0000002
	oskOwnerReadPerm  int = 0000001

	UnixFileTypemask   int = 0170000
	UnixFileRegular    int = 0100000
	UnixFileDirectory  int = 0040000
	UnixFileSymlink    int = 0120000
	UnixSetuid         int = 0004000
	UnixSetgid         int = 0002000
	UnixStickybit      int = 0001000
	UnixOwnerReadPerm  int = 0000400
	UnixOwnerWritePerm int = 0000200
	UnixOwnerExecPerm  int = 0000100
	UnixGroupReadPerm  int = 0000040
	UnixGroupWritePerm int = 0000020
	UnixGroupExecPerm  int = 0000010
	UnixOtherReadPerm  int = 0000004
	UnixOtherWritePerm int = 0000002
	UnixOtherExecPerm  int = 0000001
	UnixRwRwRw         int = 0000666

	crcpoly uint = 0xA001 /* crc-16 (x^16+x^15+x^2+1) */

	/* huf.c */

	pbit byte = 5 /* smallest integer such that (1 << pbit) > * np */
	tbit byte = 5 /* smallest integer such that (1 << tbit) > * nt */
	cbit byte = 9 /* smallest integer such that (1 << cbit) > * nc */

	/* larc.c */
	magic0 byte = 18
	magic5 byte = 19

	/* lharc.c */
	CmdUnknown byte = 0
	CmdExtract byte = 1
	CmdAdd     byte = 2
	CmdList    byte = 3
	CmdDelete  byte = 4

	/* shuf.c */
	n1 int = 286 /* alphabet size */
	// nolint: unused
	n2        int = (2*n1 - 1) /* # of nodes in huffman tree */
	extrabits int = 8          /* >= log2(f-threshold+258-n1) */
	bufbits   int = 16         /* >= log2(maxbuf) */

	/* util.c */
	buffersize int = 2048

	/* slide.c */
	/*
	   percolate  1
	   nil        0
	   hash(p, c) ((p) + ((c) << hash1) + hash2)
	*/

)

// nolint: unused
func sjisFirstP(c byte) bool {
	return (c >= 0x80 && c < 0xa0) || (c >= 0xe0 && c < 0xfd)
}

// nolint: unused
func sjisSecond(c byte) bool {
	return ((c >= 0x40 && c < 0xfd) && c != 0x7f)
}

// nolint: unused
func x0201KanaP(c byte) bool {
	return (0xa0 < c && c < 0xe0)
}

/*
func peekbits(n byte) int {
	return (bitbuf >> (sizeof(bitbuf)*8 - (n)))
}*/

/* crcio.c */

func initializeCrc(crc *uint) {
	(*crc) = 0
}

func updateCrc(crc uint, c byte) uint {

	return crctable[(crc^uint(c))&0xff] ^ (crc >> uint(charBit))
}

// nolint: unused
func strequ(a, b string) bool {
	if a[0] == b[0] {
		return a == b
	}
	return false
}
