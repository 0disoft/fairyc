package main

const dependencyRootedUnitC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stddef.h>
#include <wchar.h>
#include <stdio.h>
size_t fairyc_private_windows_directory_open_v1(const uint8_t *,size_t);
size_t fairyc_private_windows_root_file_open_v1(size_t,const uint8_t *,size_t,uint32_t);
int32_t fairyc_private_windows_root_file_write_v1(size_t,const uint8_t *,size_t,size_t *);
int32_t fairyc_private_windows_root_file_rename_v1(size_t,size_t,const uint8_t *,size_t);
int32_t fairyc_private_windows_root_file_delete_v1(size_t);
int32_t fairyc_private_windows_root_file_close_v1(size_t);
int wmain(int argc,WCHAR **argv) {
    if(argc!=3)return 1;
    int missing_open=wcscmp(argv[2],L"missing-open")==0;
    int missing_set=wcscmp(argv[2],L"missing-set")==0;
    size_t root=fairyc_private_windows_directory_open_v1((const uint8_t *)argv[1],(wcslen(argv[1])+1)*2);
    if(!root)return 2;
    const WCHAR first[]=L"before.tmp",second[]=L"after.tmp",missing[]=L"absent.tmp";
    size_t file=fairyc_private_windows_root_file_open_v1(root,(const uint8_t *)first,(sizeof(first)-2),2);
    if(missing_open) {
        DWORD error=GetLastError();
        if(file)fairyc_private_windows_root_file_close_v1(file);
        fairyc_private_windows_root_file_close_v1(root);
        if(file || error!=ERROR_PROC_NOT_FOUND)return 3;
        puts("dynamic: missing-open error=127");return 42;
    }
    if(!file)return 4;
    const uint8_t data=42;size_t written=0;
    if(!fairyc_private_windows_root_file_write_v1(file,&data,1,&written) || written!=1)return 5;
    int32_t renamed=fairyc_private_windows_root_file_rename_v1(file,root,(const uint8_t *)second,sizeof(second)-2);
    DWORD rename_error=GetLastError();
    if(!fairyc_private_windows_root_file_delete_v1(file) || !fairyc_private_windows_root_file_close_v1(file))return 6;
    if(missing_set) {
        fairyc_private_windows_root_file_close_v1(root);
        if(renamed || rename_error!=ERROR_PROC_NOT_FOUND)return 7;
        puts("dynamic: missing-set error=127");return 42;
    }
    if(!renamed)return 8;
    SetLastError(0);
    file=fairyc_private_windows_root_file_open_v1(root,(const uint8_t *)missing,sizeof(missing)-2,1);
    DWORD error=GetLastError();
    if(file)fairyc_private_windows_root_file_close_v1(file);
    fairyc_private_windows_root_file_close_v1(root);
    if(file || error!=ERROR_FILE_NOT_FOUND)return 9;
    puts("dynamic: create+rename+NTSTATUS mapping");return 42;
}
`
