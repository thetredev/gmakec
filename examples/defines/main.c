#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main() {
#ifndef DEFINE_WITHOUT_VALUE
    fprintf(stderr, "DEFINE_WITHOUT_VALUE is not defined!\n");
    abort();
#endif // DEFINE_WITHOUT_VALUE

#ifndef DEFINE_WITH_INT_VALUE
    fprintf(stderr, "DEFINE_WITH_INT_VALUE is not defined!\n");
    abort();
#else
    assert(DEFINE_WITH_INT_VALUE == 69);
#endif // DEFINE_WITH_INT_VALUE

#ifndef DEFINE_WITH_STRING_VALUE
    fprintf(stderr, "DEFINE_WITH_STRING_VALUE is not defined!\n");
    abort();
#else
    assert(strcmp(DEFINE_WITH_STRING_VALUE, "hello") == 0);
#endif // DEFINE_WITH_STRING_VALUE

#ifndef DEFINE_WITH_STRING_NUMBER_VALUE
    fprintf(stderr, "DEFINE_WITH_STRING_NUMBER_VALUE is not defined!\n");
    abort();
#else
    assert(strcmp(DEFINE_WITH_STRING_NUMBER_VALUE, "420") == 0);
#endif // DEFINE_WITH_STRING_NUMBER_VALUE
    return 0;
}
