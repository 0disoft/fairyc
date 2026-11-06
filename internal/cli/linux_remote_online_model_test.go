package cli

import (
	"bytes"
	"strings"
	"testing"
)

func linuxOnlineFaults(t *testing.T, fixtures []*linuxOnlineFixture) []*linuxOnlineFixture {
	t.Helper()
	var base *linuxOnlineFixture
	for _, f := range fixtures {
		if f.name == "download/deflated" {
			base = f
		}
	}
	if base == nil || !base.ok || base.calls != 1 {
		t.Fatal("missing cold-download reference fixture")
	}
	var result []*linuxOnlineFixture
	add := func(name string, mode, fault uint32, output string) *linuxOnlineFixture {
		f := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{name: "controls/" + name, initial: linuxOnlineClone(base.initial), final: linuxOnlineClone(base.initial), output: output, fault: fault}, mode: mode, calls: 1, closes: 1, replies: make(map[string][]byte)}
		for name, data := range base.replies {
			f.replies[name] = bytes.Clone(data)
		}
		result = append(result, f)
		return f
	}
	for _, c := range []struct {
		name, output string
		mode         uint32
	}{
		{"factory-error", "error 10 3 5 0\n", 1}, {"partial-factory-and-close", "error 10 3 5 63\n", 2},
		{"read-error", "error 10 3 62 0\n", 3}, {"read-and-close-error", "error 10 3 62 63\n", 4},
		{"successful-body-close-error", "error 10 3 0 63\n", 5}, {"redirect", "error 10 4 0 0\n", 6},
		{"content-encoding", "error 10 4 0 0\n", 7}, {"declared-byte-limit", "error 10 5 0 0\n", 8},
		{"native-overread", "error 10 3 0 0\n", 9}, {"empty-factory-success", "error 10 3 0 0\n", 10},
		{"cache-zero-write", "error 11 1 0 0\n", 11}, {"cache-sync-error", "error 11 1 0 0\n", 12},
		{"cache-close-error", "error 11 1 0 1\n", 13}, {"cache-collision-differs", "error 11 4 0 0\n", 15},
		{"deadline-after-download", "error 9 3 0 0\n", 16}, {"deadline-before-transport", "error 9 3 0 0\n", 17},
		{"clock-start-error", "error 9 5 5 0\n", 20}, {"native-timeout", "error 10 3 110 0\n", 21},
		{"native-header-error", "error 10 3 61 0\n", 22},
	} {
		f := add(c.name, c.mode, 0, c.output)
		if c.mode == 1 || c.mode == 10 || c.mode == 21 {
			f.closes = 0
		}
		if c.mode == 17 || c.mode == 20 {
			f.calls, f.closes = 0, 0
		}
		if c.mode == 13 || c.mode == 15 {
			for name, data := range base.final {
				if strings.HasSuffix(name, ".zip") {
					f.final[name] = bytes.Clone(data)
					if c.mode == 15 {
						f.final[name][0] ^= 1
					}
				}
			}
		}
	}
	for _, c := range []struct {
		name string
		mode uint32
	}{{"matching-cache-competitor", 14}, {"remaining-request-budget-15000", 18}, {"remaining-request-budget-one", 19}} {
		f := add(c.name, c.mode, 0, base.output)
		f.ok = true
		f.final = linuxOnlineClone(base.final)
	}
	f := add("wrong-download-digest", 0, 0, "error 5 1 0 0\n")
	for name := range f.replies {
		f.replies[name] = []byte("wrong bytes")
	}
	f = add("directory-publication-collision", 0, 13, "error 7 1 0 0\n")
	f.final["vendor"] = nil
	for name, data := range base.final {
		if strings.HasSuffix(name, ".zip") {
			f.final[name] = bytes.Clone(data)
		}
	}
	for _, c := range []struct {
		name, output string
		mode         uint32
	}{
		{"read-and-lock-cleanup", "error 10 3 62 1\n", 3},
		{"read-close-and-lock-cleanup", "error 10 3 62 63\n", 4},
	} {
		f = add(c.name, c.mode, 8, c.output)
		f.final[".fairyc-fetch.lock"] = nil
	}
	f = add("competing-lock-after-observation", 0, 14, "error 3 1 17 0\n")
	f.calls, f.closes = 0, 0
	f.final[".fairyc-fetch.lock"] = nil
	f.final[".fairyc-fetch.lock/keep"] = []byte("foreign lock")
	f = add("successful-body-lock-cleanup", 0, 8, "error 8 1 0 0\n")
	f.final = linuxOnlineClone(base.final)
	f.final[".fairyc-fetch.lock"] = nil
	return result
}

