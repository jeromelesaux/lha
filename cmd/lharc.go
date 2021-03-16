package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jeromelesaux/lha"
)

const ()

var (
	version            = flag.Bool("version", false, "print version of application ")
	listfile           = flag.Bool("l", false, "List ")
	listfileverbose    = flag.Bool("lv", false, "Verbose List ")
	extractfile        = flag.Bool("e", false, "EXtract from archive ")
	updatefile         = flag.Bool("u", false, "Update newer files to archive ")
	deletefile         = flag.Bool("d", false, "Delete from archive ")
	addfile            = flag.String("a", "", "Add(or replace) to archive ")
	movefile           = flag.Bool("m", false, "Move to archive ")
	createfile         = flag.String("c", "", "re-Construct new archive ")
	testcrcfile        = flag.Bool("t", false, "Test file CRC in archive ")
	verbosemodeoption  = flag.Bool("v", false, "verbose ")
	nonexecuteoption   = flag.Bool("n", false, "not execute")
	forceoption        = flag.Bool("f", false, "force (over write at extract) ")
	printstdoutarchive = flag.Bool("p", false, "Print to STDOUT from archive ")
	genericformat      = flag.Int("g", 2, " Generic format (for compatibility) 0/1/2 header level (a/u/c) ")
	compressionmethod  = flag.Int("o", 0, "o[567] compression method (a/u/c) ")
	headerlevel        = flag.Int("h", 2, "0/1/2 header level (a/u/c) ")
	excludefiles       = flag.String("x", "", "x=<pattern>  eXclude files (a/u/c), format : -x file1,file2,file3 ")
	archiveNameOption  = flag.String("archive", "", "archive file path ")

	cmd byte = lha.CmdUnknown

	timestampArchive bool

	quietMode int

	newArchive                  bool
	updateIfNewer               bool
	deleteAfterAppend           bool
	backupOldArchive            bool
	getFilenameFromStdin        bool
	recoverArchiveWhenInterrupt bool
	recursiveArchiving          bool

	temporaryFD         int
	defaultLzhuffMethod = lha.Lzhuff5MethodNum
	l                   *lha.Lha
)

func main() {

	var globalError error
	flag.Parse()

	if *archiveNameOption == "" {
		printVersion()
		return
	}

	l = lha.NewLha(*archiveNameOption)

	initVariable() /* Added N.Watazaki */

	parseOption()
	sortFiles()
	/* make crc table */
	lha.MakeCrcTable()

	switch cmd {
	case lha.CmdExtract:
		globalError = lha.CommandExtract(*archiveNameOption, l) // to be tested
	case lha.CmdAdd:
		globalError = lha.CommandAdd(*archiveNameOption, l) // to implement
	case lha.CmdList:
		globalError = lha.CommandList(*archiveNameOption, l) // to be tested
	case lha.CmdDelete:
		lha.CommadDelete(*archiveNameOption, l) // to implemennt
	default:
		fmt.Fprintf(os.Stderr, "option unknown.")
		printVersion()
	}

	if globalError != nil {
		fmt.Fprintf(os.Stderr, "Execution error :%v\n", globalError.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

/*
  Parse LHA command and options.
*/
func parseOption() {
	if *listfile {
		cmd = lha.CmdList
		l.CmdFilev = make([]string, 0)
		return
	}
	if *extractfile {
		cmd = lha.CmdExtract
		return
	}

	if *printstdoutarchive {
		l.OutputToStdout = true
		cmd = lha.CmdExtract
		return
	}

	if *createfile != "" {
		newArchive = true
		cmd = lha.CmdAdd
		l.CmdFilev = append(l.CmdFilev, *createfile)
		l.CmdFilec++
		return
	}

	if *addfile != "" {
		cmd = lha.CmdAdd
		l.CmdFilev = append(l.CmdFilev, *addfile)
		l.CmdFilec++
	}

	if *deletefile {
		cmd = lha.CmdDelete

	}

	if *updatefile {
		cmd = lha.CmdAdd
		updateIfNewer = true

	}

	if *movefile {
		cmd = lha.CmdAdd
		deleteAfterAppend = true

	}

	if *listfile {
		cmd = lha.CmdList
	}
	if *listfileverbose {
		cmd = lha.CmdList
		l.VerboseListing = true

	}

	if *testcrcfile {
		cmd = lha.CmdExtract
		lha.VerifyMode = true

	}
	parseSubOption()
	return
}

func parseSubOption() {
	if *genericformat != 5 {
		lha.GenericFormat = true
		l.HeaderLevel = 0
	}

	if *compressionmethod != 0 {
		l.CompressMethod = lha.Lzhuff1MethodNum
		l.HeaderLevel = 0
		switch *compressionmethod {
		case 5:
			l.CompressMethod = lha.Lzhuff5MethodNum
		case 6:
			l.CompressMethod = lha.Lzhuff6MethodNum
			break
		case 7:
			l.CompressMethod = lha.Lzhuff7MethodNum
		default:
			fmt.Fprintf(os.Stderr, "invalid compression method 'o%d'", *compressionmethod)
			return
		}
	}

	if *excludefiles != "" {
		l.ExcludeFiles = strings.Split(*excludefiles, ",")
	}

	if *headerlevel != 2 {
		switch *headerlevel {
		case 0:
			l.HeaderLevel = 0
		case 1:
			l.HeaderLevel = 1
		}
	}
	return
}

func printVersion() {
	flag.PrintDefaults()
	fmt.Fprintf(os.Stdout, "%s version %s (%s)\n", lha.PackageName, lha.PackageVersion, lha.PlatForm)
	//fmt.Fprintf(os.Stdout, "  configure options: %s\n", lhaConfigureOptions)
}

func initVariable() { /* Added N.Watazaki */
	/* options */
	lha.Quiet = false
	lha.TextMode = false
	l.Verbose = false
	l.Noexec = false /* debugging option */
	l.Force = false
	timestampArchive = false

	l.CompressMethod = defaultLzhuffMethod /* defined in config.h */

	l.HeaderLevel = 2 /* level 2 */
	quietMode = 0

	/* view command flags */
	l.VerboseListing = false

	/* extract command flags */
	l.OutputToStdout = false

	/* append command flags */
	l.NewArchive = false
	l.UpdateIfNewer = false
	l.DeleteAfterAppend = false
	lha.GenericFormat = false

	recoverArchiveWhenInterrupt = false
	getFilenameFromStdin = false
	l.IgnoreDirectory = false
	l.ExcludeFiles = make([]string, 0)
	lha.VerifyMode = false

	lha.ConvertCase = false

	l.ExtractDirectory = ""
	temporaryFD = -1

	l.BackupOldArchive = false

	l.ExtractBrokenArchive = false
	l.DecodeMacbinaryContents = false
	l.SortContents = true
	recursiveArchiving = true
	l.DumpLzss = false
}

func sortFiles() {
	if l.SortContents {
		sort.Strings(l.CmdFilev)
	}
}
