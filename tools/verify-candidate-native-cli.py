"""Verify the Fairy-owned public Windows build/run pipeline with real programs."""
from pathlib import Path
import argparse,subprocess,json,hashlib,ctypes,shutil,time
root=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('--development',type=Path,required=True)
parser.add_argument('--release',type=Path,required=True)
parser.add_argument('--reference',type=Path)
parser.add_argument('--profile',choices=['development','release','both'],default='both')
parser.add_argument('--output',type=Path,required=True)
selected=parser.parse_args()
if not __debug__:parser.error('verification requires Python without -O')
work=selected.output.resolve();work.mkdir(parents=True,exist_ok=True)
profiles=['development','release'] if selected.profile=='both' else [selected.profile]
images={'development':selected.development.resolve(),'release':selected.release.resolve()}
invocation_tag='run-'+str(time.time_ns())
rows=[]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
save=lambda p,v:p.write_text(json.dumps(v,indent=2)+'\n',encoding='utf-8',newline='\n')
buffer=ctypes.create_unicode_buffer(32768);assert ctypes.windll.kernel32.GetTempPathW(32768,buffer)>0
temporary=Path(buffer.value)
def scratch_names():return {p.name for p in temporary.glob('fairyc-native-*')}
def fixture(profile,name,source):
    path=work/'fixtures'/profile/invocation_tag/('한글 '+name)
    (path/'src').mkdir(parents=True,exist_ok=True)
    (path/'fairy.toml').write_text('schema = "fairyc.manifest/v1"\n[package]\nuid = "example.public-native-'+name+'"\nname = "native-app"\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n',encoding='utf-8',newline='\n')
    (path/'src/main.fy').write_text(source,encoding='utf-8',newline='\n')
    return path
def run(argv,cwd,code=0,out=b'',err=b'',stdin=None,label=''):
    before=scratch_names();start=time.monotonic()
    result=subprocess.run(list(map(str,argv)),cwd=cwd,input=stdin,capture_output=True,timeout=180)
    def logged(a):
        a=str(a);data=a.encode('utf-8');return a if len(data)<=256 else dict(bytes=len(data),sha256=hashlib.sha256(data).hexdigest())
    row=dict(label=label,argv=[logged(a) for a in argv],cwd=str(cwd),exit=result.returncode,seconds=time.monotonic()-start,stdoutBytes=len(result.stdout),stdoutSHA256=hashlib.sha256(result.stdout).hexdigest(),stderr=result.stderr.decode(errors='replace'))
    rows.append(row);save(work/'executions.json',rows)
    assert result.returncode==code and result.stdout==out and (result.stderr==err if err is not None else (result.stderr.startswith(b'fairyc:') or result.stderr.startswith(b'main.fy:'))),row
    assert scratch_names()==before,'owned scratch directory leaked'
    print('passed',label,flush=True)
    return result
