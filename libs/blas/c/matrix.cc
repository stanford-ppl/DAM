#include <Eigen/Dense>

using Eigen::MatrixXd;

extern "C" void *alloc_matrix(unsigned int rows, unsigned int cols)
{
    MatrixXd *m = new MatrixXd(rows, cols);
    return (void *)m;
}

extern "C" void free_matrix(void *m)
{
    MatrixXd *mat = (MatrixXd *)m;
    delete mat;
}

extern "C" void set_matrix_val(void *m, unsigned int rows, unsigned int cols, double val)
{
    MatrixXd *mat = (MatrixXd *)m;
    (*mat)(rows, cols) = val;
}

extern "C" double get_matrix_val(void *m, unsigned int rows, unsigned int cols)
{
    MatrixXd *mat = (MatrixXd *)m;
    return (*mat)(rows, cols);
}
