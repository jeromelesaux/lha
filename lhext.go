package lha

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
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

type stringPool struct {
	Used   int
	Size   int
	N      int
	Buffer [][]byte
}

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

func inquireExtract(name string, l *Lha) (bool, error) {

	l.skipFlg = false

	stbuf, err := os.Lstat(name)

	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	if !stbuf.Mode().IsRegular() {
		return false, fmt.Errorf("\"%s\" already exists (not a file)", name)
	}

	if l.Noexec {
		return false, fmt.Errorf("EXTRACT %s but file is exist", name)
	} else {
		if !l.Force {

			switch inquire("OverWrite ?(Yes/[No]/All/Skip)", name, "YyNnAaSs\n") {
			case 0:
			case 1: /* Y/y */

			case 2:
			case 3: /* N/n */
			case 8: /* Return */
				return false, fmt.Errorf("skip no response")
			case 4:
			case 5: /* A/a */
				l.Force = true

			case 6:
			case 7: /* S/s */
				l.skipFlg = true

			}
		}
	}

	if l.Noexec {
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

func openOldArchive(l *Lha) (*os.File, error) {
	return os.Open(l.archiveName)
}

func modifyFilenameExtenstion(buffer, ext []byte, size int) []byte {
	buffer = append(buffer, ext...)
	return buffer

}

func buildStandardArchiveName(buffer, original []byte, size int) []byte {
	copy(buffer, original[:size])
	return modifyFilenameExtenstion(buffer, []byte(archiveNameExtension), size)
}

func buildTemporaryFile(l *Lha) (*os.File, error) {
	f, err := ioutil.TempFile("./", "lh")
	if err != nil {
		return nil, err
	}
	l.temporaryName = f.Name()
	return f, nil
}
func copyOldOne(oafp *io.Reader, nafp *io.Writer, hdr *LzHeader, l *Lha) error {
	if !l.Noexec {
		var v uint
		l.readingFilename = l.archiveName
		l.writingFilename = string(l.temporaryName)
		_, err := copyfile(oafp, nafp, hdr.HeaderSize+hdr.PackedSize, 0, &v)
		if err != nil {
			return err
		}

		/* directory and symlink are ignored for time-stamp archiving */
		copy(hdr.Method, "-lhd-")
		if l.mostRecent < hdr.UnixLastModifiedStamp {
			l.mostRecent = hdr.UnixLastModifiedStamp
		}
	}
	return nil
}

func initSP(sp *stringPool) {
	sp.Size = 1024 - 0
	sp.Used = 0
	sp.N = 0
	sp.Buffer = make([][]byte, sp.N)
}

func addSP(sp *stringPool, name []byte, size int) {
	for sp.Used+size > sp.Size {
		sp.Size *= 2
	}
	sp.Buffer[sp.N] = make([]byte, size)
	copy(sp.Buffer[sp.N], name)
	sp.Used += size
	sp.N++
}

func finishSP(sp *stringPool, vCount *int, vVector *[]string) {
	for i := 0; i < sp.N; i++ {
		(*vVector) = append((*vVector), string(sp.Buffer[i]))
	}
}

func findUpdateFiles(oafp *io.Reader, l *Lha) {
	name := make([]byte, FilenameLength)
	sp := stringPool{}
	hdr := NewLzHeader()
	var lenName int

	initSP(&sp)

	for {
		err, hasHeader := hdr.GetHeader(oafp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while getting header : %v\n", err.Error())
			break
		}
		if !hasHeader {
			break
		}
		if (int(hdr.UnixMode) & UnixFileTypemask) == UnixFileRegular {
			_, err = os.Lstat(string(name))

			if os.IsExist(err) { /* exist ? */
				addSP(&sp, hdr.Name, len(hdr.Name)+1)
			}
		} else {
			if (int(hdr.UnixMode) & UnixFileTypemask) == UnixFileDirectory {
				copy(name, hdr.Name) /* ok */
				lenName = len(name)
				if lenName > 0 && name[lenName-1] == '/' {
					lenName--
					name[lenName] = 0 /* strip tail '/' */
				}
				_, err = os.Lstat(string(name))

				if os.IsExist(err) { /* exist ? */
					addSP(&sp, name, lenName+1)
				}
			}
		}
		//	fseeko(oafp, hdr.packed_size, SEEK_CUR)
	}

	//fseeko(oafp, pos, SEEK_SET)

	finishSP(&sp, &l.CmdFilec, &l.CmdFilev)

}
func buildBackupName(buffer, original string) []byte {
	buffer = original
	return modifyFilenameExtenstion([]byte(buffer), []byte(backupNameExtension), len(buffer))
}
func buildBackupFile(l *Lha) error {
	return os.Rename(l.archiveName, string(l.backupArchiveName))
}

func reportArchiveNameIfDifferent(l *Lha) {
	if !Quiet && string(l.newArchiveName) == string(l.newArchiveNameBuffer) {
		fmt.Printf("New archive file is \"%s\"", l.newArchiveName)
	}
}

func removeFiles(filec int, filev []string) {
	for i := 0; i < filec; i++ {
		os.Remove(filev[i])
	}
}

func (l *Lha) addOne(fp *io.Reader, nafp *io.Writer, hdr *LzHeader) error {
	//var next_pos int
	var vOriginalSize, vPackedSize int

	l.readingFilename = string(hdr.Name)
	l.writingFilename = string(l.temporaryName)

	/* directory and symlink are ignored for time-stamp archiving */
	if string(hdr.Method) == "-lhd-" {
		if l.mostRecent < hdr.UnixLastModifiedStamp {
			l.mostRecent = hdr.UnixLastModifiedStamp
		}
	}

	if GenericFormat { /* [generic] doesn't need directory info. */
		return nil
	}

	WriteHeader(nafp, hdr) /* DUMMY */

	if (hdr.UnixMode & uint16(UnixFileSymlink)) == uint16(UnixFileSymlink) {
		if !Quiet {
			fmt.Printf("%s -> %s\t- Symbolic Link\n", hdr.Name, hdr.Realname)
		}
	}

	if hdr.OriginalSize == 0 { /* empty file, symlink or directory */
		finishIndicator2(string(hdr.Name), "Frozen", 0)
		return nil /* previous write_header is not DUMMY. (^_^) */
	}

	hdr.Crc, _ = l.EncodeLzhuf(fp, nafp, hdr.OriginalSize,
		&vOriginalSize, &vPackedSize, hdr.Name, hdr.Method)

	if vPackedSize < vOriginalSize {
		//	next_pos = 0 //ftello(nafp)
	} else { /* retry by stored method */
		var err error
		hdr.Crc, err = encodeStoredCrc(fp, nafp, hdr.OriginalSize, &vOriginalSize, &vPackedSize)
		//if ftruncate(fileno(nafp), next_pos) == -1 {
		if err != nil {
			return fmt.Errorf("cannot truncate archive error :%v", err.Error())
		}
		//}

		//if chsize(fileno(nafp), next_pos) == -1 {
		//	return fmt.Errorf("cannot truncate archive")
		//}

		copy(hdr.Method, []byte(Lzhuff0Method))
	}
	hdr.OriginalSize = vOriginalSize
	hdr.PackedSize = vPackedSize
	// go back to the beginning to set the new header
	(*nafp).(*os.File).Seek(0, io.SeekStart)
	WriteHeader(nafp, hdr)
	(*nafp).(*os.File).Seek(0, io.SeekEnd)
	return nil
}

func (l *Lha) appendIt(name string, oafp *io.Reader, nafp *io.Writer) (*io.Reader, error) {
	ahdr := NewLzHeader()
	hdr := NewLzHeader()
	var fp io.Reader
	var cmp int
	var filec int
	var filev []string
	var i int
	var stbuf os.FileInfo
	var err error

	var directory, symlink bool

	stbuf, err = os.Lstat(name)
	if err != nil {
		return oafp, fmt.Errorf("Cannot access file \"%s\" error :%v", name, err.Error())
	}

	directory = stbuf.IsDir()
	symlink = stbuf.Mode()&os.ModeSymlink != 0

	if !directory && !symlink && !l.Noexec {
		fp, err = os.Open(name)
		if err != nil {
			return oafp, fmt.Errorf("Cannot open file \"%s\": %s", name, err.Error())
		}
	}

	l.initHeader(name, stbuf, hdr)

	cmp = 0 /* avoid compiler warnings `uninitialized' */
	for *oafp != nil {
		var hasHeader bool
		err, hasHeader = ahdr.GetHeader(oafp)
		if !hasHeader {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while getting Lzheader error : %v\n", err.Error())
		}

		if !l.SortContents {
			if !l.Noexec {
				copyOldOne(oafp, nafp, ahdr, l)
			} else {
			}
			cmp = -1 /* to be -1 always */
			continue
		}

		if string(ahdr.Name) == string(hdr.Name) {
			cmp = 0
		}
		if cmp < 0 { /* SKIP */
			/* copy old to new */
			if !l.Noexec {
				copyOldOne(oafp, nafp, ahdr, l)
			}

		} else {
			if cmp == 0 { /* REPLACE */
				/* drop old archive's */

				break
			} else { /* cmp > 0, INSERT */

				break
			}
		}
	}

	if err == nil || cmp > 0 { /* not in archive */
		if l.Noexec {
			fmt.Printf("ADD %s\n", name)
		} else {
			l.addOne(&fp, nafp, hdr)
		}
	} else { /* cmp == 0 */
		if !l.UpdateIfNewer ||
			ahdr.UnixLastModifiedStamp < hdr.UnixLastModifiedStamp {
			/* newer than archive's */
			if l.Noexec {
				fmt.Printf("REPLACE %s\n", name)
			} else {
				l.addOne(&fp, nafp, hdr)
			}
		} else { /* copy old to new */
			if !l.Noexec {
				copyOldOne(oafp, nafp, ahdr, l)
			}
		}
	}

	fp.(*os.File).Close()
	//fp.Close()

	if directory && l.RecursiveArchiving { /* recursive call */
		if findFiles(name, &filec, &filev, l) {
			for i = 0; i < filec; i++ {
				oafp, err = l.appendIt(filev[i], oafp, nafp)
				if err != nil {
					return oafp, err
				}
			}
			freeFiles(filec, &filev)
		}
	}
	return oafp, nil
}

func CommandAdd(archiveFilepath string, l *Lha) error {
	l.archiveName = archiveFilepath
	var ahdr = NewLzHeader()
	var oafp io.Reader
	var nafp io.Writer
	var i int
	//	var old_header int
	var oldArchiveExist bool
	var fw *os.File

	l.mostRecent = 0

	/* exit if no operation */
	if !l.UpdateIfNewer && l.CmdFilec == 0 {
		return fmt.Errorf("No files given in argument, do nothing")
	}

	/* open old archive if exist */
	fr, err := openOldArchive(l)
	if err != nil {
		oldArchiveExist = false
	} else {
		oldArchiveExist = true
		oafp = fr
	}

	if l.UpdateIfNewer && l.CmdFilec == 0 {
		fmt.Printf("No files given in argument")
		if err != nil {
			return fmt.Errorf("archive file \"%s\" does not exists", l.archiveName)
		}
	}

	if l.newArchive && oldArchiveExist {
		fr.Close()
	}

	if fr != nil && archiveIsMsdosSfx1([]byte(l.archiveName)) {
		buildStandardArchiveName(
			l.newArchiveNameBuffer,
			[]byte(l.archiveName),
			len(l.archiveName))
		l.newArchiveName = string(l.newArchiveNameBuffer)
	} else {
		l.newArchiveName = l.archiveName
	}

	/* build temporary file */
	/* avoid compiler warnings `uninitialized' */
	if !l.Noexec {
		var err error
		fw, err = buildTemporaryFile(l)
		if err != nil {
			return err
		}
		nafp = fw
	}

	/* find needed files when automatic update */
	if l.UpdateIfNewer && l.CmdFilec == 0 {
		findUpdateFiles(&oafp, l)
	}

	/* build new archive file */
	/* cleaning arguments */
	//cleaningFiles(&CmdFilec, &CmdFilev)
	if l.CmdFilec == 0 {
		fr.Close()
		if !l.Noexec {
			fw.Close()
		}
		return nil
	}

	for i = 0; i < l.CmdFilec; i++ {
		var j int

		if l.CmdFilev[i] == l.archiveName {
			/* exclude target archive */
			fmt.Printf("specified file \"%s\" is the generating archive. skip", l.CmdFilev[i])
			for j = i; j < l.CmdFilec-1; j++ {
				l.CmdFilev[j] = l.CmdFilev[j+1]
			}
			l.CmdFilec--
			i--
			continue
		}

		/* exclude files specified by -x option */
		if len(l.ExcludeFiles) > 0 {
			for j = 0; j < len(l.ExcludeFiles); j++ {
				a := strings.ToUpper(l.ExcludeFiles[j])
				b := strings.ToUpper(filepath.Base(l.CmdFilev[i]))
				if a != b {
					pf, _ := l.appendIt(l.CmdFilev[i], &oafp, &nafp)
					oafp = *pf
				}
			}
		} else {
			pf, _ := l.appendIt(l.CmdFilev[i], &oafp, &nafp)
			oafp = *pf
		}

	}

	if fr != nil {
		//	old_header = ftello(oafp)
		for {
			_, hasHeader := (*ahdr).GetHeader(&oafp)
			if !hasHeader {
				break
			}
			if !l.Noexec {
				//fseeko(oafp, old_header, SEEK_SET)
				copyOldOne(&oafp, &nafp, ahdr, l)
			}
			//	old_header = ftello(oafp)
		}
		fr.Close()
	}

	//newArchiveSize := 0 /* avoid compiler warnings `uninitialized' */
	if !l.Noexec {
		//	var tmp int

		writeArchiveTail(&nafp)
		//	tmp = ftello(nafp)
		//	if tmp == -1 {
		//		warning("ftello(): %s", strerror(errno))
		//		new_archive_size = 0
		//	} else {
		//		new_archive_size = tmp
		//	}
		//	fclose(nafp)
		fw.Close()
	}

	/* build backup archive file */
	if oldArchiveExist && l.BackupOldArchive {
		if err := buildBackupFile(l); err != nil {
			fmt.Fprintf(os.Stderr, "error while buildBackupFile : %v\n", err.Error())
		}
	}

	reportArchiveNameIfDifferent(l)

	/* copy temporary file to new archive file */
	if !l.Noexec {

		if len(l.newArchiveName) >= 0 {
			err := os.Rename(l.temporaryName, l.newArchiveName)
			if err != nil {
				return fmt.Errorf("error while renaming file error : %v", err.Error())
				//	temporaryToNewArchiveFile(newArchiveSize)
			}
		}

		/* set new archive file mode/group */
		setArchiveFileMode()
	}

	/* remove archived files */
	if l.DeleteAfterAppend {
		removeFiles(l.CmdFilec, l.CmdFilev)
	}

	return nil
}

func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func setArchiveFileMode() {

}

func CommandList(archiveFilepath string, l *Lha) error {
	l.archiveName = archiveFilepath
	var afp io.Reader
	hdr := NewLzHeader()
	var i int

	var packedSizeTotal int
	var originalSizeTotal int
	var listFiles int
	var err error

	/* initialize total count */

	/* open archive file */
	f, err := os.Open(l.archiveName)
	if err != nil {
		return fmt.Errorf("Cannot open archive \"%s\"", l.archiveName)
	}
	afp = f

	/* print header message */
	if !Quiet {
		listHeader(l)
	}

	/* print each file information */
	var hasHeader bool
	for {
		err, hasHeader = hdr.GetHeader(&afp)
		if !hasHeader {
			break
		}
		if l.needFile(string(hdr.Name[:])) {
			l.listOne(hdr)
			listFiles++
			packedSizeTotal += hdr.PackedSize
			originalSizeTotal += hdr.OriginalSize
		}

		i = hdr.PackedSize
		v := make([]byte, i)
		afp.Read(v)
	}

	/* close archive file */
	f.Close()

	/* print tailer message */
	if !Quiet {
		listTailer(listFiles, packedSizeTotal, originalSizeTotal, l)
	}

	return err
}

func listTailer(listFiles, packedSizeTotal, originalSizeTotal int, l *Lha) {
	printBar(l)
	v := 's'
	if listFiles == 1 {
		v = ' '
	}
	fmt.Printf(" Total %9d file%c ", listFiles, v)
	printSize(packedSizeTotal, originalSizeTotal, l)
	fmt.Printf(" ")
	if l.VerboseListing {
		fmt.Printf("           ")
	}
	printStamp(0, l)
	fmt.Printf("\n")
}

func printBar(l *Lha) {
	if l.VerboseListing {
		if l.Verbose {
			/*      PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP            LV */
			fmt.Printf("---------- ----------- ------- ------- ------ ---------- ------------------- ---\n")
		} else {
			/*      PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP     NAME */
			fmt.Printf("---------- ----------- ------- ------- ------ ---------- ------------ ----------\n")
		}
	} else {
		if l.Verbose {
			/*      PERMISSION  UID  GID      SIZE  RATIO     STAMP     LV */
			fmt.Printf("---------- ----------- ------- ------ ------------ ---\n")
		} else {
			/*      PERMISSION  UID  GID      SIZE  RATIO     STAMP           NAME */
			fmt.Printf("---------- ----------- ------- ------ ------------ --------------------\n")
		}
	}
}

func listHeader(l *Lha) {
	if l.VerboseListing {
		if l.Verbose {
			fmt.Printf("PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP            LV\n")
		} else {
			fmt.Printf("PERMISSION  UID  GID    PACKED    SIZE  RATIO METHOD CRC     STAMP     NAME\n")
		}
	} else {
		if l.Verbose {
			fmt.Printf("PERMISSION  UID  GID      SIZE  RATIO     STAMP     LV\n")
		} else {
			fmt.Printf("PERMISSION  UID  GID      SIZE  RATIO     STAMP           NAME\n")
		}
	}
	printBar(l)
}
func CommadDelete(archiveFilepath string, l *Lha) error {
	l.archiveName = archiveFilepath

	return nil
}

func CommandExtract(archiveFilepath string, l *Lha) error {
	hdr := NewLzHeader()
	var pos int
	var afp io.Reader
	//var read_size int

	l.archiveName = archiveFilepath

	/* open archive file */
	f, err := os.Open(l.archiveName)
	if err != nil {
		return fmt.Errorf("Cannot open archive file \"%s\"", l.archiveName)
	}
	afp = f

	if archiveIsMsdosSfx1([]byte(l.archiveName)) {
		hdr.SeekLhaHeader(&afp)
	}

	/* extract each files */
	for {
		err, hasHeader := hdr.GetHeader(&afp)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !hasHeader {
			return nil
		}
		pos = 0
		if l.needFile(string(hdr.Name[:])) {
			readSize, err := l.extractOne(&afp, hdr)
			if err != nil {
				return err
			}
			if readSize != hdr.PackedSize {
				/* when error occurred in extract_one(), should adjust
				   point of file stream */
				if err := skipToNextpos(&afp, pos, hdr.PackedSize, readSize); err != nil {
					return fmt.Errorf("Cannot seek to next header position from \"%s\"", hdr.Name)
				}
			}
		} else {
			if err := skipToNextpos(&afp, pos, hdr.PackedSize, 0); err == nil {
				fmt.Fprintf(os.Stdout, "Cannot seek to next header position from \"%s\"", hdr.Name)
			}
		}
	}

	/* close archive file */
	f.Close()

	/* adjust directory information */
	l.adjustDirinfo()

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

func (l *Lha) adjustDirinfo() {
	for l.dirinfo != nil && l.dirinfo.Hdr != nil {
		/* message("adjusting [%s]", dirinfo->hdr.Name); */
		adjustInfo(string((*l.dirinfo).Hdr.Name[:]), (*l.dirinfo).Hdr)
		l.dirinfo = l.dirinfo.Next
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

func makeNameWithPathcheck(name string, namesz int, q string, l *Lha) (bool, []byte, error) {

	var offset int
	if len(l.ExtractDirectory) > 0 {
		name = l.ExtractDirectory + "/"
		offset += len(name)
	}
	var p int
	p = strings.Index(q, "/")
	for p != -1 {
		name += q[p:]
		offset += len(q) - p

		_, err := os.Lstat(name)
		if err != nil {
			return false, []byte(name), err
		}
		_, err = filepath.EvalSymlinks(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "this not a symlink [%s] : %v\n ", name, err)
			return false, []byte(name), nil
		}
		p = strings.Index(q[p:], "/")
	}
	last := namesz - offset
	if offset == 0 {
		last = len(q) - 1
	}
	name = string(q[offset:last])
	return true, []byte(name), nil
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

func (l *Lha) addDirinfo(name string, hdr *LzHeader) {
	var p, tmp, top *LzHeaderList

	(*p) = LzHeaderList{}
	if string((*hdr).Method[:]) != LzhdirsMethod {
		return
	}
	(*p).Hdr = &LzHeader{}
	(*p).Hdr.Name = (*hdr).Name
	top.Next = l.dirinfo
	for tmp = top; tmp.Next != nil; tmp = tmp.Next {
		if string((*p).Hdr.Name) == string((*tmp).Next.Hdr.Name) {
			(*p).Next = (*tmp).Next
			(*tmp).Next = p
			break
		}
	}

	if (*tmp).Next == nil {
		(*p).Next = nil
		(*tmp).Next = p
	}
	l.dirinfo = (*top).Next
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

func (l *Lha) extractOne(afp *io.Reader, hdr *LzHeader) (int, error) {
	var fp io.Writer
	name := make([]byte, FilenameLength)
	var crc uint
	var method int
	var saveQuiet, saveVerbose, upFlag bool
	var q = hdr.Name
	var c byte
	var readSize int

	p := strings.LastIndex(string(hdr.Name[:]), "/")
	if p == -1 {
		p = 0
	}
	if l.IgnoreDirectory {
		p++
	} else {

		if isDirectoryTraversal(string(hdr.Name[p:])) {
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
	ok, name, err := makeNameWithPathcheck(string(name[:]), len(name), string(hdr.Name[p:]), l)
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
		l.readingFilename = l.archiveName
		l.writingFilename = string(name[:])
		if l.OutputToStdout || VerifyMode {
			/* "Icon\r" should be a resource fork file encoded in MacBinary
			   format, so that it should be skipped. */
			if hdr.ExtendType == ExtendMacos && l.DecodeMacbinaryContents && filepath.Base(string(name[:])) == "Icon\r" {
				return readSize, nil
			}

			if l.Noexec {
				v := "EXTRACT"
				if VerifyMode {
					v = "VERIFY"
				}
				fmt.Printf("%s %s\n", v, name)
				return readSize, nil
			}

			saveQuiet = Quiet
			saveVerbose = l.Verbose
			if !Quiet && l.OutputToStdout {
				fmt.Fprintf(os.Stdout, "::::::::\n%s\n::::::::\n", string(name[:]))
				Quiet = true
				l.Verbose = false
			} else {
				if VerifyMode {
					Quiet = false
					l.Verbose = true
				}
			}
			crc = uint(l.DecodeLzhuf(*afp,
				os.Stdout,
				hdr.OriginalSize,
				hdr.PackedSize,
				string(name[:]),
				method,
				&readSize))
			Quiet = saveQuiet
			l.Verbose = saveVerbose
		} else {
			if l.skipFlg == false {

				upFlag, _ = inquireExtract(string(name[:]), l)
				if upFlag == false && l.Force == false {
					return readSize, nil
				}
			}

			if l.skipFlg == true {
				_, err := os.Lstat(string(name[:]))
				if err != nil && os.IsExist(err) {
					return 0, err
				}
				if l.Force != true {
					if Quiet != true {
						fmt.Fprintf(os.Stderr, "%s : Skipped...\n", string(name[:]))
					}
					return readSize, nil
				}
			}
			if l.Noexec {
				return readSize, nil
			}
			var err error
			fp, err = openWithMakePath(string(name[:]))
			if err == nil {
				crc = uint(l.DecodeLzhuf(
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
			if !l.IgnoreDirectory && !VerifyMode && !l.OutputToStdout {
				if l.Noexec {
					if Quiet != true {
						fmt.Fprintf(os.Stderr, "EXTRACT %s (directory)\n", string(name[:]))
					}
					return readSize, nil
				}
				/* NAME has trailing SLASH '/', (^_^) */
				if (hdr.UnixMode & uint16(UnixFileTypemask)) == uint16(UnixFileSymlink) {
					var lcode int
					if l.skipFlg == false {
						upFlag, _ = inquireExtract(string(name[:]), l)
						if upFlag == false && l.Force == false {
							return readSize, nil
						}
					}

					if l.skipFlg == true {
						_, err := os.Lstat(string(name[:]))
						if err == nil && l.Force != true {
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
					l.addDirinfo(string(name[:]), hdr)
				}
			}
		} else {
			if l.Force { /* force extract */
				//goto extractRegular
			} else {
				return 0, fmt.Errorf("Unknown file type: \"%s\". use `f' option to force extract", name)
			}
		}
	}

	if !l.OutputToStdout && !VerifyMode {
		if (hdr.UnixMode & uint16(UnixFileTypemask)) != uint16(UnixFileDirectory) {
			adjustInfo(string(name[:]), hdr)
		}
	}

	return readSize, nil

}

func (l *Lha) needFile(name string) bool {
	if l.CmdFilec == 0 {
		return true
	}
	for i := 0; i < len(l.CmdFilev); i++ {
		if l.CmdFilev[i] == name {
			return true
		}
	}
	return false
}

func (l *Lha) listOne(hdr *LzHeader) {
	var mode int
	var p string
	var method [6]byte
	var modebits [11]byte = [11]byte{'-', '-', '-', '-', '-', '-', '-', '-', '-', '-', '-'}

	if l.Verbose {
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

	printSize(hdr.PackedSize, hdr.OriginalSize, l)

	if l.VerboseListing {
		if hdr.HasCrc {
			fmt.Printf(" %s %04x", method, hdr.Crc)
		} else {
			fmt.Printf(" %s ****", method)
		}
	}

	fmt.Printf(" ")
	printStamp(hdr.UnixLastModifiedStamp, l)

	if !l.Verbose {
		if (hdr.UnixMode & uint16(UnixFileSymlink)) != uint16(UnixFileSymlink) {
			fmt.Printf(" %s", hdr.Name)
		} else {
			fmt.Printf(" %s -> %s", hdr.Name, hdr.Realname)
		}
	}
	if l.Verbose {
		fmt.Printf(" [%d]", hdr.HeaderLevel)
	}
	fmt.Printf("\n")

}

func printStamp(t int64, l *Lha) {
	if l.VerboseListing && l.Verbose {
		fmt.Printf("                   ") /* 19 spaces */
	} else {
		fmt.Printf("            ") /* 12 spaces */
	}
}

func printSize(packedSize, originalSize int, l *Lha) {
	if l.VerboseListing {
		fmt.Printf("%7d ", packedSize)
	}

	fmt.Printf("%7d ", originalSize)
	if originalSize == 0 {
		fmt.Printf("******")
	} else { /* Changed N.Watazaki */
		fmt.Printf("%5.1f%%", float64(packedSize)*100.0/float64(originalSize))
	}
}

func freeFiles(filec int, filev *[]string) {
	freeSp(filev)
}

func freeSp(filev *[]string) {
	(*filev) = (*filev)[:0]
}

func findFiles(name string, vfilec *int, vfilev *[]string, l *Lha) bool {

	sp := &stringPool{}
	newname := make([]byte, FilenameLength)

	copy(newname, []byte(name))
	length := len(newname)
	if length > 0 && newname[length-1] != '/' {
		newname = append(newname, '/')
	}

	fileinfo, _ := os.Lstat(name)
	if !fileinfo.IsDir() {
		return false
	}

	initSP(sp)
	files, err := ioutil.ReadDir(name)
	if err != nil {
		return false
	}
	if len(files) == 0 {
		return false
	}
	for j := 0; j < len(files); j++ {
		var toExclude bool
		// are they excluded ?
		for i := 0; i < len(l.ExcludeFiles); i++ {
			a := strings.ToUpper(l.ExcludeFiles[i])
			b := strings.ToUpper(files[j].Name())
			if a == b {
				toExclude = true
				break
			}
		}
		if !toExclude {
			newname := filepath.Join(name, files[j].Name())
			addSP(sp, []byte(newname), len(newname))
		}
	}
	finishSP(sp, vfilec, vfilev)
	sort.Strings(*vfilev)
	//cleaningFiles(vfilec, vfilev)
	return true
}
