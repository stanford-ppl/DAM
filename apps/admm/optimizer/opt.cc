#include <iostream>
#include <Eigen/Dense>

using Eigen::MatrixXd;

extern "C" int demo()
{
    std::cout << "C++ function called" << std::endl;
    MatrixXd m(2, 2);
    m(0, 0) = 3;
    m(1, 0) = 2.5;
    m(0, 1) = 42;
    m(1, 1) = m(0, 0) + m(1, 0);
    std::cout << m << std::endl;
    std::cout << "Exiting C++ function" << std::endl;
    return 1;
}
