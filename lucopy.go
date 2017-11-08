package lufact

import (
	"fmt"
	"math"
)

// lucopy copies dense column to sparse structure, pivot, and divide.
//
// Copy column jcol from the dense vector to the sparse data structure,
// zeroing out the dense vector.  Then find the diagonal element (either
// by partial or threshold pivoting or by looking for row number=col
// number), move it from L into U, and divide the column of L by it.
//
// Input variables:
//   pivot   = -1 for columns with no diagonal element
//           = 0 for no pivoting
//           = 1 for partial (row) pivoting
//           = 2 for threshold (row) pivoting
//   pthresh  fraction of max pivot candidate acceptable for pivoting
//   jcol    Current column number.
//   ncol    Total number of columns; upper bound on row counts.
//
// Modified variables:
//   lastlu                 Index of last nonzero in lu, updated here.
//   lu                     On entry, cols 1 through jcol-1 of Pt(L-I+U).
//                          On exit, cols 1 through jcol.
//   lurow, lcolst, ucolst  Nonzero structure of columns 1 through jcol
//                          of PtL and PtU.
//                          No pivoting has been done, so on entry the
//                          element that will be U(jcol,jcol) is still
//                          somewhere in L; on exit, it is the last
//                          nonzero in column jcol of U.
//   rperm                  The row permutation P.
//                          rperm(r) = s > 0 means row r of A is row s of PA.
//                          rperm(r) = 0 means row r of A has not yet been used
//                          as a pivot.  On input, perm reflects rows 1 through
//                          jcol-1 of PA; on output, rows 1 through jcol.
//   cperm                  The column permutation.
//   dense                  On entry, column jcol of Pt(U(jcol,jcol)*(L-I)+U).
//                          On exit, zero.
//   flops                  flop count
//
// Output variable:
//   zpivot                 > 0 for success (pivot row), -1 for zero pivot element.
func lucopy(pivot int, pthresh, dthresh float64, nzcount int,
	jcol, ncol int, lastlu *int, lu []float64, lurow, lcolst, ucolst []int,
	rperm, cperm []int, dense []float64, pattern []int, twork []float64) (int, error) {
	// Local variables:
	//   nzptr       Index into lurow of current nonzero.
	//   nzst, nzend Loop bounds for nzptr.
	//   irow        Row number of current nonzero (according to A, not PA).
	//   pivrow      Pivot row number (according to A, not PA).
	//   maxpiv      Temporary to find maximum element in column for pivoting.
	//   utemp       Temporary for computing maxpiv.
	//   ujj         Diagonal element U(jcol,jcol) = PtU(pivrow,jcol).
	//   ujjptr      Index into lu and lurow of diagonal element.
	//   dptr        Temporary index into lu and lurow.
	//   diagptr     Index to diagonal element of QAQt
	//   diagpiv     Value of diagonal element

	// Copy column jcol from dense to sparse, recording the position of
	// the diagonal element.
	ujjptr := 0

	if pivot <= 0 {
		// No pivoting, diagonal element has irow = jcol.
		// Copy the column elements of U and L, throwing out zeros.

		if ucolst[jcol+1]-1 < ucolst[jcol] {
			//zpivot = -1
			return -1, fmt.Errorf("zero length (U-I+L) column")
		}

		// Start with U.
		nzcpy := ucolst[jcol]
		for nzptr := ucolst[jcol]; nzptr <= lcolst[jcol]-1; nzptr++ {
			irow := lurow[nzptr]

			if pattern[irow] != 0 || irow == cperm[jcol] {
				lurow[nzcpy] = irow
				lu[nzcpy] = dense[irow]
				dense[irow] = 0.0
				nzcpy = nzcpy + 1
			} else {
				dense[irow] = 0.0
			}
		}
		lastu := nzcpy - 1

		//    Now do L. Same action as U, except that we search for diagonal.
		for nzptr := lcolst[jcol]; nzptr <= ucolst[jcol+1]-1; nzptr++ {
			irow := lurow[nzptr]
			//if irow == cperm[jcol] {
			if pattern[irow] == 2 {
				ujjptr = nzcpy
			}
			if pattern[irow] != 0 || /*irow == cperm(jcol)) then*/ pattern[irow] == 2 {
				lurow[nzcpy] = irow
				lu[nzcpy] = dense[irow]
				dense[irow] = 0.0
				nzcpy = nzcpy + 1
			} else {
				dense[irow] = 0.0
			}
		}

		lcolst[jcol] = lastu + 1
		ucolst[jcol+1] = nzcpy
		*lastlu = nzcpy - 1

		if pivot == -1 {
			zpivot := 0 //pivrow
			return zpivot, nil
		}

	} else {
		var udthreshabs, ldthreshabs float64

		// Partial and threshold pivoting.
		if ucolst[jcol+1]-1 < lcolst[jcol] {
			//zpivot = -1
			return -1, fmt.Errorf("zero length L column")
		}

		// Partial pivoting, diagonal elt. has max. magnitude in L.
		// Compute the drop threshold for the column
		if nzcount <= 0 {
			maxpivglb := -1.0
			for nzptr := ucolst[jcol]; nzptr <= lcolst[jcol]-1; nzptr++ {
				irow := lurow[nzptr]
				utemp := math.Abs(dense[irow])
				if utemp > maxpivglb {
					maxpivglb = utemp
				}
			}
			udthreshabs = dthresh * maxpivglb

			maxpivglb = -1.0
			for nzptr := lcolst[jcol]; nzptr <= ucolst[jcol+1]-1; nzptr++ {
				irow := lurow[nzptr]
				utemp := math.Abs(dense[irow])
				if utemp > maxpivglb {
					maxpivglb = utemp
				}
			}
			ldthreshabs = dthresh * maxpivglb
		} else {
			i := 0
			for nzptr := ucolst[jcol]; nzptr <= lcolst[jcol]-1; nzptr++ {
				i = i + 1
				irow := lurow[nzptr]
				utemp := math.Abs(dense[irow])
				twork[i] = utemp
			}
			if nzcount < i {
				var kth float64
				dordstat(i, i-nzcount+1, twork, &kth, &i)
				udthreshabs = kth
			} else {
				udthreshabs = 0.0
			}

			i = 0
			for nzptr := lcolst[jcol]; nzptr <= ucolst[jcol+1]-1; nzptr++ {
				i = i + 1
				irow := lurow[nzptr]
				utemp := math.Abs(dense[irow])
				twork[i] = utemp
			}
			if nzcount < i {
				var kth float64
				dordstat(i, i-nzcount+1, twork, &kth, &i)
				ldthreshabs = kth
			} else {
				ldthreshabs = 0.0
			}

		}

		// Copy the column elements of U, throwing out zeros.
		nzcpy := ucolst[jcol]
		if lcolst[jcol]-1 >= ucolst[jcol] {
			for nzptr := ucolst[jcol]; nzptr <= lcolst[jcol]-1; nzptr++ {
				irow := lurow[nzptr]

				//if (pattern(irow) .ne. 0 .or. pattern(irow) .eq. 2) then
				if pattern[irow] != 0 || math.Abs(dense[irow]) >= udthreshabs {
					lurow[nzcpy] = irow
					lu[nzcpy] = dense[irow]
					dense[irow] = 0.0
					nzcpy = nzcpy + 1
				} else {
					dense[irow] = 0.0
				}
			}
		}
		lastu := nzcpy - 1

		// Copy the column elements of L, throwing out zeros.
		// Keep track of maximum magnitude element for pivot.

		if ucolst[jcol+1]-1 < lcolst[jcol] {
			//zpivot = -1
			return -1, fmt.Errorf("zero length L column")
		}

		// Partial pivoting, diagonal elt. has max. magnitude in L.
		diagptr := 0
		diagpiv := 0.0

		ujjptr = 0
		maxpiv := -1.0
		maxpivglb := -1.0

		for nzptr := lcolst[jcol]; nzptr <= ucolst[jcol+1]-1; nzptr++ {
			irow := lurow[nzptr]
			utemp := math.Abs(dense[irow])

			//if irow == cperm[jcol] {
			if pattern[irow] == 2 {
				diagptr = irow
				diagpiv = utemp
				//if diagpiv == 0 { print*, 'WARNING: Numerically zero diagonal element at col', jcol }
			}

			// original
			//if utemp > maxpiv {

			// do not pivot outside the pattern
			// if utemp > maxpiv && pattern[irow] != 0 {

			// Pivot outside pattern.
			if utemp > maxpiv {
				ujjptr = irow
				maxpiv = utemp
			}

			// Global pivot outside pattern.
			if utemp > maxpivglb {
				maxpivglb = utemp
			}
		}

		// Threshold pivoting.
		if diagptr != 0 && diagpiv >= (pthresh*maxpiv) {
			ujjptr = diagptr
		}

		if diagptr == 0 && ujjptr == 0 {
			fmt.Printf("error: %v", ucolst[jcol+1]-lcolst[jcol])
		}

		//if diagptr != ujjptr {
		//	print("pivoting", pthresh, maxpiv, diagpiv, diagptr)
		//}

		diagptr = ujjptr
		ujjptr = 0

		for nzptr := lcolst[jcol]; nzptr <= ucolst[jcol+1]-1; nzptr++ {
			irow := lurow[nzptr]
			utemp := math.Abs(dense[irow])

			//if irow == cperm[jcol] {
			//   diagptr = nzcpy
			//   diagpiv = utemp
			//}

			//if utemp > maxpiv {
			//   ujjptr = nzcpy
			//   maxpiv = utemp
			//}

			// Pattern dropping.
			// if pattern[irow] == 0 && irow != diagptr {

			// Pattern + threshold dropping.

			if pattern[irow] == 0 && irow != diagptr && utemp < ldthreshabs {
				dense[irow] = 0.0
			} else {
				if irow == diagptr {
					ujjptr = nzcpy
				}

				lurow[nzcpy] = irow
				lu[nzcpy] = dense[irow]
				dense[irow] = 0.0
				nzcpy = nzcpy + 1
			}
		}

		lcolst[jcol] = lastu + 1
		ucolst[jcol+1] = nzcpy
		*lastlu = nzcpy - 1

	}

	// Diagonal element has been found. Swap U(jcol,jcol) from L into U.

	if ujjptr == 0 {
		return -1, fmt.Errorf("ujjptr not set (1)" /*diagptr*/, ujjptr, lcolst[jcol], ucolst[jcol+1]-1)
	}

	pivrow := lurow[ujjptr]
	ujj := lu[ujjptr]

	if ujj == 0.0 {
		return -1, fmt.Errorf("numerically zero diagonal element at column %v", jcol)
	}
	dptr := lcolst[jcol]
	lurow[ujjptr] = lurow[dptr]
	lu[ujjptr] = lu[dptr]
	lurow[dptr] = pivrow
	lu[dptr] = ujj
	lcolst[jcol] = dptr + 1

	// Record the pivot in P.

	rperm[pivrow] = jcol
	//if pivrow == 38 {
	//	print("exchanging", jcol, pivrow)
	//}

	// Divide column jcol of L by U(jcol,jcol).

	nzst := lcolst[jcol]
	nzend := ucolst[jcol+1] - 1
	if nzst > nzend {
		zpivot := pivrow
		return zpivot, nil
	}
	for nzptr := nzst; nzptr <= nzend; nzptr++ {
		lu[nzptr] = lu[nzptr] / ujj
	}

	zpivot := pivrow
	return zpivot, nil
}
