#include <stdio.h>

void _gdb_expr () {
    printf("Hello world\n");
}

// run with
// gdb -q -ex "set pagination off" -ex "attach 1" -ex "compile file -raw hook.c" -ex "detach" -ex "q"
