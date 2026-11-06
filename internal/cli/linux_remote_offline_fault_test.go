package cli

import (
	"strings"
	"testing"
)

// These are independent fault expectations, not Go-parity claims. The model
// injects Linux-specific links, retained-name collisions and uncertain errors.
func linuxOfflineFaults(t *testing.T, base *linuxOfflineFixture) []*linuxOfflineFixture {
	t.Helper()
	clone := func(tree map[string][]byte) map[string][]byte {
		copy := make(map[string][]byte, len(tree))
		for name, data := range tree {
			copy[name] = data
		}
		return copy
	}
	var cache string
	for name := range base.initial {
		if strings.HasSuffix(name, ".zip") {
			cache = name
		}
	}
	if cache == "" {
		t.Fatal("control archive missing")
	}
	stage := "vendor/.fairyc-source-" + strings.TrimSuffix(strings.TrimPrefix(cache, "build/package-sources/"), ".zip")
	var fixtures []*linuxOfflineFixture
	add := func(id uint32, name, output string) *linuxOfflineFixture {
		f := &linuxOfflineFixture{name: "controls/" + name, initial: clone(base.initial), final: clone(base.initial), fault: id, output: output, kinds: map[string]uint32{}}
		fixtures = append(fixtures, f)
		return f
	}
	add(1, "cache-hardlink", "error 4 4 0\n")
	f := add(2, "cache-symlink", "error 4 4 0\n")
	f.kinds[cache] = 40960
	f = add(3, "member-symlink", "error 7 4 0\n")
	f.initial["vendor"] = nil
	f.initial["vendor/math"] = []byte{}
	f.final = clone(f.initial)
	f.kinds["vendor/math"] = 40960
	f = add(4, "stale-stage", "error 7 4 0\n")
	f.initial["vendor"] = nil
	f.initial[stage] = nil
	f.initial[stage+"/keep"] = []byte("preserve stale stage")
	f.final = clone(f.initial)
	for _, control := range []struct {
		id           uint32
		name, output string
	}{
		{5, "zero-write", "error 7 1 0\n"}, {6, "sync-error", "error 7 1 0\n"},
		{7, "stage-cleanup-error", "error 7 1 1\n"}, {12, "deadline-before-publication", "error 7 3 0\n"},
		{13, "publication-collision", "error 7 1 0\n"}, {15, "partial-native-stage-error", "error 7 1 88\n"},
	} {
		f = add(control.id, control.name, control.output)
		f.final["vendor"] = nil
		if control.id == 7 || control.id == 15 {
			f.final[stage] = nil
		}
	}
	f = add(8, "lock-cleanup-error", "error 8 1 0\n")
	f.final[".fairyc-fetch.lock"] = nil
	f.final["vendor"] = nil
	f.final["vendor/math"] = nil
	f.final["vendor/math/src"] = nil
	f.final["vendor/math/fairy.toml"] = []byte(remoteLibraryManifest)
	f.final["vendor/math/src/math.fy"] = []byte("module math;pub fn i32 answer(){return 42;}\n")
	f.final["vendor/math/.review"] = []byte{0, 255, 1, 2}
	add(9, "clock-start-error", "error 9 5 0\n")
	add(10, "clock-error-before-lock", "error 9 3 0\n")
	add(11, "deadline-after-lock", "error 9 3 0\n")
	f = add(14, "competing-lock-after-observation", "error 3 1 0\n")
	f.final[".fairyc-fetch.lock"] = nil
	f.final[".fairyc-fetch.lock/keep"] = []byte("foreign lock")
	return fixtures
}

const linuxOfflineFaultModel = `
static uint32_t offline_creates,offline_directory_publications;
static void offline_alter(void){
    offline_creates=offline_directory_publications=0;
    if(offline_fault==1){for(size_t i=1;i<remote_node_count;++i){if(strstr(remote_nodes[i].name,".zip")!=NULL){remote_nodes[i].links=2;}}}
    if(offline_fault==5 || offline_fault==7){remote_case=9;}if(offline_fault==6){remote_case=7;}
}
uint32_t fairyc_private_linux_package_directory_create_v1(uint32_t root,const uint8_t *leaf,size_t size,uint32_t *child,uint32_t *parent,uint32_t *closed){
    ++offline_creates;
    if(offline_fault==14 && strcmp((const char *)leaf,".fairyc-fetch.lock")==0){
        assert(offline_creates==1);offline_seed(".fairyc-fetch.lock",0,16384,NULL);
        static const uint8_t keep[]="foreign lock";offline_seed(".fairyc-fetch.lock/keep",sizeof keep-1,32768,keep);*child=*parent=*closed=0;return 17;
    }
    uint32_t code=offline_base_create(root,leaf,size,child,parent,closed);
    if(offline_fault==15 && strncmp((const char *)leaf,".fairyc-source-",15)==0 && code==0){*closed=88;return 5;}return code;
}
uint32_t fairyc_private_linux_package_directory_publish_v1(uint32_t child,uint32_t parent,const uint8_t *from,size_t from_size,uint32_t root,const uint8_t *leaf,size_t size){
    ++offline_directory_publications;
    if(offline_fault==13){assert(offline_directory_publications==1);return 17;}return offline_base_publish(child,parent,from,from_size,root,leaf,size);
}
uint32_t fairyc_private_linux_package_directory_erase_v1(uint32_t child,uint32_t parent,const uint8_t *leaf,size_t size){
    if((offline_fault==7 && strncmp((const char *)leaf,".fairyc-source-",15)==0) || (offline_fault==8 && strcmp((const char *)leaf,".fairyc-fetch.lock")==0)){return 5;}
    return offline_base_erase(child,parent,leaf,size);
}
`
