// Copyright 1988 John Gilbert and Tim Peierls
// All rights reserved.

package gp

import "fmt"

// ludfs :  Depth-first search to allocate storage for U
//
// Input parameters:
//   jcol             current column number.
//   a, arow, acolst  the matrix A; see lufact for format.
//   rperm            row permutation P.
//                    perm(r) = s > 0 means row r of A is row s < jcol of PA.
//                    perm(r) = 0 means row r of A has not yet been used as a
//                    pivot and is therefore still below the diagonal.
//   cperm            column permutation.
//
// Modified parameters (see below for exit values):
//   lastlu           last used position in lurow array.
//   lurow, lcolst, ucolst  nonzero structure of Pt(L-I+U);
//                          see lufact for format.
//   dense            current column as a dense vector.
//   found            integer array for marking nonzeros in this column of
//                    Pt(L-I+U) that have been allocated space in lurow.
//                    Also, marks reached columns in depth-first search.
//                    found(i)=jcol if i was found in this column.
//   parent           parent(i) is the parent of vertex i in the dfs,
//                    or 0 if i is a root of the search.
//   child            child(i) is the index in lurow of the next unexplored
//                    child of vertex i.
//                    Note that parent and child are also indexed according to
//                    the vertex numbering of A, not PA; thus child(i) is
//                    the position of a nonzero in column rperm(i),
//                    not column i.
//
// Output parameters:
//   error            0 if successful, 1 otherwise
//
// On entry:
//   found(*)<jcol
//   dense(*)=0.0
//   ucolst(jcol)=lastlu+1 is the first free index in lurow.
//
// On exit:
//   found(i)=jcol iff i is a nonzero of column jcol of PtU or
//     a non-fill nonzero of column jcol of PtL.
//   dense(*)=column jcol of A.
//     Note that found and dense are kept according to the row
//     numbering of A, not PA.
//   lurow has the rows of the above-diagonal nonzeros of col jcol of U in
//     reverse topological order, followed by the non-fill nonzeros of col
//     jcol of L and the diagonal elt of U, in no particular order.
//     These rows also are numbered according to A, not PA.
//   lcolst(jcol) is the index of the first nonzero in col j of L.
//   lastlu is the index of the last non-fill nonzero in col j of L.
func ludfs(jcol int, a []float64, arow, acolst []int, lastlu *int, lurow, lcolst, ucolst, rperm, cperm []int, dense []float64, found, parent, child []int) error {
	// Local variables:
	//   nzast, nzaend   range of indices in arow for column jcol of A.
	//   nzaptr          pointer to current position in arow.
	//   krow            current vertex in depth-first search (numbered
	//                   according to A, not PA).
	//   nextk           possible next vertex in depth-first search.
	//   chdend          next index after last child of current vertex.
	//   chdptr          index of current child of current vertex
	var nextk, chdend, chdptr int

	// Depth-first search through columns of L from each nonzero of
	// column jcol of A that is above the diagonal in PA.

	// For each krow such that A(krow,jcol) is nonzero do...

	nzast := acolst[cperm[jcol-off]-off]
	nzaend := acolst[cperm[jcol-off]+1-off]

	if nzaend < nzast {
		return fmt.Errorf("ludfs, negative length for column %v of A. nzast=%v nzend=%v", jcol, nzast, nzaend)
	}
	nzaend = nzaend - 1
	for nzaptr := nzast; nzaptr <= nzaend; nzaptr++ {
		krow := arow[nzaptr-off]

		// Copy A(krow,jcol) into the dense vector. If above diagonal in
		// PA, start a depth-first search in column rperm(krow) of L.

		dense[krow-off] = a[nzaptr-off]
		if rperm[krow-off] == 0 {
			goto l500
		}
		if found[krow-off] == jcol {
			goto l500
		}
		if dense[krow-off] == 0.0 {
			goto l500
		}
		parent[krow-off] = 0
		found[krow-off] = jcol
		chdptr = lcolst[rperm[krow-off]-off]

		// The main depth-first search loop starts here.
		// repeat
		//   if krow has a child that is not yet found
		//   then step forward
		//   else step back
		// until a step back leads to 0
	l100:
		// Look for an unfound child of krow.
		chdend = ucolst[rperm[krow-off]+1-off]

	l200:
		if chdptr >= chdend {
			goto l400
		}
		nextk = lurow[chdptr-off]
		chdptr = chdptr + 1
		if rperm[nextk-off] == 0 {
			goto l200
		}
		if found[nextk-off] == jcol {
			goto l200
		}

		// Take a step forward.

		//l300:
		child[krow-off] = chdptr
		parent[nextk-off] = krow
		krow = nextk
		found[krow-off] = jcol
		chdptr = lcolst[rperm[krow-off]-off]
		goto l100

		// Take a step back.

		// Allocate space for U(rperm(k),jcol) = PtU(krow,jcol) in the sparse data structure.

	l400:
		*lastlu = *lastlu + 1
		lurow[*lastlu-off] = krow
		krow = parent[krow-off]
		if krow == 0 {
			goto l500
		}
		chdptr = child[krow-off]
		goto l100

		// The main depth-first search loop ends here.
	l500:
		continue
	}
	// Close off column jcol of U and allocate space for the non-fill
	// entries of column jcol of L.
	// The diagonal element goes in L, not U, until we do the column
	// division at the end of the major step.

	lcolst[jcol-off] = *lastlu + 1
	for nzaptr := nzast; nzaptr <= nzaend; nzaptr++ {
		krow := arow[nzaptr-off]
		if rperm[krow-off] != 0 {
			continue
		}
		found[krow-off] = jcol
		*lastlu = *lastlu + 1
		lurow[*lastlu-off] = krow
	}

	return nil
}
