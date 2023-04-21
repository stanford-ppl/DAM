package blas

import "unsafe"

/*
	extern void *alloc_matrix(unsigned long rows, unsigned long cols);
	extern void set_matrix_val(void *m, unsigned long rows, unsigned long cols, double val);
	extern double get_matrix_val(void *m, unsigned long rows, unsigned long cols);
	extern void free_matrix(void *m);
*/
import "C"

// Corresponds to MatrixXd in Eigen
type Matrix struct {
	rows  C.ulong
	cols  C.ulong
	c_ptr unsafe.Pointer
}

func AllocMatrix(rows, cols uint) Matrix {
	crows := C.ulong(rows)
	ccols := C.ulong(cols)
	ptr := C.alloc_matrix(crows, ccols)
	mat := Matrix{crows, ccols, ptr}
	return mat
}

func (m *Matrix) Free() {
	C.free_matrix(m.c_ptr)
}

func (m *Matrix) Set(rows, cols uint, val float64) {
	crows := C.ulong(rows)
	ccols := C.ulong(cols)
	cval := C.double(val)
	C.set_matrix_val(m.c_ptr, crows, ccols, cval)
}

func (m *Matrix) Get(rows, cols uint) float64 {
	crows := C.ulong(rows)
	ccols := C.ulong(cols)
	cval := C.get_matrix_val(m.c_ptr, crows, ccols)
	return float64(cval)
}
