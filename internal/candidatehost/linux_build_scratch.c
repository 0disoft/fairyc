//go:build ignore

/* Appended after rooted Linux I/O. Only the new private tree is cleanup-owned. */
#include <dirent.h>
#include <stdlib.h>
struct fy_linux_build_scratch { int fd; dev_t device; ino_t identity; char path[4096]; };
static uint32_t fy_linux_build_clear(int fd,unsigned depth,size_t *work) {
    if(depth>16) { return EOVERFLOW; }
    int copy=openat(fd,".",O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);
    if(copy<0) { return (uint32_t)errno; }
    DIR *stream=fdopendir(copy);if(stream==NULL){uint32_t code=(uint32_t)errno;(void)close(copy);return code;}
    uint32_t code=0;struct dirent *entry;
    while(1) {
        errno=0;entry=readdir(stream);if(entry==NULL){code=(uint32_t)errno;break;}
        if(strcmp(entry->d_name,".")==0 || strcmp(entry->d_name,"..")==0){continue;}
        if(++*work>16384){code=EOVERFLOW;break;}
        struct stat info;if(fstatat(fd,entry->d_name,&info,AT_SYMLINK_NOFOLLOW)<0){code=(uint32_t)errno;break;}
        if(S_ISDIR(info.st_mode)) {
            int child=openat(fd,entry->d_name,O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);
            if(child<0){code=(uint32_t)errno;break;}
            code=fy_linux_build_clear(child,depth+1,work);(void)close(child);
            if(code!=0){break;}
            if(unlinkat(fd,entry->d_name,AT_REMOVEDIR)<0){code=(uint32_t)errno;break;}
        } else if(unlinkat(fd,entry->d_name,0)<0){code=(uint32_t)errno;break;}
    }
    if(closedir(stream)<0 && code==0){code=(uint32_t)errno;}return code;
}
uint32_t fairyc_private_linux_build_scratch_drop_v1(size_t raw) {
    if(raw==0){return 0;}struct fy_linux_build_scratch *scratch=(struct fy_linux_build_scratch *)(uintptr_t)raw;size_t work=0;
    uint32_t code=fy_linux_build_clear(scratch->fd,0,&work);
    if(code==0){struct stat info;if(lstat(scratch->path,&info)<0){code=(uint32_t)errno;}else if(!S_ISDIR(info.st_mode)||info.st_dev!=scratch->device||info.st_ino!=scratch->identity){code=ESTALE;}else if(rmdir(scratch->path)<0){code=(uint32_t)errno;}}
    if(close(scratch->fd)<0 && code==0){code=(uint32_t)errno;}free(scratch);return code;
}
uint32_t fairyc_private_linux_build_scratch_open_v1(const uint8_t *root,size_t size,uint8_t *path,size_t capacity,size_t *length,size_t *output) {
    if(output==NULL || length==NULL){return EINVAL;}*output=0;*length=0;
    if(!fy_linux_root_path(root,size,0) || path==NULL || capacity<4096){return EINVAL;}
    char *absolute=realpath((const char *)root,NULL);if(absolute==NULL){return (uint32_t)errno;}
    size_t count=strlen(absolute);if(count>4000){free(absolute);return ENAMETOOLONG;}
    struct fy_linux_build_scratch *scratch=calloc(1,sizeof *scratch);if(scratch==NULL){free(absolute);return ENOMEM;}
    int written=snprintf(scratch->path,sizeof scratch->path,"%s/fairyc-build-XXXXXX",absolute);free(absolute);
    if(written<0 || (size_t)written>=sizeof scratch->path){free(scratch);return EINVAL;}
    if(mkdtemp(scratch->path)==NULL){uint32_t code=(uint32_t)errno;free(scratch);return code;}
    scratch->fd=open(scratch->path,O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);
    if(scratch->fd<0){uint32_t code=(uint32_t)errno;(void)rmdir(scratch->path);free(scratch);return code;}
    struct stat info;if(fstat(scratch->fd,&info)<0){uint32_t code=(uint32_t)errno;(void)close(scratch->fd);(void)rmdir(scratch->path);free(scratch);return code;}
    scratch->device=info.st_dev;scratch->identity=info.st_ino;
    *length=strlen(scratch->path);memcpy(path,scratch->path,*length+1);*output=(size_t)(uintptr_t)scratch;return 0;
}
