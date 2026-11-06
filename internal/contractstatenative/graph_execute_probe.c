//go:build ignore

#include <stdio.h>
#include <string.h>
#include <fcntl.h>
#include <io.h>

static uint8_t fygr_probe_arena[1048576], fygr_probe_scratch[1048576], fygr_probe_values[1048576];
static uint8_t fygr_probe_trace[65536];
static size_t fygr_probe_trace_size;
static fycs_overlay_entry_v1 fygr_probe_writes[64];
static fycs_graph_epoch_v1 fygr_probe_epochs[64];

static uint64_t fygr_probe_number(size_t width) {
    size_t i; uint64_t value=0u;
    for (i=0u;i<width;++i) { int next=getchar(); if(next==EOF){return UINT64_MAX;} value=(value<<8u)|(uint8_t)next; }
    return value;
}
static void fygr_probe_put(uint64_t value,size_t width) {
    size_t i=width; while(i!=0u){--i;(void)putchar((int)((value>>(i*8u))&255u));}
}
static int32_t fygr_probe_observe(void *user,uint64_t operation,uint32_t site,const uint8_t key[32],
    uint8_t codec,const uint8_t *value,size_t length,uint8_t found) {
    size_t i; uint8_t *record;
    (void)user;(void)site;(void)key;
    if(length>sizeof(fygr_probe_trace)-fygr_probe_trace_size || sizeof(fygr_probe_trace)-fygr_probe_trace_size-length<14u){return FYCS_OV_INVALID;}
    record=fygr_probe_trace+fygr_probe_trace_size;
    for(i=0u;i<8u;++i){record[7u-i]=(uint8_t)(operation>>(i*8u));}
    record[8]=codec;record[9]=found;
    for(i=0u;i<4u;++i){record[13u-i]=(uint8_t)((uint64_t)length>>(i*8u));}
    if(length!=0u){memcpy(record+14u,value,length);}
    fygr_probe_trace_size+=14u+length;
    return FYCS_OV_OK;
}
int main(void) {
    fycs_overlay_v1 transaction={0};fycs_graph_v1 context={0};
    size_t max_writes,max_value,max_key,i;
    uint64_t max_bytes;
    uint8_t exit_kind=0u;uint32_t trap_kind=0u;
    int32_t status;
    if(_setmode(_fileno(stdin),_O_BINARY)==-1 || _setmode(_fileno(stdout),_O_BINARY)==-1){return 1;}
    max_writes=(size_t)fygr_probe_number(8u);max_bytes=fygr_probe_number(8u);
    max_value=(size_t)fygr_probe_number(8u);max_key=(size_t)fygr_probe_number(8u);
    if(max_writes==0u || max_writes>64u || max_bytes==0u || max_bytes>1048576u || max_value==0u || max_key==0u || fgetc(stdin)!=EOF){return 2;}
    status=fycs_overlay_begin_v1(&transaction,NULL,0u,fygr_probe_writes,max_writes,fygr_probe_arena,fygr_probe_scratch,sizeof(fygr_probe_arena),max_bytes,max_value);
    if(status!=0){return 3;}
    status=fycs_graph_begin_v1(&context,&transaction,fygr_probe_values,sizeof(fygr_probe_values),fygr_probe_epochs,64u);
    if(status!=0){return 4;}
    status=fycs_graph_execute_v1(&context,max_key,fygr_probe_observe,NULL,&exit_kind,&trap_kind);
    fygr_probe_put((uint32_t)status,1u);fygr_probe_put(exit_kind,1u);fygr_probe_put(trap_kind,4u);
    fygr_probe_put(context.cost,8u);fygr_probe_put(fygr_probe_trace_size,4u);
    fygr_probe_put(transaction.count,2u);fygr_probe_put(transaction.write_bytes,8u);
    if(fygr_probe_trace_size!=0u){(void)fwrite(fygr_probe_trace,1u,fygr_probe_trace_size,stdout);}
    for(i=0u;i<transaction.count;++i){
        const fycs_overlay_entry_v1 *entry=&transaction.writes[i];
        (void)fwrite(entry->key,1u,32u,stdout);fygr_probe_put(entry->deleted,1u);fygr_probe_put(entry->length,4u);
        if(entry->length!=0u){(void)fwrite(entry->value,1u,entry->length,stdout);}
    }
    return fflush(stdout)==0?0:5;
}