io_source=r'''module main;
import std::process as process;
import std::io as io;
import std::fs as fs;
import std::bytes as bytes;
@effects(filesystem,console) fn i32 main(){var bytes::Buffer data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,10000).is_ok);var usize i=1;while(i<process::arg_count()){let n=switch(process::arg(i,bytes::view_mut(ref mut data))){case .ok(n):{n}case .err(_):{return 90;}};let ref u8[] view=bytes::view(ref data);assert(io::write_bytes(view[..n]).is_ok);assert(io::write("\n").is_ok);i+=1;}let file=switch(fs::create_new(c"ran.txt")){case .ok(p):{p}case .err(_):{return 91;}};assert(fs::close(file).is_ok);let n=switch(io::read_stdin(bytes::view_mut(ref mut data))){case .ok(n):{n}case .err(_):{return 92;}};let ref u8[] input=bytes::view(ref data);assert(io::write_bytes(input[..n]).is_ok);assert(io::write_stderr("child stderr\n").is_ok);return 134;}
'''
args=['plain','','한글🧚','ends\\','quote"here','--release','x'*8192]
stdin=b'binary\x00stdin\n';expected=''.join(a+'\n' for a in args).encode('utf-8')+stdin
for profile in profiles:
    image=images[profile]
    base=fixture(profile,'simple','module main;fn i32 main(){return 42;}\n')
    run([image,'build','--bad'],base,2,err=b'fairyc: command "build" accepts only optional --release and --locked\n',label=profile+' invalid build option')
    run([image,'run','--locked','--locked'],base,2,err=b'fairyc: command "run" received --locked more than once\n',label=profile+' duplicate run option')
    run([image,'build'],base,out=b'built: build/native-app.exe (x86_64-windows-msvc, development, lock updated)\n',label=profile+' default build')
    product=base/'build/native-app.exe';run([product],base,42,label=profile+' built executable')
    run([image,'build','--release','--locked'],base,out=b'built: build/native-app.exe (x86_64-windows-msvc, release, lock verified)\n',label=profile+' locked release build')
    run([image,'run','--locked'],base,42,label=profile+' ordinary run exit')
    prior=sha(product)
    absent=work/(profile+'-missing-driver');absent.mkdir(exist_ok=True);shutil.copyfile(image,absent/'fairyc.exe')
    run([absent/'fairyc.exe','build','--locked'],base,6,err=None,label=profile+' missing companion')
    assert sha(product)==prior
    (base/'src/main.fy').write_text('module main;fn i32 main(){return 43;}\n',encoding='utf-8',newline='\n')
    run([image,'build','--locked'],base,7,err=None,label=profile+' stale lock')
    assert sha(product)==prior
    alias=fixture(profile,'output-alias','module main;fn i32 main(){return 0;}\n')
    outside=work/'outside'/profile/invocation_tag;outside.mkdir(parents=True,exist_ok=True)
    protected=outside/'native-app.exe';protected.write_bytes(b'prior outside image')
    (alias/'build').symlink_to(outside,target_is_directory=True)
    run([image,'build'],alias,6,err=None,label=profile+' output directory escape')
    assert protected.read_bytes()==b'prior outside image'
    io_project=fixture(profile,'io',io_source)
    run([image,'run','--',*args],io_project,134,out=expected,err=b'child stderr\n',stdin=stdin,label=profile+' argv stdin stdout stderr and reserved ordinary exit')
    assert not (io_project/'ran.txt').exists() and not (io_project/'build').exists()
    if profile=='development' and selected.reference:
        run([selected.reference.resolve(),'run','--locked','--',*args],io_project,134,out=expected,err=b'child stderr\n',stdin=stdin,label='Go reference exact program streams')
    trap=fixture(profile,'trap','module main;fn i32 main(){assert(false);return 0;}\n')
    run([image,'run','--release'],trap,5,err=b'trap: assertion-failed\n',label=profile+' recognized trap')
    negative=fixture(profile,'negative','module main;fn i32 main(){return -1;}\n')
    run([image,'run'],negative,4294967295,label=profile+' full Windows exit bits')
    invalid=fixture(profile,'invalid','module main;fn bool main(){return true;}\n')
    run([image,'build'],invalid,2,err=None,label=profile+' invalid entry')
    missing=fixture(profile,'missing','module main;fn i32 main(){return 0;}\n')
    run([image,'run','--locked'],missing,7,err=None,label=profile+' missing lock')
    oversized=fixture(profile,'output-limit',r'''module main;import std::io as io;@effects(console) fn i32 main(){var u8[4096] data=[65;4096];var usize i=0;while(i<17){assert(io::write_bytes(ref data).is_ok);i+=1;}return 0;}''')
    run([image,'run'],oversized,6,err=None,label=profile+' bounded combined output')
save(work/'execution-receipt.json',dict(schema='fairyc.candidate-native-cli-execution/v1',passed=True,executions=rows,profiles=profiles,images={p:dict(path=str(images[p]),sha256=sha(images[p]),driverSHA256=sha(images[p].parent/'fairyc-candidate-build.exe')) for p in profiles},productGoFrontendInvoked=False,goReferenceInvocations=sum(r['label']=='Go reference exact program streams' for r in rows),arguments=dict(values=args[:-1],lastRepeatedByte='x',lastBytes=8192),stdinSHA256=hashlib.sha256(stdin).hexdigest(),programStdoutSHA256=hashlib.sha256(expected).hexdigest(),scratchDirectoryLeaks=False,verifierSHA256=sha(Path(__file__))))
