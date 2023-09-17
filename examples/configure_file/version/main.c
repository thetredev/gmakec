#include <stdio.h>
#include <stdlib.h>

#include "version.h"

int main() {
#ifndef MY_VERSION
    fprintf(stderr, "MY_VERSION is not defined!\n");
    abort();
#else
    printf("version: %s\n", MY_VERSION);
#endif // MY_VERSION

#ifndef MY_VERSION_MAJOR
    fprintf(stderr, "MY_VERSION_MAJOR is not defined!\n");
    abort();
#else
    printf("major: %d\n", MY_VERSION_MAJOR);
#endif // MY_VERSION_MAJOR

#ifndef MY_VERSION_MINOR
    fprintf(stderr, "MY_VERSION_MINOR is not defined!\n");
    abort();
#else
    printf("minor: %d\n", MY_VERSION_MINOR);
#endif // MY_VERSION_MINOR

#ifndef MY_VERSION_PATCH
    fprintf(stderr, "MY_VERSION_PATCH is not defined!\n");
    abort();
#else
    printf("patch: %d\n", MY_VERSION_PATCH);
#endif // MY_VERSION_PATCH

#ifndef MY_VERSION_TWEAK
    fprintf(stderr, "MY_VERSION_TWEAK is not defined!\n");
    // no abort because not mandatory?
#endif // MY_VERSION_TWEAK

    return 0;
}
