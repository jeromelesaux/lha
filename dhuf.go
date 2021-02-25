package lha

var (
	child  [treesize]int16
	parent [treesize]int16
	block  [treesize]int16
	edge   [treesize]int16
	stock  [treesize]int16
	sNode  [treesize / 2]int16 /* Changed N.Watazaki */
	/*  node[..] -> s_node[..] */
	freq        [treesize]uint16
	totalP      uint16
	avail       int
	mostP       int
	nn          int
	nextcount   uint
	nMax        uint
	dicbit      uint16
	maxmatch    uint16
	decodeCount int
)

const (
	threshold int = 3 /* choose optimal value */
	/* dhuf.c */
	nChar     int = (256 + 60 - threshold + 1)
	treesizeC int = (nChar * 2)
	treesizeP int = (128 * 2)
	treesize  int = (treesizeC + treesizeP)
	rootC     int = 0
	rootP     int = treesizeC
)

func startCDyn( /* void */ ) {
	var i, j, f int
	n1 = 512
	if nMax >= uint(256+maxmatch-uint16(threshold)+1) {
		n1 = int(nMax) - 1
	}

	for i = 0; i < treesizeC; i++ {
		stock[i] = int16(i)
		block[i] = 0
	}
	i = 0
	j = int(nMax)*2 - 2
	for ; i < int(nMax); i++ {
		freq[j] = 1
		child[j] = int16(^i)
		sNode[i] = int16(j)
		block[j] = 1
		j--
	}
	avail = 2
	edge[1] = int16(nMax) - 1
	i = int(nMax)*2 - 2
	for j >= 0 {
		freq[j] = freq[i] + freq[i-1]
		f = int(freq[j])
		child[j] = int16(i)
		parent[i-1] = int16(j)
		parent[i] = parent[i-1]

		if f == int(freq[j+1]) {
			block[j] = block[j+1]
			edge[block[j]] = int16(j)
		} else {
			block[j] = stock[avail]
			avail++
			edge[block[j]] = int16(j)
		}
		i -= 2
		j--
	}
}

func startPDyn( /* void */ ) {
	freq[rootP] = 1
	child[rootP] = int16(^nChar)
	sNode[nChar] = int16(rootP)
	block[rootP] = stock[avail]
	avail++
	edge[block[rootP]] = int16(rootP)
	mostP = rootP
	totalP = 0
	nn = 1 << dicbit
	nextcount = 64
}

func decodeStartDyn( /* void */ ) {
	nMax = 286
	maxmatch = Maxmatch
	initGetbits()
	initCodeCache()
	startCDyn()
	startPDyn()
}

func reconst(start, end int) {
	var i, j, k, l, b int
	var f, g uint
	j = start
	for i = start; i < end; i++ {
		k = int(child[i])
		if k < 0 {
			freq[j] = (freq[i] + 1) / 2
			child[j] = int16(k)
			j++
		}
		b = int(block[i])
		if int(edge[b]) == i {
			avail--
			stock[avail] = int16(b)
		}
	}
	j--
	i = end - 1
	l = end - 2
	for i >= start {
		for i >= l {
			freq[i] = freq[j]
			child[i] = child[j]
			i--
			j--
		}
		f = uint(freq[l] + freq[l+1])
		for k = start; f < uint(freq[k]); k++ {

		}
		for j >= k {
			freq[i] = freq[j]
			child[i] = child[j]
			i--
			j--
		}
		freq[i] = uint16(f)
		child[i] = int16(l) + 1
		i--
		l -= 2
	}
	f = 0
	for i = start; i < end; i++ {
		j = int(child[i])
		if j < 0 {
			sNode[^j] = int16(i)
		} else {
			parent[j-1] = int16(i)
			parent[j] = parent[j-1]
		}
		g = uint(freq[i])
		if g == f {
			block[i] = int16(b)
		} else {
			block[i] = stock[avail]
			avail++
			b = int(block[i])
			edge[b] = int16(i)
			f = g
		}
	}
}

