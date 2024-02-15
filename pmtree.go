package lha

import "fmt"

type tree struct {
	root     byte
	leftarr  []byte
	rightarr []byte
}

var (
	/*	tree1left  [32]byte
		tree1right [32]byte
		tree2left  [8]byte
		tree2right [8]byte */
	tree1      = tree{root: 0, leftarr: make([]byte, 32), rightarr: make([]byte, 32)}
	tree2      = tree{root: 0, leftarr: make([]byte, 8), rightarr: make([]byte, 8)}
	tree1bound byte
	mindepth   byte
)

func (l *Lha) maketree1() {
	var i, nbits, x int
	var table1 [32]byte

	tree1bound = byte(l.getbits(5))
	mindepth = byte(l.getbits(3))
	if mindepth == 0 {
		treeSetsingle(&tree1, tree1bound-1)
	} else {
		for i = 0; i < 32; i++ {
			table1[i] = 0
		}
		nbits = int(l.getbits(3))
		for i = 0; i < int(tree1bound); i++ {
			x = int(l.getbits(byte(nbits)))
			table1[i] = byte(x) - 1 + mindepth
			if x == 0 {
				table1[i] = 0
			}
		}
		_ = treeRebuild(&tree1, tree1bound, mindepth, 31, table1[:])
	}
}

func (l *Lha) maketree2(tree2bound int) { /* in use: 5 <= tree2bound <= 8 */

	var i, count, index int
	var table2 [8]byte

	if tree1bound < 10 {
		/* tree1bound=1..8: character only, offset value is no needed. */
		/* tree1bound=9: offset value is not encoded by Huffman tree */
		return
	}

	if tree1bound == 29 && mindepth == 0 {
		/* the length value is just 256 and offset value is just 0 */
		return
	}

	/* need to build tree2 for offset value */

	for i = 0; i < 8; i++ {
		table2[i] = 0
	}
	for i = 0; i < tree2bound; i++ {
		table2[i] = byte(l.getbits(3))
	}

	index = 0
	count = 0
	for i = 0; i < tree2bound; i++ {
		if table2[i] != 0 {
			index = i
			count++
		}
	}

	if count == 1 {
		treeSetsingle(&tree2, byte(index))
	} else {
		if count > 1 {
			_ = treeRebuild(&tree2, byte(tree2bound), 1, 7, table2[:])
		}
	}
	// Note: count == 0 is possible!
	//       Excluding that possibility was a bug in version 1.

}

func (l *Lha) treeGet(t *tree) int {
	var i int
	i = int((*t).root)
	for i < 0x80 {
		i = int((*t).rightarr[i])
		if l.getbits(1) == 0 {
			i = int((*t).leftarr[i])
		}

	}
	return i & 0x7F
}

func (l *Lha) tree1Get() int {
	return l.treeGet(&tree1)
}

func (l *Lha) tree2Get() int {
	return l.treeGet(&tree2)
}

func treeSetsingle(t *tree, value byte) {
	t.root = 128 | value
}

func treeRebuild(t *tree,
	bound byte,
	mindepth byte,
	maxdepth byte,
	table []byte) error {
	var parentarr [32]byte
	var d byte
	var i, curr, empty, n int

	/* validate table */
	{
		var count [32]uint
		var total float64

		for i = 0; i < int(bound); i++ {
			if table[i] > maxdepth {
				return fmt.Errorf("bad table")
			}
			count[table[i]]++
		}
		total = 0.0
		for i = int(mindepth); i <= int(maxdepth); i++ {
			var max_leaves uint = (1 << i)
			if count[i] > max_leaves {
				return fmt.Errorf("bad table")
			}
			total += 1.0 / float64(max_leaves) * float64(count[i])
		}
		if total != 1.0 {
			/* check the Kraft's inequality */
			return fmt.Errorf("bad table")
		}
	}

	/* initialize tree */
	t.root = 0
	for i = 0; i < int(bound); i++ {
		t.leftarr[i] = 0
		t.rightarr[i] = 0
		parentarr[i] = 0
	}

	/* build tree */
	for i = 0; i < int(mindepth)-1; i++ {
		t.leftarr[i] = byte(i) + 1
		parentarr[i+1] = byte(i)
	}

	curr = int(mindepth) - 1
	empty = int(mindepth)
	for d = mindepth; d <= maxdepth; d++ {
		for i = 0; i < int(bound); i++ {
			if table[i] != d {
				continue
			}
			if t.leftarr[curr] == 0 {
				t.leftarr[curr] = byte(i) | 128
				continue
			}

			t.rightarr[curr] = byte(i) | 128
			n = 0
			for t.rightarr[curr] != 0 {
				if curr == 0 { /* root? -> done */
					return nil
				}
				curr = int(parentarr[curr])
				n++
			}

			t.rightarr[curr] = byte(empty)
			for {
				parentarr[empty] = byte(curr)
				curr = empty
				empty++

				n--
				if n == 0 {
					break
				}
				t.leftarr[curr] = byte(empty)
			}
		}

		if t.leftarr[curr] == 0 {
			t.leftarr[curr] = byte(empty)
		} else {
			t.rightarr[curr] = byte(empty)
		}

		parentarr[empty] = byte(curr)
		curr = empty
		empty++
	}

	/* unreachable */
	return fmt.Errorf("bad table")
}
