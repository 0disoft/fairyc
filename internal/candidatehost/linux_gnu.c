//go:build ignore

/* Private OS marshalling only. Appended after rooted/process/capture adapters.
   Fairy selects sources, compiler flags, invocation order and artifact admission. */
#include <dirent.h>
#include <sys/random.h>

uint32_t fairyc_private_linux_gnu_stage_v1(const uint8_t *path,size_t size,uint32_t *output) {
    if(output==NULL){return EINVAL;}*output=0;
    if(!fy_linux_root_path(path,size,0) || path[0]!='/'){return EINVAL;}
    uint32_t code=fairyc_private_linux_directory_open_v1(path,size,output);
    if(code!=0){return code;}
    int fd=fy_linux_root_fd(*output);struct stat info;
    if(fstat(fd,&info)!=0){code=(uint32_t)errno;}
    else if((info.st_mode&0777)!=0700){code=EACCES;}
    int copy=code==0?openat(fd,".",O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC):-1;
    if(code==0 && copy<0){code=(uint32_t)errno;}
    DIR *dir=copy>=0?fdopendir(copy):NULL;
    if(copy>=0 && dir==NULL){code=(uint32_t)errno;(void)close(copy);}
    if(dir!=NULL){
        for(;;){errno=0;struct dirent *entry=readdir(dir);if(entry==NULL){if(errno!=0){code=(uint32_t)errno;}break;}
            if(strcmp(entry->d_name,".")==0 || strcmp(entry->d_name,"..")==0){continue;}
            if(strcmp(entry->d_name,"packet.bin")!=0){code=EINVAL;break;}
            if(fstatat(fd,"packet.bin",&info,AT_SYMLINK_NOFOLLOW)!=0){code=(uint32_t)errno;break;}
            if(!S_ISREG(info.st_mode)){code=EINVAL;break;}
        }
        if(closedir(dir)!=0 && code==0){code=(uint32_t)errno;}
    }
    if(code!=0){(void)fairyc_private_linux_root_file_close_v1(*output);*output=0;}return code;
}

static uint32_t fy_linux_gnu_write(int fd,const uint8_t *data,size_t size) {
    size_t at=0;while(at<size){ssize_t n=write(fd,data+at,size-at>65536?65536:size-at);
        if(n<0 && errno==EINTR){continue;}if(n<=0){return n<0?(uint32_t)errno:EIO;}at+=(size_t)n;
    }return 0;
}
uint32_t fairyc_private_linux_gnu_write_v1(uint32_t root,const uint8_t *leaf,size_t size,const uint8_t *data,size_t length) {
    int fd=fy_linux_root_fd(root);if(fd<0 || !fy_linux_root_path(leaf,size,1) || data==NULL || length==0 || length>8388608){return EINVAL;}
    int file=openat(fd,(const char *)leaf,O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW|O_CLOEXEC,0600);
    if(file<0){return (uint32_t)errno;}uint32_t code=fy_linux_gnu_write(file,data,length);
    if(close(file)!=0 && code==0){code=(uint32_t)errno;}return code;
}
/* A size query allocates nothing; the subsequent read independently rechecks size. */
uint32_t fairyc_private_linux_gnu_read_v1(uint32_t root,const uint8_t *leaf,size_t size,uint8_t *data,size_t capacity,size_t *length) {
    if(length==NULL){return EINVAL;}*length=0;int fd=fy_linux_root_fd(root);
    if(fd<0 || !fy_linux_root_path(leaf,size,1) || capacity>67108864 || (capacity!=0 && data==NULL)){return EINVAL;}
    int file=openat(fd,(const char *)leaf,O_RDONLY|O_NOFOLLOW|O_NONBLOCK|O_CLOEXEC);
    if(file<0){return (uint32_t)errno;}struct stat before,after;memset(&after,0,sizeof after);uint32_t code=0;
    if(fstat(file,&before)!=0){code=(uint32_t)errno;}
    else if(!S_ISREG(before.st_mode) || before.st_size<=0 || before.st_size>67108864){code=EINVAL;}
    else{*length=(size_t)before.st_size;}
    if(code==0 && capacity!=0){
        if(*length!=capacity){code=ESTALE;}else{
            size_t at=0;while(at<capacity){ssize_t n=read(file,data+at,capacity-at>65536?65536:capacity-at);
                if(n<0 && errno==EINTR){continue;}if(n<=0){code=n<0?(uint32_t)errno:EIO;break;}at+=(size_t)n;}
            if(code==0 && fstat(file,&after)!=0){code=(uint32_t)errno;}
            if(code==0 && (after.st_size!=before.st_size || after.st_mtim.tv_sec!=before.st_mtim.tv_sec || after.st_mtim.tv_nsec!=before.st_mtim.tv_nsec)){code=ESTALE;}
        }
    }
    if(close(file)!=0 && code==0){code=(uint32_t)errno;}if(code!=0){*length=0;}return code;
}