const linuxOnlineModel = `
static uint32_t online_mode,online_opens,online_closes,online_live,online_cache_publications;
static size_t online_position,online_reply_count,online_current;
struct online_payload{char address[4096];uint8_t data[65536];size_t size;};
static struct online_payload online_payloads[8];
static size_t online_now;
static void online_reply(const char *address,size_t size,uint32_t kind,const uint8_t *data){
    assert(kind==32768 && online_reply_count<8 && strlen(address)<4096 && size<=65536);
    struct online_payload *p=&online_payloads[online_reply_count++];memcpy(p->address,address,strlen(address)+1);memcpy(p->data,data,size);p->size=size;
}
uint32_t fairyc_private_linux_monotonic_ms_v1(size_t *output){
    ++offline_calls;*output=online_now;if(online_mode==20 && offline_calls==1){*output=0;return 5;}
    if(offline_calls==4){if(online_mode==17){online_now=121000;}if(online_mode==18){online_now=106000;}if(online_mode==19){online_now=120999;}}*output=online_now;return 0;
}
uint32_t fairyc_private_linux_https_open_v1(const uint8_t *address,size_t size,size_t maximum,size_t timeout,size_t *output){
    assert(online_live==0 && size==strlen((const char *)address)+1 && maximum==16777216 && timeout==(online_mode==18?15000u:online_mode==19?1u:30000u));
    ++online_opens;*output=0;online_current=online_reply_count;
    for(size_t i=0;i<online_reply_count;++i){if(strcmp((const char *)address,online_payloads[i].address)==0){online_current=i;break;}}
    assert(online_current<online_reply_count && online_opens<=online_reply_count);
    if(online_mode==1){return 5;}if(online_mode==10){return 0;}if(online_mode==21){return 110;}
    online_live=1;online_position=0;*output=77;return online_mode==2?5:0;
}
uint32_t fairyc_private_linux_https_headers_v1(size_t raw,uint32_t *status,uint8_t *encoding,size_t capacity,size_t *length,size_t *declared){
    assert(raw==77 && online_live && capacity==128);*status=online_mode==6?302:200;*length=0;*declared=online_payloads[online_current].size;
    if(online_mode==7){memcpy(encoding,"gzip",4);*length=4;}if(online_mode==8){*declared=16777217;}return online_mode==22?61:0;
}
uint32_t fairyc_private_linux_https_read_v1(size_t raw,uint8_t *data,size_t capacity,size_t *copied){
    assert(raw==77 && online_live && capacity>0 && capacity<=65536);*copied=0;
    if(online_mode==3 || online_mode==4){return 62;}if(online_mode==9){*copied=capacity+1;return 0;}
    struct online_payload *p=&online_payloads[online_current];assert(online_position<=p->size);size_t count=p->size-online_position;
    if(count>capacity){count=capacity;}if(count>25){count=25;}memcpy(data,p->data+online_position,count);online_position+=count;*copied=count;return 0;
}
uint32_t fairyc_private_linux_https_close_v1(size_t raw){
    assert(raw==77 && online_live);online_live=0;++online_closes;
    if(online_mode==11){remote_case=9;}if(online_mode==12){remote_case=7;}if(online_mode==13){remote_case=6;}
    if(online_mode==14 || online_mode==15){remote_case=online_mode;}
    if(online_mode==16){online_now=121000;}return online_mode==2 || online_mode==4 || online_mode==5?63:0;
}
uint32_t fairyc_private_linux_root_file_publish_new_v1(uint32_t root,uint32_t handle,const uint8_t *leaf,size_t size){
    int cache=size==69 && memcmp(leaf+64,".zip",4)==0;if(cache){++online_cache_publications;}
    uint32_t code=online_base_file_publish(root,handle,leaf,size);if(cache && (online_mode==14 || online_mode==15)){remote_case=0;}return code;
}
`

const linuxOnlineModelEntry = `
void online_probe_setup(void){
    const char *value=getenv("FAIRYC_ONLINE_CASE");assert(value!=NULL);offline_index=(uint32_t)strtoul(value,NULL,10);
    offline_calls=online_mode=online_opens=online_closes=online_live=online_cache_publications=0;online_reply_count=online_position=online_current=0;online_now=1000;
    remote_fs_probe_mode(0);online_fixture(offline_index);
}
void online_probe_finish(void){online_expected(offline_index);}
`
