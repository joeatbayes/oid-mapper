#include <stdio.h>
#include <stdlib.h>
#include <string.h>
// To Build
//   gcc -o exReadln exReadln.c

/*
   read only step reads 531MiB/s consistently from SATA/SSD consuming 30% of one core.
   Took 2m24 seconds.  
   
   Read against the NVME drive in same NUC.  1.8 Gib/s.  Took 0m46s consumed 98% of 1 core.
   
*/
int main(void) {
	//  ../data/stage/generated_oids.340m.map.txt"
    FILE *fp = fopen("/home/jwork/index/340wrk-.merge.s-256.srt", "r");
    if(fp == NULL) {
        perror("Unable to open file!");
        exit(1);
    }

    // Read lines using POSIX function getline
    char *line = NULL;
    size_t len = 0;
    int cnt = 0;
    int bytesRead = 0;
    while(getline(&line, &len, fp) != -1) {
        // fputs(line, stdout);
        // fputs("|*\n", stdout);
        cnt++;
        bytesRead += len;
    }
    printf("\n\nRead %zd lines bytes Read %zd\n", cnt, bytesRead);
    fclose(fp);
    free(line);     // getline will resize the input buffer as necessary
                    // the user needs to free the memory when not needed!
}