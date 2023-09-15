// "inspired" by https://www.geeksforgeeks.org/multithreading-in-c/

#include <stdio.h>
#include <unistd.h>
#include <pthread.h>

#include "mylib.h"

void *myThreadFun(void *vargp)
{
    (void) vargp;
    sleep(1);

    printf("Printing GeeksQuiz from Thread \n");
    return NULL;
}

int myfunc() {
    pthread_t thread_id;

    printf("Before thread\n");
    pthread_create(&thread_id, NULL, myThreadFun, NULL);
    pthread_join(thread_id, NULL);
    printf("After thread\n");

    return 0;
}
