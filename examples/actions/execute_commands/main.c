#include <stdio.h>
#include <stdlib.h>

int main() {
#ifndef SYSTEM_PYTHON_VERSION
    fprintf(stderr, "SYSTEM_PYTHON_VERSION is not defined!\n");
#else
    printf("SYSTEM_PYTHON_VERSION is: %s\n", SYSTEM_PYTHON_VERSION);
#endif // SYSTEM_PYTHON_VERSION

#ifndef PRE_CONFIGURE_FAILED
    fprintf(stderr, "PRE_CONFIGURE_FAILED is not defined!\n");
    abort();
#else
    printf("PRE_CONFIGURE_FAILED is: %d\n", PRE_CONFIGURE_FAILED);
#endif // SYSTEM_PYTHON_VERSION
    return 0;
}