uint32_t fairyc_private_linux_gnu_find_v1(const uint8_t *name,size_t size,uint8_t *output,size_t capacity,size_t *length) {
    if(length==NULL){return EINVAL;}*length=0;
    if(!fy_linux_root_path(name,size,1) || output==NULL || capacity<4096 ||
       (strcmp((const char *)name,"gcc")!=0 && strcmp((const char *)name,"clang")!=0 && strcmp((const char *)name,"getconf")!=0)){return EINVAL;}
    const char *path=getenv("PATH");if(path==NULL){return ENOENT;}size_t total=strnlen(path,1048577);
    if(total>1048576){return EOVERFLOW;}size_t first=0;
    for(size_t i=0;i<=total;++i){if(i<total && path[i]!=':'){continue;}size_t width=i-first;
        if(width<=4000){char candidate[4096];int n=snprintf(candidate,sizeof candidate,"%.*s/%s",(int)(width==0?1:width),width==0?".":path+first,(const char *)name);
            if(n>0 && (size_t)n<sizeof candidate && access(candidate,X_OK)==0){
                char *resolved=realpath(candidate,NULL);if(resolved!=NULL){struct stat info;size_t count=strlen(resolved);
                    if(count<4096 && stat(resolved,&info)==0 && S_ISREG(info.st_mode)){memcpy(output,resolved,count+1);*length=count;free(resolved);return 0;}free(resolved);
                }
            }
        }first=i+1;
    }return ENOENT;
}

/* Child inherits the companion's outer-owned process group. It never creates
   a group which could escape the caller's deadline/descendant cleanup. */
