"""Test-only reader/executor of emitted scalar MIR, compared with verified Go MIR.

This is an independent observation utility, never a candidate runtime dependency.
The complete product MIR verifier and native input execution are later units.
"""
import re

FIELDS = {
    'T': 'id binding module declaration result signature_start signature_count start end external'.split(),
    'S': 'id type'.split(),
    'F': 'id target entry local_start local_count block_start block_count parameter_start parameter_count start end public'.split(),
    'L': 'id function binding type parameter mutable start end'.split(),
    'B': 'id function first last kind value left right operator type target alternative check callee destination argument_start argument_count trap_mask start end'.split(),
    'O': 'id function block kind result type owner_type operator left right next payload_start payload_end boolean start end literal'.split(),
    'A': 'id value'.split(),
}
TYPES = {1:'i32',2:'i64',3:'u8',4:'u32',5:'usize',6:'bool',7:'void'}
OPS = {1:'+',2:'-',3:'!',4:'~',5:'*',6:'/',7:'%',8:'&',9:'|',10:'^',11:'<<',12:'>>',13:'==',14:'!=',15:'<',16:'<=',17:'>',18:'>='}
TRAPS = {1:'integer-overflow',2:'division-by-zero',4:'signed-division-overflow',8:'invalid-shift',16:'assertion-failed'}

def mask(operator,type,unary=False):
    if unary: return 1 if operator==2 else 0
    if operator in (1,2,5): return 1
    if operator in (6,7): return 6 if type in (1,2) else 2
    return {11:9,12:8}.get(operator,0)

def read_mir(stdout,source):
    lines=stdout.splitlines(); assert lines[:2]==['fairyc.scalar-mir/v1','P\t0']
    tables={tag:{} for tag in FIELDS}
    for line in lines[2:]:
        parts=line.split('\t');tag=parts.pop(0);assert tag in FIELDS and len(parts)==len(FIELDS[tag]),line
        row={key:value if key=='literal' else int(value) for key,value in zip(FIELDS[tag],parts)}
        assert row['id'] not in tables[tag] and all(v>=0 for k,v in row.items() if k!='literal'),row
        if 'start' in row: assert 0<=row['start']<row['end']<=len(source),row
        tables[tag][row['id']]=row
    for tag,table in tables.items():
        base=0 if tag in ('A','S') else 1
        assert sorted(table)==list(range(base,base+len(table))),tag
    return tables

