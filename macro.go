package lha

const (
	methodTypeStorage = 5
	filenameLength    = 1024

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
	backupNameExtension  string = ".bak"

	dtext []byte

	/* for filename conversion */
	none     int = 0
	codeEuc  int = 1
	codeSJIS int = 2
	codeUTF8 int = 3
	codeCAP  int = 4 /* Columbia AppleTalk Program */
	toLower  int = 1
	toUpper  int = 2

	/* ------------------------------------------------------------------------ */
	/*  LHa File Definitions                                                    */
	/* ------------------------------------------------------------------------ */
	lzhuff0Method string = "-lh0-"
	lzhuff1Method string = "-lh1-"
	lzhuff2Method string = "-lh2-"
	lzhuff3Method string = "-lh3-"
	lzhuff4Method string = "-lh4-"
	lzhuff5Method string = "-lh5-"
	lzhuff6Method string = "-lh6-"
	lzhuff7Method string = "-lh7-"
	larcMethod    string = "-lzs-"
	larc5Method   string = "-lz5-"
	larc4Method   string = "-lz4-"
	lzhdirsMethod string = "-lhd-"
	pmarc0Method  string = "-pm0-"
	pmarc2Method  string = "-pm2-"

	/* Added N.Watazaki ..V */
	lzhuff0MethodNum int = 0
	lzhuff1MethodNum int = 1
	lzhuff2MethodNum int = 2
	lzhuff3MethodNum int = 3
	lzhuff4MethodNum int = 4
	lzhuff5MethodNum int = 5
	lzhuff6MethodNum int = 6
	lzhuff7MethodNum int = 7
	larcMethodNum    int = 8
	larc5MethodNum   int = 9
	larc4MethodNum   int = 10
	lzhdirsMethodNum int = 11
	pmarc0MethodNum  int = 12
	pmarc2MethodNum  int = 13

	maxDicsiz int = (int(1) << maxDicbit)

	extendGeneric byte = 0
	extendUnix    byte = 'U'
	extendMsdos   byte = 'M'
	extendMacos   byte = 'm'
	extendOs9     byte = '9'
	extendOs2     byte = '2'
	extendOs68k   byte = 'K'
	extend0s386   byte = '3' /* OS-9000??? */
	extendHuman   byte = 'H'
	extendCpm     byte = 'C'
	extendFlex    byte = 'F'
	extendRunser  byte = 'R'
	extendAmiga   byte = 'A'
	/* this OS type is not official */

	extendTownsos byte = 'T'
	extendXosk    byte = 'X' /* OS-9 for X68000 (?) */
	extendJava    byte = 'J'

	/*---------------------------------------------------------------------------*/

	genericAttribute          byte = 0x20
	genericDirectoryAttribute byte = 0x10

	currentUnixMinorVersion byte = 0x00

	lhaPathsep byte = 0xff /* path separator of the
	   filename in lha header.
	   it should compare with
	   `unsigned char' or `int',
	   that is not '\xff', but 0xff. */

	oskRwRwRw         int = 0000033
	oskFileRegular    int = 0000000
	oskDirectoryPerm  int = 0000200
	oskSharedPerm     int = 0000100
	oskOtherExecPerm  int = 0000040
	oskOtherWritePerm int = 0000020
	oskOtherReadPerm  int = 0000010
	oskOwnerExecPerm  int = 0000004
	oskOwnerWritePerm int = 0000002
	oskOwnerReadPerm  int = 0000001

	unixFileTypemask   int = 0170000
	unixFileRegular    int = 0100000
	unixFileDirectory  int = 0040000
	unixFileSymlink    int = 0120000
	unixSetuid         int = 0004000
	unixSetgid         int = 0002000
	unixStickybit      int = 0001000
	unixOwnerReadPerm  int = 0000400
	unixOwnerWritePerm int = 0000200
	unixOwnerExecPerm  int = 0000100
	unixGroupReadPerm  int = 0000040
	unixGroupWritePerm int = 0000020
	unixGroupExecPerm  int = 0000010
	unixOtherReadPerm  int = 0000004
	unixOtherWritePerm int = 0000002
	unixOtherExecPerm  int = 0000001
	unixRwRwRw         int = 0000666

	crcpoly uint16 = 0xa001 /* crc-16 (x^16+x^15+x^2+1) */

	/* huf.c */

	pbit byte = 5 /* smallest integer such that (1 << pbit) > * np */
	tbit byte = 5 /* smallest integer such that (1 << tbit) > * nt */
	cbit byte = 9 /* smallest integer such that (1 << cbit) > * nc */

	/* larc.c */
	magic0 byte = 18
	magic5 byte = 19

	/* lharc.c */
	cmdUnknown byte = 0
	cmdExtract byte = 1
	cmdAdd     byte = 2
	cmdList    byte = 3
	cmdDelete  byte = 4

	/* shuf.c */
	n1        int = 286        /* alphabet size */
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

func sjisFirstP(c byte) bool {
	return (c >= 0x80 && c < 0xa0) || (c >= 0xe0 && c < 0xfd)
}

func sjisSecond(c byte) bool {
	return ((c >= 0x40 && c < 0xfd) && c != 0x7f)
}

func x0201KanaP(c byte) bool {
	return (0xa0 < c && c < 0xe0)
}

/*
func peekbits(n byte) int {
	return (bitbuf >> (sizeof(bitbuf)*8 - (n)))
}*/

/* crcio.c */

func initialize_crc(crc *uint) {
	(*crc) = 0
}

func updateCrc(crc *uint, c uint) uint {

	return crctable[((*crc)^(c))&0xff] ^ ((*crc) >> charBit)
}

func strequ(a, b string) bool {
	if a[0] == b[0] {
		return a == b
	}
	return false
}
