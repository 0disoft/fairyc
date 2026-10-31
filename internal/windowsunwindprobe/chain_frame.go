package windowsunwindprobe

// Independent instruction-boundary stacks test partial prologs, fixed-frame
// chained bodies and already-restored epilogs. Wrong stacks never run as code.
const FrameChainCompanionC = `
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
extern int fy_chain(void);
static void fy_frame_return(void) { }
static DWORD64 *fy_frame_register(CONTEXT *c,unsigned int r) {
    switch(r) {case 3:return &c->Rbx;case 5:return &c->Rbp;case 6:return &c->Rsi;case 7:return &c->Rdi;
        case 12:return &c->R12;case 13:return &c->R13;case 14:return &c->R14;case 15:return &c->R15;default:return NULL;}
}
static int fy_frame_context(unsigned int offset,unsigned int wrong) {
    __declspec(align(16)) DWORD64 stack[32] = {0};
    CONTEXT expected,context;
    RtlCaptureContext(&expected);
    for(unsigned int r=0;r<16;++r) {DWORD64 *v=fy_frame_register(&expected,r);if(v)*v=UINT64_C(0x1122334400000000)+r;}
    for(unsigned int i=0;i<10;++i) {(&expected.Xmm6)[i].Low=UINT64_C(0x8877665500000000)+i;(&expected.Xmm6)[i].High=(LONGLONG)(UINT64_C(0x6655443300000000)+i);}
    stack[8]=expected.Rbp;stack[9]=(DWORD64)(uintptr_t)&fy_frame_return;
    memcpy(&stack[wrong==1?6:4],&expected.Xmm6,16);
    expected.Rip=stack[9];expected.Rsp=(DWORD64)(uintptr_t)&stack[10];
    context=expected;context.Rip=(DWORD64)(uintptr_t)&fy_chain+offset;
    unsigned int rsp=offset==0||offset==31?9:offset==1||offset==30?8:0;
    context.Rsp=(DWORD64)(uintptr_t)&stack[rsp];
    if(offset>=10&&offset<31) {context.Rbp=(DWORD64)(uintptr_t)&stack[4]+(wrong==2?8:0);}
    if(offset>=15&&offset<26) {context.Xmm6.Low=0xdead;context.Xmm6.High=0xbeef;}
    DWORD64 base=0,frame=0;PVOID data=NULL;
    PRUNTIME_FUNCTION entry=RtlLookupFunctionEntry(context.Rip,&base,NULL);if(!entry)return 59;
    PEXCEPTION_ROUTINE handler=RtlVirtualUnwind(UNW_FLAG_NHANDLER,base,context.Rip,entry,&context,&data,&frame,NULL);
    if(handler||context.Rip!=expected.Rip||context.Rsp!=expected.Rsp)return 60;
    for(unsigned int r=0;r<16;++r) {DWORD64 *v=fy_frame_register(&context,r);if(v&&*v!=*fy_frame_register(&expected,r))return 61;}
    if(memcmp(&context.Xmm6,&expected.Xmm6,10*sizeof(M128A)))return 62;
    // Establishing a new frame is checked only before the epilog starts.
    if(offset<26&&frame!=(DWORD64)(uintptr_t)&stack[offset<10?rsp:0])return 63;
    return 0;
}
int32_t fairyc_fixture_chain_link(void) {
    static const BYTE prolog[]={0x55,0x48,0x83,0xec,0x40,0x48,0x8d,0x6c,0x24,0x20,0x0f,0x29,0x74,0x24,0x20};
    static const unsigned int offsets[]={0,1,5,10,15,16,21,26,30,31};
    if(memcmp((const void *)(uintptr_t)&fy_chain,prolog,sizeof(prolog))||fy_chain()!=42)return 70;
    for(unsigned int i=0;i<sizeof(offsets)/sizeof(offsets[0]);++i) {int result=fy_frame_context(offsets[i],0);if(result){fprintf(stderr,"frame pc=%u error=%d\n",offsets[i],result);return result;}}
    if(fy_frame_context(15,1)!=62||fy_frame_context(15,2)!=60)return 71;
    printf("{\"contexts\":10,\"frameChecks\":7,\"vectorRestores\":3,\"rejectedContexts\":2}\n");
    return 42;
}
`
