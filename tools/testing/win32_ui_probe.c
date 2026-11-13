#define UNICODE
#define _UNICODE
#include "native/ui_core.h"
#include <stdio.h>
#include <string.h>
static unsigned checks;
#define CHECK(test) do { ++checks; if (!(test)) { fprintf(stderr,"UI core check failed at %d\n",__LINE__);return 1; } } while (0)
int main(void) {
    CHECK(fairy_ui_should_pump(FALSE,TRUE,TRUE));
    CHECK(!fairy_ui_should_pump(TRUE,TRUE,TRUE));
    CHECK(!fairy_ui_should_pump(FALSE,FALSE,TRUE));
    CHECK(fairy_ui_should_pump(TRUE,FALSE,FALSE));
    const uint8_t unicode[]={0xed,0x95,0x9c,0xea,0xb8,0x80,0x20,0xce,0xa9};
    WCHAR *wide=fairy_ui_wide(unicode,sizeof(unicode));CHECK(wide && wide[0]==0xd55c && wide[1]==0xae00 && wide[3]==0x3a9 && !wide[4]);
    char utf8[16];int count=WideCharToMultiByte(CP_UTF8,WC_ERR_INVALID_CHARS,wide,4,utf8,sizeof(utf8),NULL,NULL);
    CHECK(count==sizeof(unicode) && !memcmp(utf8,unicode,sizeof(unicode)));HeapFree(GetProcessHeap(),0,wide);
    wide=fairy_ui_wide(NULL,0);CHECK(wide && !*wide);HeapFree(GetProcessHeap(),0,wide);
    const uint8_t invalid[]={0xc0,0xaf},nul[]={'a',0,'b'};
    CHECK(!fairy_ui_wide(invalid,sizeof(invalid)));CHECK(!fairy_ui_wide(nul,sizeof(nul)));
    CHECK(!fairy_ui_wide(NULL,1));CHECK(!fairy_ui_wide(unicode,131072));
    RECT source={10,20,1010,720},copy={0};
    CHECK(fairy_ui_dpi_suggestion(MAKEWPARAM(144,144),(LPARAM)&source,&copy));
    source.left=99;CHECK(copy.left==10 && copy.bottom==720);
    CHECK(!fairy_ui_dpi_suggestion(MAKEWPARAM(144,192),(LPARAM)&source,&copy));
    CHECK(!fairy_ui_dpi_suggestion(0,(LPARAM)&source,&copy));
    CHECK(!fairy_ui_dpi_suggestion(MAKEWPARAM(144,144),0,&copy));
    source.right=source.left;CHECK(!fairy_ui_dpi_suggestion(MAKEWPARAM(144,144),(LPARAM)&source,&copy));
    CHECK(!fairy_ui_open(NULL,NULL,0,0));CHECK(!fairy_ui_create_child(NULL,L"EDIT",0,101));
    CHECK(fairy_ui_text(NULL,unicode,sizeof(unicode))==1);CHECK(fairy_ui_read(NULL,(uint8_t *)utf8,sizeof(utf8))==SIZE_MAX);
    CHECK(fairy_ui_focus(NULL)==1 && fairy_ui_next_focus(NULL,0)==1);
    CHECK(fairy_ui_enable(NULL,1)==1 && fairy_ui_move(NULL,0,0,10,10)==1);
    CHECK(fairy_ui_extent(NULL,0)==-1 && fairy_ui_redraw(NULL)==1);
    CHECK(!fairy_ui_apply_dpi(NULL,&copy));CHECK(fairy_ui_live(NULL,NULL,1,0)==1);
    CHECK(fairy_ui_name(NULL,NULL,NULL,unicode,sizeof(unicode))==1);
    printf("{\"ok\":true,\"verification\":\"win32-ui-core\",\"checks\":%u,\"guiLaunched\":false,\"inputInjected\":false}\n",checks);
    return 0;
}
