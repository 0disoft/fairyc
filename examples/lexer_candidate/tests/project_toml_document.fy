module tests::project_toml_document;
import project_toml_document as document;
import project_toml_model as m;
import std::bytes as bytes;

fn void manifest(ref u8[] source){
    switch(document::parse(source)){case .ok(tree):{
        let usize schema=m::lookup(ref tree,1,b"schema");assert(m::node(ref tree,schema).kind==2 && bytes::equal(m::value(ref tree,schema),b"fairyc.manifest/v1"));
        let usize package=m::lookup(ref tree,1,b"package");assert(m::node(ref tree,package).kind==1 && m::node(ref tree,package).children==5);
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,package,b"uid")),b"example.tools.app"));
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,package,b"name")),b"example-app"));
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,package,b"version")),b"1.2.3"));
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,package,b"language")),b"fairyc-language/20"));
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,package,b"source")),b"src"));
        let usize workspace=m::lookup(ref tree,1,b"workspace");let usize members=m::lookup(ref tree,workspace,b"members");let m::Node array=m::node(ref tree,members);
        assert(array.kind==3 && array.children==2 && bytes::equal(m::value(ref tree,array.first),b"libs/net") && bytes::equal(m::value(ref tree,array.last),b"libs/util"));
        let usize dependencies=m::lookup(ref tree,1,b"dependencies");let usize net=m::lookup(ref tree,dependencies,b"net");
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,net,b"path")),b"libs/net") && bytes::equal(m::value(ref tree,m::lookup(ref tree,net,b"version")),b"1.4.0"));
        let usize features=m::lookup(ref tree,net,b"features");assert(m::node(ref tree,features).kind==3 && m::node(ref tree,features).children==0);
        assert(m::source_bytes(ref tree)==source.len && m::count(ref tree)==17);
    }case .err(_):{assert(false);}};
}
fn void rejected(ref u8[] source,u32 code){switch(document::parse(source)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==code && e.start<=e.end && e.end<=source.len);}};}

@test
fn void reads_manifest_tables_dotted_keys_and_inline_forms(){
    manifest(b"schema = 'fairyc.manifest/v1'\n[package]\nuid='example.tools.app'\nname='example-app'\nversion='1.2.3'\nlanguage='fairyc-language/20'\nsource='src'\n[workspace]\nmembers=['libs/net', # item\n 'libs/util',]\n[dependencies.net]\npath='libs/net'\nversion='1.4.0'\nfeatures=[]\n");
    manifest(b"schema='fairyc.manifest/v1'\npackage.uid='example.tools.app'\npackage.name='example-app'\npackage.version='1.2.3'\npackage.language='fairyc-language/20'\npackage.source='src'\nworkspace.members=['libs/net','libs/util']\ndependencies.net.path='libs/net'\ndependencies.net.version='1.4.0'\ndependencies.net.features=[]");
    manifest(b"# header\r\n\"schema\"=\"fairyc.manifest/v1\"\r\n'package'={ uid='example.tools.app', name='example-app', version='1.2.3',\n# TOML 1.1 inline trivia\nlanguage='fairyc-language/20', source='src', }\r\nworkspace={members=['libs/net','libs/util']}\r\ndependencies={net={path='libs/net',version='1.4.0',features=[]}} # end");
}

