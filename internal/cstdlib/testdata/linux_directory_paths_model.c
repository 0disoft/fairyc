#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <stdarg.h>
#include <errno.h>
#ifndef ESTALE
#define ESTALE 116
#endif
typedef uint32_t mode_t;
typedef uint64_t dev_t;
struct stat {mode_t st_mode;int64_t st_size;dev_t st_dev;uint64_t st_ino;};
struct dirent {char d_name[64];};
typedef struct {int descriptor,position;struct dirent entry;} DIR;
#define O_RDONLY 0
#define O_DIRECTORY 1
#define O_NOFOLLOW 2
#define O_CLOEXEC 4
#define AT_SYMLINK_NOFOLLOW 256
#define AT_REMOVEDIR 512
#define SYS_renameat2 316
#define RENAME_NOREPLACE 1
#define S_ISREG(m) ((m)==1)
#define S_ISDIR(m) ((m)==2)
#define S_ISLNK(m) ((m)==3)
struct fy_std_file_result_v0 {uint32_t status;int32_t error_number;size_t count;};
static struct fy_std_file_result_v0 fy_std_file_result_v0(uint32_t status,int32_t code,size_t count){struct fy_std_file_result_v0 r={status,code,count};return r;}
static int fy_std_file_path_storage_length_v0(const uint8_t *path,size_t *length){if(path==NULL)return 0;for(size_t i=0;i<4096;i++){if(!path[i]){*length=i+1;return i!=0;}}return 0;}
struct node {int exists,parent,kind;dev_t device;uint64_t inode;char name[64];};
static struct node nodes[80];
static int model_handles,model_stream_failure;
static uint64_t next_inode=100;
static int find_node(int parent,const char *name){if(!strcmp(name,"."))return parent;if(!strcmp(name,".."))return nodes[parent].parent?nodes[parent].parent:1;for(int i=2;i<80;i++)if(nodes[i].exists&&nodes[i].parent==parent&&!strcmp(nodes[i].name,name))return i;errno=ENOENT;return -1;}
static void assign_node(int id,int parent,int kind,const char *name){nodes[id].exists=1;nodes[id].parent=parent;nodes[id].kind=kind;nodes[id].device=1;nodes[id].inode=next_inode++;memcpy(nodes[id].name,name,strlen(name)+1);}
static void reset_tree(void){memset(nodes,0,sizeof nodes);assign_node(1,0,2,".");assign_node(2,1,2,"root");assign_node(3,2,1,"file");assign_node(4,2,3,"link");assign_node(5,2,2,"sub");assign_node(6,5,1,"nested");assign_node(7,1,2,"outside");assign_node(8,7,1,"valuable");model_handles=0;model_stream_failure=0;}
static int model_open(const char *path,int flags){if((strcmp(path,".")&&strcmp(path,"/"))||(flags&(O_NOFOLLOW|O_DIRECTORY|O_CLOEXEC))!=(O_NOFOLLOW|O_DIRECTORY|O_CLOEXEC)){errno=EINVAL;return -1;}model_handles++;return 1;}
static int model_openat(int parent,const char *name,int flags){int id=find_node(parent,name);if(id<0)return -1;if(!(flags&O_NOFOLLOW)){errno=EINVAL;return -1;}if(nodes[id].kind!=2){errno=nodes[id].kind==3?ELOOP:ENOTDIR;return -1;}model_handles++;return id;}
static int model_close(int descriptor){(void)descriptor;model_handles--;return 0;}
static int model_mkdirat(int parent,const char *name,mode_t mode){(void)mode;if(find_node(parent,name)>=0){errno=EEXIST;return -1;}for(int i=2;i<80;i++)if(!nodes[i].exists){assign_node(i,parent,2,name);return 0;}errno=ENOMEM;return -1;}
static int model_unlinkat(int parent,const char *name,int flags){int id=find_node(parent,name);if(id<0)return -1;if(flags==AT_REMOVEDIR){if(nodes[id].kind!=2){errno=ENOTDIR;return -1;}for(int i=2;i<80;i++)if(nodes[i].exists&&nodes[i].parent==id){errno=ENOTEMPTY;return -1;}}else if(flags!=0||nodes[id].kind==2){errno=EISDIR;return -1;}nodes[id].exists=0;return 0;}
static int model_fstat(int descriptor,struct stat *out){if(descriptor<1||!nodes[descriptor].exists){errno=EBADF;return -1;}out->st_mode=(mode_t)nodes[descriptor].kind;out->st_size=3;out->st_dev=nodes[descriptor].device;out->st_ino=nodes[descriptor].inode;return 0;}
static int model_fstatat(int parent,const char *name,struct stat *out,int flags){if(flags!=AT_SYMLINK_NOFOLLOW){errno=EINVAL;return -1;}int id=find_node(parent,name);return id<0?-1:model_fstat(id,out);}
static DIR *model_fdopendir(int descriptor){if(model_stream_failure){errno=EIO;return NULL;}DIR *d=calloc(1,sizeof *d);if(d){d->descriptor=descriptor;d->position=2;}return d;}
static struct dirent *model_readdir(DIR *d){for(int i=d->position;i<80;i++){d->position=i+1;if(nodes[i].exists&&nodes[i].parent==d->descriptor){memcpy(d->entry.d_name,nodes[i].name,strlen(nodes[i].name)+1);return &d->entry;}}return NULL;}
static int model_dirfd(DIR *d){return d->descriptor;}
static int model_closedir(DIR *d){model_handles--;free(d);return 0;}
static long model_syscall(long number,...){va_list args;va_start(args,number);int first_parent=va_arg(args,int);const char *first=va_arg(args,const char *);int second_parent=va_arg(args,int);const char *second=va_arg(args,const char *);int flags=va_arg(args,int);va_end(args);if(number!=SYS_renameat2||flags!=RENAME_NOREPLACE){errno=EINVAL;return -1;}int source=find_node(first_parent,first);if(source<0)return -1;if(find_node(second_parent,second)>=0){errno=EEXIST;return -1;}nodes[source].parent=second_parent;memcpy(nodes[source].name,second,strlen(second)+1);return 0;}
#define open model_open
#define openat model_openat
#define close model_close
#define mkdirat model_mkdirat
#define unlinkat model_unlinkat
#define fstat model_fstat
#define fstatat model_fstatat
#define fdopendir model_fdopendir
#define readdir model_readdir
#define dirfd model_dirfd
#define closedir model_closedir
#define syscall model_syscall
/* PRODUCTION */
#define CHECK(x) do{if(!(x))return __LINE__;}while(0)
int main(void){reset_tree();
CHECK(fy_std_windows_directory_ensure_v0((const uint8_t *)"new/deep").status==0&&model_handles==0);int fresh=find_node(1,"new");CHECK(fresh>0&&find_node(fresh,"deep")>0);CHECK(fy_std_windows_directory_create_v0((const uint8_t *)"created").status==0);CHECK(fy_std_windows_directory_create_v0((const uint8_t *)"created").status==1);CHECK(fy_std_windows_directory_remove_v0((const uint8_t *)"created").status==0&&model_handles==0);
CHECK(fy_std_windows_directory_rename_v0((const uint8_t *)"root",(const uint8_t *)"outside").status==1&&nodes[2].exists&&nodes[7].exists&&model_handles==0);CHECK(fy_std_windows_directory_rename_v0((const uint8_t *)"root",(const uint8_t *)"renamed").status==0&&find_node(1,"renamed")==2);CHECK(fy_std_windows_directory_rename_v0((const uint8_t *)"renamed",(const uint8_t *)"root").status==0&&model_handles==0);
struct fy_std_file_result_v0 result=fy_std_windows_dir_remove_all_v0((const uint8_t *)"root");CHECK(result.status==0&&result.count==5&&!nodes[2].exists&&!nodes[4].exists&&nodes[7].exists&&nodes[8].exists&&model_handles==0);
reset_tree();nodes[5].device=2;result=fy_std_windows_dir_remove_all_v0((const uint8_t *)"root");CHECK(result.status==1&&result.error_number==EXDEV&&result.count==2&&nodes[2].exists&&nodes[5].exists&&nodes[8].exists&&model_handles==0);
reset_tree();CHECK(fy_std_windows_dir_remove_all_v0((const uint8_t *)"root/link").status==2&&nodes[4].exists&&nodes[8].exists&&model_handles==0);CHECK(fy_std_windows_directory_ensure_v0((const uint8_t *)"root/link/child").status==1&&model_handles==0&&nodes[8].exists);CHECK(fy_std_windows_dir_remove_all_v0((const uint8_t *)".").status==2&&nodes[2].exists&&model_handles==0);CHECK(fy_std_windows_dir_remove_all_v0((const uint8_t *)"root/..").status==2&&nodes[2].exists&&model_handles==0);CHECK(fy_std_windows_dir_remove_all_v0((const uint8_t *)"/").status==2&&model_handles==0);
reset_tree();model_stream_failure=1;result=fy_std_windows_dir_remove_all_v0((const uint8_t *)"root");CHECK(result.status==1&&result.error_number==EIO&&result.count==0&&nodes[2].exists&&nodes[3].exists&&model_handles==0);
reset_tree();memset(nodes+3,0,sizeof nodes-sizeof nodes[0]*3);for(int i=3;i<69;i++)assign_node(i,i-1,2,"child");result=fy_std_windows_dir_remove_all_v0((const uint8_t *)"root");CHECK(result.status==1&&result.error_number==EOVERFLOW&&result.count==0&&nodes[2].exists&&nodes[68].exists&&model_handles==0);
reset_tree();int code;int root=fy_std_linux_directory_open_path_v0((const uint8_t *)"root",4,0,&code);size_t remaining=0,removed=0;CHECK(root>=0&&fy_std_linux_directory_remove_children_v0(root,0,1,&remaining,&removed)==EOVERFLOW&&removed==0&&model_handles==0&&nodes[3].exists);return 0;}
