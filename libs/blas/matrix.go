package blas

import "unsafe"

/*
	extern void *alloc_matrix(unsigned int rows, unsigned int cols);
	extern void set_matrix_val(void *m, unsigned int rows, unsigned int cols, double val);
	extern double get_matrix_val(void *m, unsigned int rows, unsigned int cols);
	extern void free_matrix(void *m);
*/
import "C"

type Matrix struct {
	rows  uint64
	cols  uint64
	c_ptr unsafe.Pointer
}

func AllocMatrix(rows, cols uint) unsafe.Pointer {
	crows := C.uint(rows)
	ccols := C.uint(cols)
	mat := C.alloc_matrix(crows, ccols)
	return mat
}

func FreeMatrix(m unsafe.Pointer) {
	C.free_matrix(m)
}

func SetMatrixVal(mat unsafe.Pointer, rows, cols uint, val float64) {
	crows := C.uint(rows)
	ccols := C.uint(cols)
	cval := C.double(val)
	C.set_matrix_val(mat, crows, ccols, cval)
}

func GetMatrixVal(mat unsafe.Pointer, rows, cols uint) float64 {
	crows := C.uint(rows)
	ccols := C.uint(cols)
	cval := C.get_matrix_val(mat, crows, ccols)
	return float64(cval)
}
