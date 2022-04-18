#define _GNU_SOURCE

#include <stdio.h>

#include <sched.h>
// #include <linux/fcntl.h>      /* Definition of AT_* constants */
#include <sys/syscall.h>      /* Definition of SYS_* constants */
       
#include <fcntl.h>

#include <unistd.h>
#include <linux/limits.h>
#include <stdlib.h>

void _gdb_expr () {
    printf("Hello world\n");
    // run with
    // gdb -q -ex "set pagination off" -ex "attach 1" -ex "compile file -raw hook.c" -ex "detach" -ex "q"
}

/*

This program works similar to nsenter, but it uses "execveat" so that we can execute the program in
the host namespace. This allows to execute a static binary in the mount namespace of a different process.
The use case in mind is static-bash and distroless containers.
*/


int main(int argc, char *argv[]) {
    char* bin = "/bin/bash";
    pid_t pid = 1;
    if (argc == 2) {
        bin = argv[1];
    } else if (argc > 2) {
        pid = atoi(argv[1]);
        bin = argv[2];
    }

#if DEBUG
    printf("getting pid %u\n", pid);
#endif

    // open the file descriptor for the binary. we want it as an FD, so we can use it
    // after we chroot.
    int bash_fd = open(bin, O_PATH | O_CLOEXEC);
    if (bash_fd < 0){
        perror("open");
        return 1;
    }

    // Get the pid that we want to enter
    int procfd = syscall(SYS_pidfd_open, pid, 0);
    if (procfd < 0){
        perror("pidfd_open");
        return 1;
    }

    // Get the root path, so we can chroot to it
    char pathbuf[PATH_MAX];
    snprintf(pathbuf, sizeof(pathbuf), "/proc/%u/root", pid);
    int root_fd = open(pathbuf, O_RDONLY | O_CLOEXEC);
    if (root_fd < 0){
        perror("open");
        return 1;
    }

    // move us to the mount namespace of the target process
    if (setns(procfd, CLONE_NEWNET) != 0) {
        perror("setns");
        return 1;
    }
    close(procfd);

    // chroot to the root path
    if (fchdir(root_fd) < 0) {
        perror("fchdir");
        return 1;
    }
    if (chroot(".") < 0) {
        perror("chroot");
        return 1;
    }
    if (chdir("/") < 0) {
        perror("chdir");
        return 1;
    }

    // use execveat to execute the binary. This still works even though the binary doesn't exist
    // in our file system, because we already have an fd open to it.
    char *const argv2[] = {bin, NULL};
    int ret = execveat(bash_fd, "", argv2, NULL, AT_EMPTY_PATH);
    if (ret != 0) {
        perror("execveat");
        return 1;
    }
}