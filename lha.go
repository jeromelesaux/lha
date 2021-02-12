package main

type StringPool struct {
	used   int
	size   int
	n      int
	buffer []byte
}

type interfacing struct {
	infile   []byte
	outfile  []byte
	original int
	packed   int
	readSize int
	dicbit   int
	method   int
}
