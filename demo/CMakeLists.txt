cmake_minimum_required(VERSION 3.15)
project(demo)

set(CMAKE_C_STANDARD 11)

add_executable(demo benchmark.c fft.c complex.c)
target_link_libraries(demo m)