uint32_t fairyc_private_linux_gnu_run_v1(const uint8_t *data,size_t size,size_t argc,const uint8_t *cwd,size_t cwd_size,uint32_t timeout_ms,
        uint8_t *out,size_t *out_length,uint8_t *err,size_t *err_length,uint32_t *kind,uint32_t *value) {
    if(out_length==NULL || err_length==NULL || kind==NULL || value==NULL){return EINVAL;}
    *out_length=0;*err_length=0;*kind=0;*value=0;
    if(data==NULL || out==NULL || err==NULL || argc==0 || argc>131071 || size>1048576-8*(argc+1) || argc>size ||
       !fy_linux_root_path(cwd,cwd_size,0) || cwd[0]!='/' || timeout_ms==0 || timeout_ms>600000){return EINVAL;}
    char **argv=calloc(argc+1,sizeof *argv);if(argv==NULL){return ENOMEM;}size_t at=0;
    for(size_t i=0;i<argc;++i){const uint8_t *end=memchr(data+at,0,size-at);if(end==NULL || (i==0 && (end==data || data[0]!='/'))){free(argv);return EINVAL;}argv[i]=(char *)(data+at);at=(size_t)(end-data)+1;}
    if(at!=size){free(argv);return EINVAL;}
    size_t env_count=0;while(environ[env_count]!=NULL){if(env_count==131071){free(argv);return EOVERFLOW;}++env_count;}
    char **env=calloc(env_count+4,sizeof *env);if(env==NULL){free(argv);return ENOMEM;}size_t used=0;
    for(size_t i=0;i<env_count;++i){if(strncmp(environ[i],"TMPDIR=",7)!=0 && strncmp(environ[i],"LC_ALL=",7)!=0 && strncmp(environ[i],"LANG=",5)!=0){env[used++]=environ[i];}}
    char temporary[4104];(void)snprintf(temporary,sizeof temporary,"TMPDIR=%s",(const char *)cwd);env[used++]=temporary;env[used++]="LC_ALL=C";env[used++]="LANG=C";
    int output[2]={-1,-1},error[2]={-1,-1},input[2]={-1,-1};uint64_t started=0;uint32_t code=fy_linux_process_now(&started);
    if(code==0){code=fy_linux_capture_pipe(output,3);}if(code==0){code=fy_linux_capture_pipe(error,3);}if(code==0){code=fy_linux_capture_pipe(input,3);}
    if(code==0){code=fy_linux_capture_close(&input[1]);}
    posix_spawn_file_actions_t actions;int initialized=0;pid_t pid=0;
    if(code==0){code=(uint32_t)posix_spawn_file_actions_init(&actions);initialized=code==0;}
    if(code==0){code=(uint32_t)posix_spawn_file_actions_addchdir_np(&actions,(const char *)cwd);}
    if(code==0){code=(uint32_t)posix_spawn_file_actions_adddup2(&actions,input[0],0);}
    if(code==0){code=(uint32_t)posix_spawn_file_actions_adddup2(&actions,output[1],1);}
    if(code==0){code=(uint32_t)posix_spawn_file_actions_adddup2(&actions,error[1],2);}
    if(code==0){code=(uint32_t)posix_spawn_file_actions_addclosefrom_np(&actions,3);}
    if(code==0){code=(uint32_t)posix_spawn(&pid,argv[0],&actions,NULL,argv,env);}
    if(initialized){(void)posix_spawn_file_actions_destroy(&actions);}free(argv);free(env);
    uint32_t closed=fy_linux_capture_close(&output[1]);if(code==0){code=closed;}
    closed=fy_linux_capture_close(&error[1]);if(code==0){code=closed;}
    closed=fy_linux_capture_close(&input[0]);if(code==0){code=closed;}(void)fy_linux_capture_close(&input[1]);
    int done=0,status=0;uint64_t deadline=started+(uint64_t)timeout_ms*UINT64_C(1000000);
    while(code==0){
        uint64_t now=0;code=fy_linux_process_now(&now);if(code!=0){break;}if(now<started){code=EIO;break;}int exceeded=0;
        code=fy_linux_capture_read(&output[0],out,65536,out_length,&exceeded);
        if(code==0 && !exceeded){code=fy_linux_capture_read(&error[0],err,65536,err_length,&exceeded);}
        if(code!=0){break;}if(exceeded || *out_length+*err_length>65536){*kind=4;break;}
        if(!done){siginfo_t info;memset(&info,0,sizeof info);if(waitid(P_PID,(id_t)pid,&info,WEXITED|WNOWAIT|WNOHANG)!=0){if(errno==EINTR && now<deadline){continue;}if(errno==EINTR){*kind=3;break;}code=(uint32_t)errno;break;}done=info.si_pid!=0;}
        if(done && output[0]<0 && error[0]<0){break;}if(now>=deadline){*kind=3;break;}
        struct pollfd fds[2]={{output[0],POLLIN,0},{error[0],POLLIN,0}};int waited=poll(fds,2,10);
        if(waited<0 && errno!=EINTR){code=(uint32_t)errno;break;}if(waited>=0 && ((fds[0].revents|fds[1].revents)&POLLNVAL)){code=EBADF;break;}
    }
    if(pid>0){
        /* The leader remains unreaped until signaling; no reused PID is killed. */
        if(!done || code!=0 || *kind!=0){if(kill(pid,SIGKILL)!=0 && errno!=ESRCH && code==0){code=(uint32_t)errno;}}
        pid_t result;do{result=waitpid(pid,&status,0);}while(result<0 && errno==EINTR);
        if(result<0 && code==0){code=(uint32_t)errno;}
        if(code==0 && *kind==0){if(WIFEXITED(status)){*kind=1;*value=(uint32_t)WEXITSTATUS(status);}else if(WIFSIGNALED(status)){*kind=2;*value=(uint32_t)WTERMSIG(status);}else{code=EIO;}}
    }
    closed=fy_linux_capture_close(&output[0]);if(code==0){code=closed;}closed=fy_linux_capture_close(&error[0]);if(code==0){code=closed;}return code;
}

