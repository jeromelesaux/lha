package lha

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	skipFlg                 bool = false /* FALSE..No Skip , TRUE..Skip */
	dirinfo                      = &LzHeaderList{}
	Noexec                  bool
	Force                   bool
	IgnoreDirectory         bool
	OutputToStdout          bool
	Verbose                 bool
	VerboseListing          bool
	DecodeMacbinaryContents bool
	readingFilename         string
	writingFilename         string
	archiveName             string
	CmdFilev                []string

	methods = []string{Lzhuff0Method,
		Lzhuff1Method,
		Lzhuff2Method,
		Lzhuff3Method,
		Lzhuff4Method,
		Lzhuff5Method,
		Lzhuff6Method,
		Lzhuff7Method,
		LarcMethod,
		Larc5Method,
		Larc4Method,
		LzhdirsMethod,
		Pmarc0Method,
		Pmarc2Method}
)

func isDirectoryTraversal(path string) bool {
	var state int = 0

	for i := 0; i < len(path); i++ {
		switch state {
		case 0:
			if path[i] == '.' {
				state = 1
			} else {
				state = 3
			}

		case 1:
			if path[i] == '.' {
				state = 2

			} else {
				if path[i] == '/' {
					state = 0
				} else {
					state = 3
				}
			}

		case 2:
			if path[i] == '/' {
				return true
			} else {
				state = 3
			}

		case 3:
			if path[i] == '/' {
				state = 0
			}

		}
	}
	return state == 2
}

func inquire(msg string, name string, selective string) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprintf(os.Stdout, "%s %s ", name, msg)
		response, _ := reader.ReadString('\n')

		for i := 0; i <= len(selective); i++ {
			if response[0] == selective[0] {
				return i - len(selective)
			}
		}
	}
	/* NOTREACHED */
}

func inquireExtract(name string) (bool, error) {

	skipFlg = false
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()
	stbuf, err := f.Stat()
	if err != nil {
		return false, err
	}

	if !stbuf.Mode().IsRegular() {
		return false, fmt.Errorf("\"%s\" already exists (not a file)", name)
	}

	if Noexec {
		return false, fmt.Errorf("EXTRACT %s but file is exist", name)
	} else {
		if !Force {

			switch inquire("OverWrite ?(Yes/[No]/All/Skip)", name, "YyNnAaSs\n") {
			case 0:
			case 1: /* Y/y */
				break
			case 2:
			case 3: /* N/n */
			case 8: /* Return */
				return false, fmt.Errorf("skip no response")
			case 4:
			case 5: /* A/a */
				Force = true
				break
			case 6:
			case 7: /* S/s */
				skipFlg = true
				break
			}
		}
	}

	if Noexec {
		fmt.Printf("EXTRACT %s\n", name)
	}
	return true, nil
}

func lhaExit(status int) {
	//cleanup()
	os.Exit(status)
}

func writeArchiveTail(nafp *io.Writer) {
	(*nafp).Write([]byte{0x00})
}

func CommandAdd(archiveFilepath string) error {
	archiveName = archiveFilepath

	return nil
}

func CommandList(archiveFilepath string) error {
	archiveName = archiveFilepath
	var afp *io.Reader
	var hdr LzHeader
	var i int

	var packedSizeTotal int
	var originalSizeTotal int
	var listFiles int
	var err error

	/* initialize total count */

	/* open archive file */
	f, err := os.Open(archiveName)
	if err != nil {
		return fmt.Errorf("Cannot open archive \"%s\"", archiveName)
	}
	*afp = f

	/* print header message */
	if !Quiet {
		listHeader()
	}

	/* print each file information */
	var hasHeader bool
	for {
		err, hasHeader = hdr.GetHeader(afp)
		if !hasHeader {
			break
		}
		if needFile(string(hdr.Name[:])) {
			listOne(&hdr)
			listFiles++
			packedSizeTotal += hdr.PackedSize
			originalSizeTotal += hdr.OriginalSize
		}

		i = hdr.PackedSize
		v := make([]byte, i)
		(*afp).Read(v)
	}

	/* close archive file */
	f.Close()

	/* print tailer message */
	if !Quiet {
		listTailer(listFiles, packedSizeTotal, originalSizeTotal)
	}

	return err
}

