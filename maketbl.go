package lha

import "fmt"

func makeTable(nchar int16, bitlen *[]byte, tablebits int16, table *[]uint16) error {
	var (
		count             [17]uint16 /* count of bitlen */
		weight            [17]uint16 /* 0x10000ul >> bitlen */
		start             [17]uint16 /* first code of bitlen */
		total             uint16
		i, l              uint
		j, k, m, n, avail int
		p                 int
	)

	avail = int(nchar)

	/* initialize */
	for i = 1; i <= 16; i++ {
		count[i] = 0
		weight[i] = 1 << (16 - i)
	}

	/* count */
	for i = 0; i < uint(nchar); i++ {
		if (*bitlen)[i] > 16 {
			/* CVE-2006-4335 */
			return fmt.Errorf("Bad table (case a)")

		} else {
			count[(*bitlen)[i]]++
		}
	}

	/* calculate first code */
	total = 0
	for i = 1; i <= 16; i++ {
		start[i] = total
		total += weight[i] * count[i]
	}
	if (total&0xffff) != 0 || tablebits > 16 { /* 16 for weight below */
		return fmt.Errorf("make_table(): Bad table (case b)")
	}

	/* shift data for make table. */
	m = 16 - int(tablebits)
	for i = 1; i <= uint(tablebits); i++ {
		start[i] >>= m
		weight[i] >>= m
	}

	/* initialize */
	j = int(start[tablebits+1]) >> m
	k = min(1<<tablebits, 4096)
	if j != 0 {
		for i = uint(j); i < uint(k); i++ {
			(*table)[i] = 0
		}
	}
	/* create table and tree */
	for j = 0; j < int(nchar); j++ {
		k = int((*bitlen)[j])
		if k == 0 {
			continue
		}
		l = uint(start[k]) + uint(weight[k])
		if k <= int(tablebits) {
			/* code in table */
			l = uint(min(int(l), 4096))
			for i = uint(start[k]); i < l; i++ {
				(*table)[i] = uint16(j)
			}
		} else {
			/* code not in table */
			i = uint(start[k])
			if (i >> m) > 4096 {
				/* CVE-2006-4337 */
				return fmt.Errorf("Bad table (case c)")
			}
			p = int(i) >> m
			i <<= tablebits
			n = k - int(tablebits)
			/* make tree (n length) */
			n--
			var setRight bool
			for n >= 0 {
				if (*table)[p] == 0 {
					left[avail] = 0
					right[avail] = 0
					(*table)[p] = uint16(avail)
					avail++
				}
				p = int((*table)[p])
				if i&0x8000 != 0 {
					setRight = true
				} else {
					setRight = false
				}
				i <<= 1
				n--
			}
			if setRight {
				right[p] = uint16(j)
			} else {
				left[p] = uint16(j)
			}
		}
		start[k] = uint16(l)
	}
	return nil
}
