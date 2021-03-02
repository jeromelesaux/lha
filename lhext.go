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

	if noexec {
		return false, fmt.Errorf("EXTRACT %s but file is exist.\n", name)
	} else {
		if !force {

			switch inquire("OverWrite ?(Yes/[No]/All/Skip)", name, "YyNnAaSs\n") {
			case 0:
			case 1: /* Y/y */
				break
			case 2:
			case 3: /* N/n */
			case 8: /* Return */
				return false, fmt.Errorf("skip no response.")
			case 4:
			case 5: /* A/a */
				force = true
				break
			case 6:
			case 7: /* S/s */
				skipFlg = true
				break
			}
		}
	}

	if noexec {
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

func cmd_extract() error {
	var hdr LzHeader
	var pos int
	var afp io.Reader
	//var read_size int

	/* open archive file */
	afp, err := os.Open(archiveName)
	if err != nil {
		return fmt.Errorf("Cannot open archive file \"%s\"", archiveName)
	}

	if archiveIsMsdosSfx1([]byte(archiveName)) {
		hdr.seekLhaHeader(&afp)
	}

	/* extract each files */
	for {
		err, hasHeader := hdr.getHeader(&afp)
		if err != nil {
			return err
		}
		if !hasHeader {
			return nil
		}
		pos = 0
		if needFile(string(hdr.name[:])) {
			read_size, err := extractOne(&afp, &hdr)
			if err != nil {
				return err
			}
			if read_size != hdr.packedSize {
				/* when error occurred in extract_one(), should adjust
				   point of file stream */
				if err := skipToNextpos(&afp, pos, hdr.packedSize, read_size); err != nil {
					return fmt.Errorf("Cannot seek to next header position from \"%s\"", hdr.name)
				}
			}
		} else {
			if err := skipToNextpos(&afp, pos, hdr.packedSize, 0); err == nil {
				fmt.Errorf("Cannot seek to next header position from \"%s\"", hdr.name)
			}
		}
	}

	/* close archive file */
	afp.(*os.File).Close()

	/* adjust directory information */
	adjustDirinfo()

	return nil
}

func adjustInfo(name string, hdr *LzHeader) {

	/* adjust file stamp */
	utimebuf := time.Unix(hdr.unixLastModifiedStamp, 0)

	if (hdr.unixMode & uint16(unixFileTypemask)) != uint16(unixFileSymlink) {
		os.Chtimes(name, utimebuf, utimebuf)
	}

	if hdr.extendType == extendUnix || hdr.extendType == extendOs68k || hdr.extendType == extendXosk {

		if (hdr.unixMode & uint16(unixFileTypemask)) != uint16(unixFileTypemask) {
			os.Chmod(name, os.FileMode(hdr.unixMode))
		}

		uid := hdr.unixUID
		gid := hdr.unixGid

		os.Chown(name, int(uid), int(gid))

	}

}

func adjustDirinfo() {
	for dirinfo != nil {
		/* message("adjusting [%s]", dirinfo->hdr.name); */
		adjustInfo(string((*dirinfo).hdr.name[:]), (*dirinfo).hdr)
		dirinfo = dirinfo.next
	}
}

func skipToNextpos(fp *io.Reader, pos, off, read_size int) error {
	if pos != -1 {
		b := make([]byte, pos+off)
		_, err := (*fp).Read(b)
		if err != nil {
			return err
		}
	} else {
		b := make([]byte, off-read_size)
		_, err := (*fp).Read(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func makeNameWithPathcheck(name string, namesz int, q string) (bool, error) {

	var offset int
	if len(extractDirectory) > 0 {
		name = fmt.Sprint("%s/", extractDirectory)
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
	if string((*hdr).method[:]) != lzhdirsMethod {
		return
	}
	(*p).hdr = &LzHeader{}
	(*p).hdr.name = (*hdr).name
	top.next = dirinfo
	for tmp = top; tmp.next != nil; tmp = tmp.next {
		if (*p).hdr.name == (*tmp).next.hdr.name {
			(*p).next = (*tmp).next
			(*tmp).next = p
			break
		}
	}

	if (*tmp).next == nil {
		(*p).next = nil
		(*tmp).next = p
	}
	dirinfo = (*top).next
}

func symlinkWithMakePath(realname string, name string) int {
	var l_code int

	err := os.Symlink(realname, name)
	if err != nil {
		makeParentPath(name)
		err = os.Symlink(realname, name)
		if err != nil {
			l_code = 1
		}
	}

	return l_code
}

func extractOne(afp *io.Reader, hdr *LzHeader) (int, error) {
	var fp io.Writer
	var name [filenameLength]byte
	var crc uint
	var method int
	var save_quiet, save_verbose, upFlag bool
	var q = hdr.name
	var c byte
	var readSize int

	p := strings.Index(string(hdr.name[:]), "/")
	if ignoreDirectory && p != 1 {
		p++
	} else {
		if !isDirectoryTraversal(string(hdr.name[p:])) {
			return 0, fmt.Errorf("Possible directory traversal hack attempt in %s", hdr.name[p:])
		}

		if hdr.name[p] == '/' {
			for hdr.name[p] == '/' {
				p++
			}

			/*
			 * if OSK then strip device name
			 */
			if hdr.extendType == extendOs68k || hdr.extendType == extendXosk {
				for {
					c = hdr.name[p]
					p++
					if p >= len(hdr.name) || c == '/' {
						break
					}
				}
				if c != 0 || hdr.name[p] != 0 {
					hdr.name[p] = '.' /* if device name only */
				}
			}
		}
	}
	ok, err := makeNameWithPathcheck(string(name[:]), len(name), string(hdr.name[p:]))
	if err != nil || !ok {
		return 0, fmt.Errorf("Possible symlink traversal hack attempt in %s", q)
	}

	/* LZHDIRS_METHODを持つヘッダをチェックする */
	/* 1999.4.30 t.okamoto */
	for method = 0; ; method++ {
		if method >= len(methods) {
			return readSize, fmt.Errorf("Unknown method \"%.*s\"; \"%s\" will be skipped ...", 5, hdr.method, name)
		}
		if string(hdr.method[:]) == methods[method] {
			break
		}
	}

	if (hdr.unixMode&uint16(unixFileTypemask)) == uint16(unixFileRegular) && method != lzhdirsMethodNum {
		//	extractRegular:
		readingFilename = archiveName
		writingFilename = string(name[:])
		if outputToStdout || verifyMode {
			/* "Icon\r" should be a resource fork file encoded in MacBinary
			   format, so that it should be skipped. */
			if hdr.extendType == extendMacos && decodeMacbinaryContents && filepath.Base(string(name[:])) == "Icon\r" {
				return readSize, nil
			}

			if noexec {
				v := "EXTRACT"
				if verifyMode {
					v = "VERIFY"
				}
				fmt.Printf("%s %s\n", v, name)
				return readSize, nil
			}

			save_quiet = quiet
			save_verbose = verbose
			if !quiet && outputToStdout {
				fmt.Fprintf(os.Stdout, "::::::::\n%s\n::::::::\n", string(name[:]))
				quiet = true
				verbose = false
			} else {
				if verifyMode {
					quiet = false
					verbose = true
				}
			}
			crc = uint(decodeLzhuf(*afp,
				os.Stdout,
				hdr.originalSize,
				hdr.packedSize,
				string(name[:]),
				method,
				&readSize))
			quiet = save_quiet
			verbose = save_verbose
		} else {
			if skipFlg == false {
				upFlag, _ = inquireExtract(string(name[:]))
				if upFlag == false && force == false {
					return readSize, nil
				}
			}

			if skipFlg == true {
				_, err := os.Lstat(string(name[:]))
				if err != nil {
					return 0, err
				}
				if force != true {
					if quiet != true {
						fmt.Fprintf(os.Stderr, "%s : Skipped...\n", string(name[:]))
					}
					return readSize, nil
				}
			}
			if noexec {
				return readSize, nil
			}
			var err error
			fp, err = openWithMakePath(string(name[:]))
			if err == nil {
				crc = uint(decodeLzhuf(*afp, fp,
					hdr.originalSize, hdr.packedSize,
					string(name[:]), method, &readSize))
				fp.(*os.File).Close()
			}

			return readSize, nil
		}

		if hdr.hasCrc && crc != hdr.crc {
			return 0, fmt.Errorf("CRC error: \"%s\"", name)
		}
	} else {
		if (hdr.unixMode&uint16(unixFileTypemask)) == uint16(unixFileDirectory) || (hdr.unixMode&uint16(unixFileTypemask)) == uint16(unixFileSymlink) || method == lzhdirsMethodNum {
			/* ↑これで、Symbolic Link は、大丈夫か？ */
			if !ignoreDirectory && !verifyMode && !outputToStdout {
				if noexec {
					if quiet != true {
						fmt.Fprintf(os.Stderr, "EXTRACT %s (directory)\n", string(name[:]))
					}
					return readSize, nil
				}
				/* NAME has trailing SLASH '/', (^_^) */
				if (hdr.unixMode & uint16(unixFileTypemask)) == uint16(unixFileSymlink) {
					var lcode int
					if skipFlg == false {
						upFlag, _ = inquireExtract(string(name[:]))
						if upFlag == false && force == false {
							return readSize, nil
						}
					}

					if skipFlg == true {
						_, err := os.Lstat(string(name[:]))
						if err == nil && force != true {
							if quiet != true {
								fmt.Fprintf(os.Stderr, "%s : Skipped...\n", string(name[:]))
							}
							return readSize, nil
						}
					}

					lcode = symlinkWithMakePath(string(hdr.realname[:]), string(name[:]))
					if lcode < 0 {
						if quiet != true {
							fmt.Fprintf(os.Stderr, "Can't make Symbolic Link \"%s\" -> \"%s\"", name, hdr.realname)
						}
					}
					if quiet != true {
						fmt.Printf("Symbolic Link %s -> %s", name, hdr.realname)
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
			if force { /* force extract */
				//goto extractRegular
			} else {
				return 0, fmt.Errorf("Unknown file type: \"%s\". use `f' option to force extract.", name)
			}
		}
	}

	if !outputToStdout && !verifyMode {
		if (hdr.unixMode & uint16(unixFileTypemask)) != uint16(unixFileDirectory) {
			adjustInfo(string(name[:]), hdr)
		}
	}

	return readSize, nil

}

func needFile(name string) bool {
	for i := 0; i < len(cmdFilev); i++ {
		if cmdFilev[i] == name {
			return true
		}
	}
	return false
}