func listTailer(listFiles, packedSizeTotal, originalSizeTotal int) {
	printBar()
	v := 's'
	if listFiles == 1 {
		v = ' '
	}
	fmt.Printf(" Total %9d file%c ", listFiles, v)
	printSize(packedSizeTotal, originalSizeTotal)
	fmt.Printf(" ")
	if VerboseListing {
		fmt.Printf("           ")
	}
	printStamp(0)
	fmt.Printf("/n")
}

func printBar() {
	if VerboseListing {
		if Verbose {
			/*      PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP            LV */
			fmt.Printf("---------- ----------- ------- ------- ------ ---------- ------------------- ---\n")
		} else {
			/*      PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP     NAME */
			fmt.Printf("---------- ----------- ------- ------- ------ ---------- ------------ ----------\n")
		}
	} else {
		if Verbose {
			/*      PERMISSION  UID  GID      SIZE  RATIO     STAMP     LV */
			fmt.Printf("---------- ----------- ------- ------ ------------ ---\n")
		} else {
			/*      PERMISSION  UID  GID      SIZE  RATIO     STAMP           NAME */
			fmt.Printf("---------- ----------- ------- ------ ------------ --------------------\n")
		}
	}
}

func listHeader() {
	if VerboseListing {
		if Verbose {
			fmt.Printf("PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP            LV\n")
		} else {
			fmt.Printf("PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP     NAME\n")
		}
	} else {
		if Verbose {
			fmt.Printf("PERMISSION  UID  GID      SIZE  RATIO     STAMP     LV\n")
		} else {
			fmt.Printf("PERMISSION  UID  GID      SIZE  RATIO     STAMP           NAME\n")
		}
	}
	printBar()
}
func CommadDelete(archiveFilepath string) error {
	archiveName = archiveFilepath

	return nil
}

func CommandExtract(archiveFilepath string) error {
	var hdr LzHeader
	var pos int
	var afp *io.Reader
	//var read_size int

	archiveName = archiveFilepath

	/* open archive file */
	f, err := os.Open(archiveName)
	if err != nil {
		return fmt.Errorf("Cannot open archive file \"%s\"", archiveName)
	}
	*afp = f

	if archiveIsMsdosSfx1([]byte(archiveName)) {
		hdr.SeekLhaHeader(afp)
	}

	/* extract each files */
	for {
		err, hasHeader := hdr.GetHeader(afp)
		if err != nil {
			return err
		}
		if !hasHeader {
			return nil
		}
		pos = 0
		if needFile(string(hdr.Name[:])) {
			readSize, err := extractOne(afp, &hdr)
			if err != nil {
				return err
			}
			if readSize != hdr.PackedSize {
				/* when error occurred in extract_one(), should adjust
				   point of file stream */
				if err := skipToNextpos(afp, pos, hdr.PackedSize, readSize); err != nil {
					return fmt.Errorf("Cannot seek to next header position from \"%s\"", hdr.Name)
				}
			}
		} else {
			if err := skipToNextpos(afp, pos, hdr.PackedSize, 0); err == nil {
				fmt.Fprintf(os.Stdout, "Cannot seek to next header position from \"%s\"", hdr.Name)
			}
		}
	}

	/* close archive file */
	f.Close()

	/* adjust directory information */
	adjustDirinfo()

	return nil
}

func adjustInfo(name string, hdr *LzHeader) {

	/* adjust file stamp */
	utimebuf := time.Unix(hdr.UnixLastModifiedStamp, 0)

	if (hdr.UnixMode & uint16(UnixFileTypemask)) != uint16(UnixFileSymlink) {
		os.Chtimes(name, utimebuf, utimebuf)
	}

	if hdr.ExtendType == ExtendUnix || hdr.ExtendType == ExtendOs68k || hdr.ExtendType == ExtendXosk {

		if (hdr.UnixMode & uint16(UnixFileTypemask)) != uint16(UnixFileTypemask) {
			os.Chmod(name, os.FileMode(hdr.UnixMode))
		}

		uid := hdr.UnixUID
		gid := hdr.UnixGid

		os.Chown(name, int(uid), int(gid))

	}

}