func swapInc(p int) int16 {
	var b, q, r, s int

	b = int(block[p])
	q = int(edge[b])
	if (q) != p { /* swap for leader */
		r = int(child[p])
		s = int(child[q])
		child[p] = int16(s)
		child[q] = int16(r)
		if r >= 0 {
			parent[r-1] = int16(q)
			parent[r] = parent[r-1]
		} else {
			sNode[^r] = int16(q)
		}
		if s >= 0 {
			parent[s-1] = int16(p)
			parent[s] = parent[s-1]
		} else {
			sNode[^s] = int16(p)
		}
		p = q
		edge[b]++
		freq[p]++
		if freq[p] == freq[p-1] {
			block[p] = block[p-1]
		} else {
			block[p] = stock[avail]
			avail++
			edge[block[p]] = int16(p) /* create block */
		}
	} else {
		if b == int(block[p+1]) {
			edge[b]++
			freq[p]++
			if freq[p] == freq[p-1] {
				block[p] = block[p-1]
			} else {
				block[p] = stock[avail]
				avail++
				edge[block[p]] = int16(p) /* create block */
			}
		} else {
			freq[p]++
			if freq[p] == freq[p-1] {
				avail--
				stock[avail] = int16(b) /* delete block */
				block[p] = block[p-1]
			}
		}
	}
	return parent[p]
}

func updateC(p int) {
	var q int

	if freq[rootC] == 0x8000 {
		reconst(0, int(nMax)*2-1)
	}
	freq[rootC]++
	q = int(sNode[p])
	for {
		q = int(swapInc(q))
		if q == rootC {
			break
		}
	}
}

func updateP(p int) {
	var q int

	if totalP == 0x8000 {
		reconst(rootP, mostP+1)
		totalP = freq[rootP]
		freq[rootP] = 0xffff
	}
	q = int(sNode[p+nChar])
	for q != rootP {
		q = int(swapInc(q))
	}
	totalP++
}

func makeNewNode(p int) {
	var q, r int

	r = mostP + 1
	q = r + 1
	child[r] = child[mostP]
	sNode[^(child[r])] = int16(r)
	child[q] = ^int16(p + nChar)
	child[mostP] = int16(q)
	freq[r] = freq[mostP]
	freq[q] = 0
	block[r] = block[mostP]
	if mostP == rootP {
		freq[rootP] = 0xffff
		edge[block[rootP]]++
	}
	parent[q] = int16(mostP)
	parent[r] = parent[q]
	mostP = q
	sNode[p+nChar] = int16(mostP)
	block[q] = stock[avail]
	avail++

	edge[block[q]] = sNode[p+nChar]
	updateP(p)
}

func encodeCDyn(c uint) {
	var bits uint
	var p, d, cnt int

	d = int(c) - n1
	if d >= 0 {
		c = uint(n1)
	}
	bits = 0
	cnt = int(bits)
	p = int(sNode[c])
	for {
		bits >>= 1
		if p&1 != 0 {
			bits |= 0x80000000
		}
		cnt++
		p = int(parent[p])
		if p == rootC {
			break
		}
	}
	if cnt <= 16 {
		putcode(byte(cnt), uint16(bits>>16))
	} else {
		putcode(16, uint16(bits>>16))
		putbits(byte(cnt-16), uint16(bits))
	}
	if d >= 0 {
		putbits(8, uint16(d))
	}
	updateC(int(c))
}

func decodeCDyn( /* void */ ) int {
	var c int
	var buf, cnt int16

	c = int(child[rootC])
	buf = int16(bitbuf)
	cnt = 0
	for {
		v := c
		if buf < 0 {
			v -= 1
		}
		c = int(child[v])
		buf <<= 1
		cnt++
		if cnt == 16 {
			fillbuf(16)
			buf = int16(bitbuf)
			cnt = 0
		}
		if c <= 0 {
			break
		}
	}
	fillbuf(byte(cnt))
	c = ^c
	updateC(c)
	if c == n1 {
		c += int(getbits(8))
	}
	return c
}

func decodePDyn( /* void */ ) uint16 {
	var c int
	var buf, cnt int16

	for decodeCount > int(nextcount) {
		makeNewNode(int(nextcount) / 64)
		nextcount += 64
		if int(nextcount) >= nn {
			nextcount = 0xffffffff
		}
	}
	c = int(child[rootP])
	buf = int16(bitbuf)
	cnt = 0
	for c > 0 {
		v := c
		if buf < 0 {
			v -= 1
		}
		c = int(child[v]) //child[c-(buf < 0)]
		buf <<= 1
		cnt++
		if cnt == 16 {
			fillbuf(16)
			buf = int16(bitbuf)
			cnt = 0
		}
	}
	fillbuf(byte(cnt))
	c = (^c) - nChar
	updateP(c)
	return (uint16(c) << 6) + getbits(6)
}

func outputDyn(code, pos int) {
	encodeCDyn(uint(code))
	if code >= 0x100 {
		encodePSt0(uint16(pos))
	}
}

/* ------------------------------------------------------------------------ */
/* lh1 */
func encodeEndDyn( /* void */ ) {
	putcode(7, 0)
}
