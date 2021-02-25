package lha

import "unsafe"

func makeCode(nchar int, bitlen *[]byte, code *[]uint16, leafNum *[]uint16) {
	var (
		weight [17]uint16 /* 0x10000ul >> bitlen */
		start  [17]uint16 /* start code */
		total  uint16
		i      int
		c      int
	)
	total = 0
	for i = 1; i <= 16; i++ {
		start[i] = total
		weight[i] = 1 << (16 - i)
		total += weight[i] * (*leafNum)[i]
	}
	for c = 0; c < nchar; c++ {
		i = int((*bitlen)[c])
		(*code)[c] = start[i]
		start[i] += weight[i]
	}
}

func countLeaf(node, nchar int, leaf_num []uint16, depth int) /* call with node = root */ {
	if node < nchar {
		if depth < 16 {
			leaf_num[depth]++
		} else {
			leaf_num[16]++
		}
	} else {
		countLeaf(int(left[node]), nchar, leaf_num, depth+1)
		countLeaf(int(right[node]), nchar, leaf_num, depth+1)
	}
}

func makeLen(nchar int, bitlen *[]byte, sort *[]uint16, leafNum *[]uint16) {
	var (
		i, k      int
		cum       uint
		sortIndex int
	)
	cum = 0
	for i = 16; i > 0; i-- {
		cum += uint((*leafNum)[i]) << (16 - i)
	}
	cum &= 0xffff
	/* adjust len */
	if cum != 0 {
		(*leafNum)[16] -= uint16(cum) /* always leaf_num[16] > cum */
		for cum != 0 {
			for i = 15; i > 0; i-- {
				if (*leafNum)[i] != 0 {
					(*leafNum)[i]--
					(*leafNum)[i+1] += 2
					break
				}
			}
			cum--
		}
	}
	/* make len */
	for i = 16; i > 0; i-- {
		k = int((*leafNum)[i])
		for k > 0 {
			(*bitlen)[(*sort)[sortIndex]] = byte(i)
			sortIndex++
			k--
		}
	}
}

/* priority queue; send i-th entry down heap */
func downheap(i int, heap *[]int16, heapsize int, freq *[]uint16) {
	var j, k uint16

	k = uint16((*heap)[i])
	j = 2 * uint16(i)
	for j <= uint16(heapsize) {
		if j < uint16(heapsize) && (*freq)[(*heap)[j]] > (*freq)[(*heap)[j+1]] {
			j++
		}
		if (*freq)[k] <= (*freq)[(*heap)[j]] {
			break
		}
		(*heap)[i] = (*heap)[j]
		i = int(j)
		j = 2 * uint16(i)
	}
	(*heap)[i] = int16(k)
}

/* make tree, calculate bitlen[], return root */
func makeTree(nchar int, freq *[]uint16, bitlen *[]byte, code *[]uint16) int16 {
	var (
		i, j, avail, root int16
		sortIndex         int
		heap              [Nc + 1]int16 /* NC >= nchar */
		heapsize          int
	)

	avail = int16(nchar)
	heapsize = 0
	heap[1] = 0
	for i = 0; i < int16(nchar); i++ {
		(*bitlen)[i] = 0
		if (*freq)[i] != 0 {
			heapsize++
			heap[heapsize] = i
		}
	}
	if heapsize < 2 {
		(*code)[heap[1]] = 0
		return heap[1]
	}

	/* make priority queue */
	for i = int16(heapsize) / 2; i >= 1; i-- {
		downheap(int(i), (*[]int16)(unsafe.Pointer(&heap)), heapsize, freq)
	}

	/* make huffman tree */
	sortIndex = 0
	for { /* while queue has at least two entries */
		i = heap[1] /* take out least-freq entry */
		if i < int16(nchar) {
			(*code)[sortIndex] = uint16(i)
			sortIndex++
			// *sort++ = i;
		}
		heap[1] = heap[heapsize]
		heapsize--
		downheap(1, (*[]int16)(unsafe.Pointer(&heap)), heapsize, freq)
		j = heap[1] /* next least-freq entry */
		if j < int16(nchar) {
			(*code)[sortIndex] = uint16(j)
			sortIndex++
			//  *sort++ = j;
		}
		root = avail /* generate new node */
		avail++
		(*freq)[root] = (*freq)[i] + (*freq)[j]
		heap[1] = root
		downheap(1, (*[]int16)(unsafe.Pointer(&heap)), heapsize, freq) /* put into queue */
		left[root] = uint16(i)
		right[root] = uint16(j)
		if heapsize <= 1 {
			break
		}
	}

	{
		var leaf_num [17]uint16

		/* make leaf_num */

		countLeaf(int(root), nchar, leaf_num[:], 0)

		/* make bitlen */
		makeLen(nchar, bitlen, code, (*[]uint16)(unsafe.Pointer(&leaf_num)))

		/* make code table */
		makeCode(nchar, bitlen, code, (*[]uint16)(unsafe.Pointer(&leaf_num)))
	}

	return root
}