/* Descriptor-relative replacement. Reject links/nonregular destinations; never
   remove the old image to make rename work. The parent directory is not fsynced. */
uint32_t fairyc_private_linux_gnu_publish_v1(const uint8_t *root,size_t root_size,const uint8_t *path,size_t size,const uint8_t *data,size_t length) {
    if(!fy_linux_root_path(root,root_size,0) || !fy_linux_root_path(path,size,0) || path[0]=='/' || data==NULL || length==0 || length>67108864){return EINVAL;}
    char relative[4096];memcpy(relative,path,size);int directory=open((const char *)root,O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);if(directory<0){return (uint32_t)errno;}
    uint32_t code=0;char *leaf=relative;size_t depth=0;
    for(char *cursor=relative;;++cursor){if(*cursor!='/' && *cursor!=0){continue;}int last=*cursor==0;*cursor=0;
        if(*leaf==0 || strlen(leaf)>255 || strcmp(leaf,".")==0 || strcmp(leaf,"..")==0 || ++depth>64){code=EINVAL;break;}if(last){break;}
        if(mkdirat(directory,leaf,0755)!=0 && errno!=EEXIST){code=(uint32_t)errno;break;}
        int child=openat(directory,leaf,O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC);if(child<0){code=(uint32_t)errno;break;}(void)close(directory);directory=child;leaf=cursor+1;
    }
    struct stat info;if(code==0){if(fstatat(directory,leaf,&info,AT_SYMLINK_NOFOLLOW)==0){if(!S_ISREG(info.st_mode)){code=EINVAL;}}else if(errno!=ENOENT){code=(uint32_t)errno;}}
    char temporary[64]={0};int file=-1;
    if(code==0){unsigned char nonce[16];size_t used=0;while(used<sizeof nonce){ssize_t n=getrandom(nonce+used,sizeof nonce-used,0);if(n<0 && errno==EINTR){continue;}if(n<=0){code=n<0?(uint32_t)errno:EIO;break;}used+=(size_t)n;}
        if(code==0){static const char hex[]="0123456789abcdef";memcpy(temporary,".fairyc-candidate-",18);for(size_t i=0;i<16;++i){temporary[18+i*2]=hex[nonce[i]>>4];temporary[19+i*2]=hex[nonce[i]&15];}memcpy(temporary+50,".tmp",5);
            file=openat(directory,temporary,O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW|O_CLOEXEC,0700);if(file<0){code=(uint32_t)errno;}
        }
    }
    int created=file>=0;
    if(code==0){code=fy_linux_gnu_write(file,data,length);}if(code==0 && fchmod(file,0700)!=0){code=(uint32_t)errno;}if(code==0 && fsync(file)!=0){code=(uint32_t)errno;}
    if(file>=0 && close(file)!=0 && code==0){code=(uint32_t)errno;}
    if(code==0 && renameat(directory,temporary,directory,leaf)!=0){code=(uint32_t)errno;}
    if(created){(void)unlinkat(directory,temporary,0);}if(close(directory)!=0 && code==0){code=(uint32_t)errno;}return code;
}