def validate(t,source):
    used_ops=set();used_edges=set();used_blocks=set();target_names={}
    def local(id,function,type=None):
        row=t['L'][id]; assert row['function']==function and row['type'] in range(1,7),row
        if type is not None: assert row['type']==type,row
        return row
    def edge(start,count):
        values=[]
        for i in range(start,start+count): used_edges.add(i);values.append(t['A'][i]['value'])
        return values
    for id,target in t['T'].items():
        name=source[target['start']:target['end']].decode('ascii');assert re.fullmatch(r'[A-Za-z_][A-Za-z_0-9]*',name)
        assert target['external']==0 and target['module']==0 and target['declaration']>0 and target['result'] in range(1,8)
        target_names[id]='main::'+name
    for fid,f in t['F'].items():
        target=t['T'][f['target']]
        params=edge(f['parameter_start'],f['parameter_count'])
        signature=[t['S'][i]['type'] for i in range(target['signature_start'],target['signature_start']+target['signature_count'])]
        assert [local(p,fid)['type'] for p in params]==signature
        assert all(t['L'][p]['parameter']==1 for p in params)
        assert sorted(id for id,l in t['L'].items() if l['function']==fid)==list(range(f['local_start']+1,f['local_start']+f['local_count']+1))
        owned=set(range(f['block_start']+1,f['block_start']+f['block_count']+1));assert f['entry'] in owned
        pending=[f['entry']]; seen=set()
        while pending:
            bid=pending.pop()
            if bid in seen: continue
            seen.add(bid);used_blocks.add(bid);assert bid in owned
            b=t['B'][bid];assert b['function']==fid and b['kind'] in range(1,8)
            op=b['first'];last=0
            while op:
                assert op>last and op not in used_ops
                o=t['O'][op];used_ops.add(op);assert o['block']==bid and o['function']==fid and o['kind'] in range(1,6)
                kind=o['kind'];type=o['type'];owner=o['owner_type'];assert owner in range(1,7)
                if kind!=5: local(o['result'],fid,type)
                else: assert o['result']==0 and op==b['last'] and b['kind']==4 and b['check']==op
                if kind==1:
                    assert o['left']==o['right']==o['operator']==0 and owner==type
                    if type==6: assert o['literal']=='-' and o['boolean'] in (0,1)
                    else: assert re.fullmatch(r'-?(0|[1-9][0-9]*)',o['literal']) and o['payload_end']-o['payload_start']==len(o['literal'])
                else:
                    local(o['left'],fid,owner)
                    if kind in (4,5) and o['right']: local(o['right'],fid,owner)
                    if kind==2: assert owner==type and o['right']==o['operator']==0
                    if kind==3: assert o['operator'] in (1,2,3,4) and o['right']==0
                    if kind==4: assert o['operator'] in OPS and o['right'] and type==(6 if o['operator']>=13 else owner)
                last=op;op=o['next']
            assert last==b['last']
            if b['kind']==1:
                assert b['target']==b['alternative']==0
                if target['result']==7: assert b['value']==0
                else: local(b['value'],fid,target['result'])
            elif b['kind']==6:
                assert b['target']==b['alternative']==0 and b['trap_mask']>0
            elif b['kind']==7:
                assert b['target']==b['alternative']==0
            elif b['kind']==2:
                assert b['alternative']==0 and b['target'] in owned
                pending.append(b['target'])
            elif b['kind']==3:
                local(b['value'],fid,6)
                assert b['type']==6 and b['target'] in owned and b['alternative'] in owned and b['target']!=b['alternative']
                pending.extend([b['target'],b['alternative']])
            elif b['kind']==4:
                o=t['O'][b['check']]
                assert (b['left'],b['right'],b['type'],b['operator'])==(o['left'],o['right'],o['owner_type'],o['operator'])
                expected=16 if b['operator']==0 else mask(b['operator'],b['type'],not b['right'])
                assert b['trap_mask']==expected and expected
                trap=t['B'][b['alternative']];assert trap['kind']==6 and trap['trap_mask']==expected and not trap['first']
                pending.extend([b['target'],b['alternative']])
            else:
                callee=t['T'][b['callee']];args=edge(b['argument_start'],b['argument_count'])
                signature=[t['S'][i]['type'] for i in range(callee['signature_start'],callee['signature_start']+callee['signature_count'])]
                assert [local(i,fid)['type'] for i in args]==signature and b['alternative']==0
                if callee['result']==7: assert b['destination']==0
                else: local(b['destination'],fid,callee['result'])
                pending.append(b['target'])
        assert seen==owned,(fid,owned-seen)
    assert used_ops==set(t['O']) and used_blocks==set(t['B']) and used_edges==set(t['A'])
    return target_names

def limits(type):
    bits={1:32,2:64,3:8,4:32,5:64}[type]
    return (-(1<<(bits-1)),(1<<(bits-1))-1,bits) if type in (1,2) else (0,(1<<bits)-1,bits)

def arithmetic(operator,a,b,type,unary=False):
    if unary:
        if operator==3: return not a
        if operator==1: return a
        if operator==2: return -a
        lo,hi,bits=limits(type);result=~a
        return result if type in (1,2) else result&hi
    if operator==1:return a+b
    if operator==2:return a-b
    if operator==5:return a*b
    if operator in (6,7):
        q=abs(a)//abs(b);q=-q if (a<0)!=(b<0) else q
        return q if operator==6 else a-q*b
    if operator==8:return a&b
    if operator==9:return a|b
    if operator==10:return a^b
    if operator==11:return a<<b
    if operator==12:return a>>b
    return {13:lambda:a==b,14:lambda:a!=b,15:lambda:a<b,16:lambda:a<=b,17:lambda:a> b,18:lambda:a>=b}[operator]()

def check(operator,a,b,type,unary):
    if operator==0:return '' if a else 'assertion-failed'
    lo,hi,bits=limits(type)
    if operator in (6,7):
        if b==0:return 'division-by-zero'
        if type in (1,2) and a==lo and b==-1:return 'signed-division-overflow'
    if operator in (11,12) and (b<0 or b>=bits):return 'invalid-shift'
    value=arithmetic(operator,a,b,type,unary)
    return '' if lo<=value<=hi else 'integer-overflow'

