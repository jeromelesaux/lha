package lha

import "io"

type StringPool struct {
	used   int
	size   int
	n      int
	buffer []byte
}

type interfacing struct {
	infile   io.Reader
	outfile  io.Writer
	original int
	packed   int
	readSize int
	dicbit   int
	method   int
}