@test
fn void keeps_table_array_scope_quoted_components_and_source_spans(){
    let ref u8[] source=b"\"a.b\"='flat'\na.b='nested'\n[implicit.child]\nx='x'\n[implicit]\ny='y'\n[[entry]]\nname='first'\n[entry.details]\nvalue='one'\n[[entry]]\nname='second'\n[entry.details]\nvalue='two'\narray=[['x'],{\"\"='\xc3\xa9', quoted=\"\"\"\nmultiline\"\"\"},]\n";
    switch(document::parse(source)){case .ok(tree):{
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,1,b"a.b")),b"flat"));
        let usize a=m::lookup(ref tree,1,b"a");assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,a,b"b")),b"nested"));
        let usize implicit=m::lookup(ref tree,1,b"implicit");assert(m::node(ref tree,implicit).state==2 && m::node(ref tree,implicit).children==2);
        let usize entry=m::lookup(ref tree,1,b"entry");let m::Node list=m::node(ref tree,entry);assert(list.kind==4 && list.children==2);
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,list.first,b"name")),b"first") && bytes::equal(m::value(ref tree,m::lookup(ref tree,list.last,b"name")),b"second"));
        let usize first_details=m::lookup(ref tree,list.first,b"details");let usize last_details=m::lookup(ref tree,list.last,b"details");assert(first_details!=last_details);
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,first_details,b"value")),b"one") && bytes::equal(m::value(ref tree,m::lookup(ref tree,last_details,b"value")),b"two"));
        let usize array=m::lookup(ref tree,last_details,b"array");let m::Node nested=m::node(ref tree,array);assert(nested.kind==3 && nested.children==2);
        assert(bytes::equal(source[nested.end-1..nested.end],b"]"));let usize table=nested.last;assert(m::node(ref tree,table).state==3);
        assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,table,b"")),b"\xc3\xa9"));assert(bytes::equal(m::value(ref tree,m::lookup(ref tree,table,b"quoted")),b"multiline"));
        let usize flat=m::lookup(ref tree,1,b"a.b");let m::Node n=m::node(ref tree,flat);assert(n.start==0 && bytes::equal(source[n.end-6..n.end],b"'flat'"));
    }case .err(_):{assert(false);}};
    switch(document::parse(b"")){case .ok(tree):{assert(m::count(ref tree)==1);}case .err(_):{assert(false);}};
    switch(document::parse(b"''=''")){case .ok(tree):{assert(m::node(ref tree,m::lookup(ref tree,1,b"")).kind==2);}case .err(_):{assert(false);}};
    switch(document::parse(b"a=true\nb={c=false}")){case .ok(tree):{let usize a=m::lookup(ref tree,1,b"a");let usize b=m::lookup(ref tree,1,b"b");let usize c=m::lookup(ref tree,b,b"c");assert(m::node(ref tree,a).kind==5 && bytes::equal(m::value(ref tree,a),b"true") && m::node(ref tree,c).kind==5 && m::node(ref tree,c).state==3 && bytes::equal(m::value(ref tree,c),b"false"));}case .err(_):{assert(false);}};
    switch(document::parse(b"a=1\nb=1979-05-27\nc=1.5\nd=[0xff,07:32:00]\n")){case .ok(tree):{assert(m::node(ref tree,m::lookup(ref tree,1,b"a")).kind==6 && m::node(ref tree,m::lookup(ref tree,1,b"b")).kind==8 && m::node(ref tree,m::lookup(ref tree,1,b"c")).kind==7);let usize d=m::lookup(ref tree,1,b"d");assert(m::node(ref tree,m::node(ref tree,d).first).kind==6 && m::node(ref tree,m::node(ref tree,d).last).kind==8);}case .err(_):{assert(false);}};
}

@test
fn void rejects_invalid_syntax_duplicates_and_closed_table_extensions(){
    rejected(b"a='x'\na='y'",1);rejected(b"a.b='x'\na.b='y'",1);rejected(b"[a]\n[a]",1);
    rejected(b"a='x'\n[a]",1);rejected(b"a=[]\n[[a]]",1);rejected(b"[[a]]\n[a]",1);
    rejected(b"a={b='x'}\na.c='y'",1);rejected(b"a={b={c='x'},b.d='y'}",1);
    rejected(b"a.b='x'\n[a]",1);rejected(b"[a.b]\nx='x'\n[a]\nb.y='y'",1);
    rejected(b"a=['x' 'y']",1);rejected(b"a=[,'x']",1);rejected(b"a=['x',,]",1);rejected(b"a=['x'",1);
    rejected(b"a={b='x' c='y'}",1);rejected(b"a={b='x',,}",1);rejected(b"a={b='x'",1);
    rejected(b"a='x' b='y'",1);rejected(b"a.'b='x'",1);rejected(b"a..b='x'",1);rejected(b"a.='x'",1);
    rejected(b"[]= 'x'",1);rejected(b"[a]trailing",1);rejected(b"[[a]",1);rejected(b"[a] # bad\x01",1);
    rejected(b"a='x'\rb='y'",1);rejected(b"# bad\x7f",1);rejected(b"a=\"x\\q\"",1);rejected(b"\x00",1);rejected(b"a='\xff'",2);
    rejected(b"a=1_",1);rejected(b"a=1979-02-30",1);rejected(b"a=trueX",1);rejected(b"a=False",1);
    var bytes::Buffer large=bytes::empty();assert(bytes::resize_zeroed(ref mut large,1048577).is_ok);rejected(bytes::view(ref large),4);
}