func adjustDirinfo() {
	for dirinfo != nil {
		/* message("adjusting [%s]", dirinfo->hdr.Name); */
		adjustInfo(string((*dirinfo).Hdr.Name[:]), (*dirinfo).Hdr)
		dirinfo = dirinfo.Next
	}
}

func skipToNextpos(fp *io.Reader, pos, off, readSize int) error {
	if pos != -1 {
		b := make([]byte, pos+off)
		_, err := (*fp).Read(b)
		if err != nil {
			return err
		}
	} else {
		b := make([]byte, off-readSize)
		_, err := (*fp).Read(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func makeNameWithPathcheck(name string, namesz int, q string) (bool, error) {

	var offset int
	if len(ExtractDirectory) > 0 {
		name = fmt.Sprint("%s/", ExtractDirectory)
		offset += len(name)
	}
	var p int
	p = strings.Index(q, "/")
	for p != -1 {
		name += q[p:]
		offset += len(q) - p

		_, err := os.Lstat(name)
		if err != nil {
			return false, err
		}
		_, err = filepath.EvalSymlinks(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "this not a symlink [%s] : %v\n ", name, err)
			return false, nil
		}
		p = strings.Index(q[p:], "/")
	}

	return true, nil
}

func makeParentPath(name string) (bool, error) {
	st, err := os.Lstat(name)
	if err != nil {
		return false, err
	}
	if st.IsDir() {
		return true, nil
	}
	err = os.MkdirAll(name, 0777)
	if err != nil {
		return false, err
	}
	return true, nil
}

func openWithMakePath(name string) (io.Writer, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, fmt.Errorf("Cannot extract a file \"%s\"", name)
	}

	return f, nil
}

func addDirinfo(name string, hdr *LzHeader) {
	var p, tmp, top *LzHeaderList

	(*p) = LzHeaderList{}
	if string((*hdr).Method[:]) != LzhdirsMethod {
		return
	}
	(*p).Hdr = &LzHeader{}
	(*p).Hdr.Name = (*hdr).Name
	top.Next = dirinfo
	for tmp = top; tmp.Next != nil; tmp = tmp.Next {
		if (*p).Hdr.Name == (*tmp).Next.Hdr.Name {
			(*p).Next = (*tmp).Next
			(*tmp).Next = p
			break
		}
	}

	if (*tmp).Next == nil {
		(*p).Next = nil
		(*tmp).Next = p
	}
	dirinfo = (*top).Next
}

func symlinkWithMakePath(realname string, name string) int {
	var lCode int

	err := os.Symlink(realname, name)
	if err != nil {
		makeParentPath(name)
		err = os.Symlink(realname, name)
		if err != nil {
			lCode = 1
		}
	}

	return lCode
}

func extractOne(afp *io.Reader, hdr *LzHeader) (int, error) {
	var fp io.Writer
	var name [FilenameLength]byte
	var crc uint
	var method int
	var saveQuiet, saveVerbose, upFlag bool
	var q = hdr.Name
	var c byte
	var readSize int

	p := strings.Index(string(hdr.Name[:]), "/")
	if IgnoreDirectory && p != 1 {
		p++
	} else {
		if !isDirectoryTraversal(string(hdr.Name[p:])) {
			return 0, fmt.Errorf("Possible directory traversal hack attempt in %s", hdr.Name[p:])
		}

		if hdr.Name[p] == '/' {
			for hdr.Name[p] == '/' {
				p++
			}

			/*
			 * if OSK then strip device name
			 */
			if hdr.ExtendType == ExtendOs68k || hdr.ExtendType == ExtendXosk {
				for {
					c = hdr.Name[p]
					p++
					if p >= len(hdr.Name) || c == '/' {
						break
					}
				}
				if c != 0 || hdr.Name[p] != 0 {
					hdr.Name[p] = '.' /* if device name only */
				}
			}
		}
	}
	ok, err := makeNameWithPathcheck(string(name[:]), len(name), string(hdr.Name[p:]))
	if err != nil || !ok {
		return 0, fmt.Errorf("Possible symlink traversal hack attempt in %s", q)
	}

	/* LZHDIRS_METHODを持つヘッダをチェックする */
	/* 1999.4.30 t.okamoto */
	for method = 0; ; method++ {
		if method >= len(methods) {
			return readSize, fmt.Errorf("Unknown method \"%.*s\"; \"%s\" will be skipped", 5, hdr.Method, name)
		}
		if string(hdr.Method[:]) == methods[method] {
			break
		}
	}

	if (hdr.UnixMode&uint16(UnixFileTypemask)) == uint16(UnixFileRegular) && method != LzhdirsMethodNum {
		//	extractRegular:
		readingFilename = archiveName
		writingFilename = string(name[:])
		if OutputToStdout || VerifyMode {
			/* "Icon\r" should be a resource fork file encoded in MacBinary
			   format, so that it should be skipped. */
			if hdr.ExtendType == ExtendMacos && DecodeMacbinaryContents && filepath.Base(string(name[:])) == "Icon\r" {
				return readSize, nil
			}

			if Noexec {
				v := "EXTRACT"
				if VerifyMode {
					v = "VERIFY"
				}
				fmt.Printf("%s %s\n", v, name)
				return readSize, nil
			}

			saveQuiet = Quiet
			saveVerbose = Verbose
			if !Quiet && OutputToStdout {
				fmt.Fprintf(os.Stdout, "::::::::\n%s\n::::::::\n", string(name[:]))
				Quiet = true
				Verbose = false
			} else {
				if VerifyMode {
					Quiet = false
					Verbose = true
				}
			}
			crc = uint(DecodeLzhuf(*afp,
				os.Stdout,
				hdr.OriginalSize,
				hdr.PackedSize,
				string(name[:]),
				method,
				&readSize))
			Quiet = saveQuiet
			Verbose = saveVerbose
		} else {
			if skipFlg == false {
				upFlag, _ = inquireExtract(string(name[:]))
				if upFlag == false && Force == false {
					return readSize, nil
				}
			}

			if skipFlg == true {
				_, err := os.Lstat(string(name[:]))
				if err != nil {
					return 0, err
				}
				if Force != true {
					if Quiet != true {
						fmt.Fprintf(os.Stderr, "%s : Skipped...\n", string(name[:]))
					}
					return readSize, nil
				}
			}
			if Noexec {
				return readSize, nil
			}
			var err error
			fp, err = openWithMakePath(string(name[:]))
			if err == nil {
				crc = uint(DecodeLzhuf(
					*afp,
					fp,
					hdr.OriginalSize,
					hdr.PackedSize,
					string(name[:]),
					method,
					&readSize))
				fp.(*os.File).Close()
			}

			return readSize, nil
		}

		if hdr.HasCrc && crc != hdr.Crc {
			return 0, fmt.Errorf("CRC error: \"%s\"", name)
		}
	} else {
		if (hdr.UnixMode&uint16(UnixFileTypemask)) == uint16(UnixFileDirectory) || (hdr.UnixMode&uint16(UnixFileTypemask)) == uint16(UnixFileSymlink) || method == LzhdirsMethodNum {
			/* ↑これで、Symbolic Link は、大丈夫か？ */
			if !IgnoreDirectory && !VerifyMode && !OutputToStdout {
				if Noexec {
					if Quiet != true {
						fmt.Fprintf(os.Stderr, "EXTRACT %s (directory)\n", string(name[:]))
					}
					return readSize, nil
				}
				/* NAME has trailing SLASH '/', (^_^) */
				if (hdr.UnixMode & uint16(UnixFileTypemask)) == uint16(UnixFileSymlink) {
					var lcode int
					if skipFlg == false {
						upFlag, _ = inquireExtract(string(name[:]))
						if upFlag == false && Force == false {
							return readSize, nil
						}
					}

					if skipFlg == true {
						_, err := os.Lstat(string(name[:]))
						if err == nil && Force != true {
							if Quiet != true {
								fmt.Fprintf(os.Stderr, "%s : Skipped...\n", string(name[:]))
							}
							return readSize, nil
						}
					}

					lcode = symlinkWithMakePath(string(hdr.Realname[:]), string(name[:]))
					if lcode < 0 {
						if Quiet != true {
							fmt.Fprintf(os.Stderr, "Can't make Symbolic Link \"%s\" -> \"%s\"", name, hdr.Realname)
						}
					}
					if Quiet != true {
						fmt.Printf("Symbolic Link %s -> %s", name, hdr.Realname)
					}
				} else { /* make directory */
					ok, err := makeParentPath(string(name[:]))
					if err != nil || !ok {
						return readSize, nil
					}
					/* save directory information */
					addDirinfo(string(name[:]), hdr)
				}
			}
		} else {
			if Force { /* force extract */
				//goto extractRegular
			} else {
				return 0, fmt.Errorf("Unknown file type: \"%s\". use `f' option to force extract", name)
			}
		}
	}

	if !OutputToStdout && !VerifyMode {
		if (hdr.UnixMode & uint16(UnixFileTypemask)) != uint16(UnixFileDirectory) {
			adjustInfo(string(name[:]), hdr)
		}
	}

	return readSize, nil

}

func needFile(name string) bool {
	for i := 0; i < len(CmdFilev); i++ {
		if CmdFilev[i] == name {
			return true
		}
	}
	return false
}

func listOne(hdr *LzHeader) {
	var mode int
	var p string
	var method [6]byte
	var modebits [11]byte = [11]byte{'-', '-', '-', '-', '-', '-', '-', '-', '-', '-', '-'}

	if Verbose {
		if (int(hdr.UnixMode) & UnixFileSymlink) != UnixFileSymlink {
			fmt.Printf("%s\n", hdr.Name)
		} else {
			fmt.Printf("%s -> %s\n", hdr.Name, hdr.Realname)
		}
	}
	copy(method[:], hdr.Method[:])
	method[5] = 0

	switch hdr.ExtendType {
	case ExtendUnix:
		mode = int(hdr.UnixMode)

		if mode&UnixFileDirectory != 0 {
			modebits[0] = 'd'
		} else {
			if (mode & UnixFileSymlink) == UnixFileSymlink {
				modebits[0] = 'l'
			} else {
				modebits[0] = '-'
			}
		}

		if mode&UnixOwnerReadPerm != 0 {
			modebits[1] = 'r'
		}
		if mode&UnixOwnerWritePerm != 0 {
			modebits[2] = 'w'
		}
		if mode&UnixSetuid != 0 {
			modebits[3] = 's'
		} else {
			if mode&UnixOwnerExecPerm != 0 {
				modebits[3] = 'x'
			}
		}
		if mode&UnixGroupReadPerm != 0 {
			modebits[4] = 'r'
		}
		if mode&UnixGroupWritePerm != 0 {
			modebits[5] = 'w'
		}
		if mode&UnixSetgid != 0 {
			modebits[6] = 's'
		} else {
			if mode&UnixGroupExecPerm != 0 {
				modebits[6] = 'x'
			}
		}
		if mode&UnixOtherReadPerm != 0 {
			modebits[7] = 'r'
		}
		if mode&UnixOtherWritePerm != 0 {
			modebits[8] = 'w'
		}
		if mode&UnixStickybit != 0 {
			modebits[9] = 't'
		} else {
			if mode&UnixOtherExecPerm != 0 {
				modebits[9] = 'x'
			}
		}
		modebits[10] = 0

		fmt.Printf("%s ", modebits)

	case ExtendOs68k:
		/**/
	case ExtendXosk: /**/
		mode = int(hdr.UnixMode)
		if mode&oskDirectoryPerm != 0 {
			modebits[0] = 'd'
		}
		if mode&oskSharedPerm != 0 {
			modebits[1] = 's'
		}
		if mode&oskOtherExecPerm != 0 {
			modebits[2] = 'e'
		}
		if mode&oskOtherWritePerm != 0 {
			modebits[3] = 'w'
		}
		if mode&oskOtherReadPerm != 0 {
			modebits[4] = 'r'
		}
		if mode&oskOwnerExecPerm != 0 {
			modebits[5] = 'e'
		}
		if mode&oskOwnerWritePerm != 0 {
			modebits[6] = 'w'
		}
		if mode&oskOwnerReadPerm != 0 {
			modebits[7] = 'r'
		}

		fmt.Printf("%s ", modebits[0:8])
	default:
		switch hdr.ExtendType { /* max 18 characters */
		case ExtendGeneric:
			p = "[generic]"

		case ExtendCpm:
			p = "[CP/M]"

		case ExtendFlex:
			p = "[FLEX]"
			break
		case ExtendOs9:
			p = "[OS-9]"
			break
		case ExtendOs68k:
			p = "[OS-9/68K]"
			break
		case ExtendMsdos:
			p = "[MS-DOS]"
			break
		case ExtendMacos:
			p = "[Mac OS]"
			break
		case ExtendOs2:
			p = "[OS/2]"
			break
		case ExtendHuman:
			p = "[Human68K]"
			break
		case Extend0s386:
			p = "[OS-386]"
			break
		case ExtendRunser:
			p = "[Runser]"
			break

			/* This ID isn't fixed */
		case ExtendTownsos:
			p = "[TownsOS]"
			break

		case ExtendJava:
			p = "[JAVA]"
			break
			/* Ouch!  Please customize it's ID.  */
		default:
			p = "[unknown]"
			break
		}
		fmt.Printf("%-11.11s", p)
	}

	switch hdr.ExtendType {
	case ExtendUnix:
	case ExtendOs68k:
	case ExtendXosk:
		if hdr.User[0] != 0 {
			fmt.Printf("%5.5s/", hdr.User)
		} else {
			fmt.Printf("%5d/", hdr.UnixUID)
		}

		if hdr.Group[0] != 0 {
			fmt.Printf("%-5.5s ", hdr.Group)
		} else {
			fmt.Printf("%-5d ", hdr.UnixGid)
		}

	default:
		fmt.Printf("%12s", "")

	}

	printSize(hdr.PackedSize, hdr.OriginalSize)

	if VerboseListing {
		if hdr.HasCrc {
			fmt.Printf(" %s %04x", method, hdr.Crc)
		} else {
			fmt.Printf(" %s ****", method)
		}
	}

	fmt.Printf(" ")
	printStamp(hdr.UnixLastModifiedStamp)

	if !Verbose {
		if (hdr.UnixMode & uint16(UnixFileSymlink)) != uint16(UnixFileSymlink) {
			fmt.Printf(" %s", hdr.Name)
		} else {
			fmt.Printf(" %s -> %s", hdr.Name, hdr.Realname)
		}
	}
	if Verbose {
		fmt.Printf(" [%d]", hdr.HeaderLevel)
	}
	fmt.Printf("\n")

}

func printStamp(t int64) {
	if VerboseListing && Verbose {
		fmt.Printf("                   ") /* 19 spaces */
	} else {
		fmt.Printf("            ") /* 12 spaces */
	}
}

func printSize(packedSize, originalSize int) {
	if VerboseListing {
		fmt.Printf("%7d ", packedSize)
	}

	fmt.Printf("%7d ", originalSize)
	if originalSize == 0 {
		fmt.Printf("******")
	} else { /* Changed N.Watazaki */
		fmt.Printf("%5.1f%%", packedSize*100.0/originalSize)
	}
}