def execute(t,names):
    by_target={f['target']:id for id,f in t['F'].items()};steps=[0]
    def invoke(fid,args,depth=0):
        assert depth<64
        f=t['F'][fid];values={t['A'][f['parameter_start']+i]['value']:a for i,a in enumerate(args)}
        bid=f['entry']
        while True:
            steps[0]+=1;assert steps[0]<10000
            b=t['B'][bid];op=b['first']
            while op:
                o=t['O'][op];kind=o['kind'];type=o['type']
                if kind==1:values[o['result']]=bool(o['boolean']) if type==6 else int(o['literal'])
                elif kind==2:values[o['result']]=values[o['left']]
                elif kind in (3,4):values[o['result']]=arithmetic(o['operator'],values[o['left']],values[o['right']] if o['right'] else 0,o['owner_type'],kind==3)
                else:assert kind==5
                op=o['next']
            if b['kind']==1:return values[b['value']] if b['value'] else None,''
            if b['kind']==2:
                bid=b['target']
            elif b['kind']==3:
                condition=values[b['value']];assert isinstance(condition,bool)
                bid=b['target'] if condition else b['alternative']
            elif b['kind']==4:
                trap=check(b['operator'],values[b['left']],values[b['right']] if b['right'] else 0,b['type'],not b['right'])
                if trap:
                    assert b['trap_mask']&next(bit for bit,name in TRAPS.items() if name==trap)
                    return None,trap
                bid=b['target']
            elif b['kind']==5:
                args=[values[t['A'][b['argument_start']+i]['value']] for i in range(b['argument_count'])]
                value,trap=invoke(by_target[b['callee']],args,depth+1)
                if trap:return None,trap
                if b['destination']:values[b['destination']]=value
                bid=b['target']
            else:raise AssertionError(('unexpected terminal',b))
    entry=next(fid for fid,f in t['F'].items() if names[f['target']]=='main::main')
    value,trap=invoke(entry,[])
    return dict(Kind='trap' if trap else 'return',Value=0 if trap else value,TrapCategory=trap)

def verify_mir(image,reference,directory,rows,run,sha,location):
    counts=dict(fixtures=0,accepted=0,returns=0,traps=0,deferred=0,unchangedFiles=0,differential=0,contractGoldens=0)
    for i,fixture in enumerate(reference['fixtures']):
        source=bytes.fromhex(fixture['inputHex']);path=directory/f'mir-{i}.fy';path.write_bytes(source);before=sha(path)
        row=run([image,'--mir-file',path],rows,'mir-'+fixture['name'])
        row['input']=dict(path=str(path),beforeSHA256=before,afterSHA256=sha(path),bytes=len(source));assert sha(path)==before
        counts['fixtures']+=1;counts['unchangedFiles']+=1
        if not fixture['admitted']:
            assert (row['exit'],row['stdout'],row['stderr'])==(2,'','error: invalid or unsupported MIR input\n'),row
            counts['deferred']+=1;continue
        assert row['exit']==0 and row['stderr']=='',row
        tables=read_mir(row['stdout'],source);names=validate(tables,source)
        # Concrete signatures and source declarations compare to actual Go MIR.
        expected={f['Name']:f for f in fixture['program']['Functions']}
        assert set(names.values())==set(expected)
        for f in tables['F'].values():
            target=tables['T'][f['target']];go=expected[names[f['target']]]
            assert (f['start'],f['end'])==(go['Span']['StartByte'],go['Span']['EndByte'])
            assert TYPES[target['result']]==go['ResultType']
            assert [TYPES[tables['S'][j]['type']] for j in range(target['signature_start'],target['signature_start']+target['signature_count'])]==(go['ParameterTypes'] or [])
        actual=execute(tables,names);expected={k:fixture['outcome'][k] for k in actual}
        assert actual==expected,dict(case=fixture['name'],actual=actual,expected=expected)
        row['mirObservation']=actual
        assert fixture['outcomeOrigin']=='go-mir-interpreter'
        assert not fixture.get('referenceExecutionError') and not fixture.get('referenceOutcome')
        row['outcomeOrigin']='go-mir-interpreter';counts['differential']+=1
        counts['accepted']+=1;counts['traps' if actual['Kind']=='trap' else 'returns']+=1
    return counts
