"""Rebuild semantic trees from Fairy-owned HIR and compare independent Go HIR."""
import re

FIELDS = {
    'F': 'id binding start end scope body parameter_start parameter_count public'.split(),
    'B': 'id kind type start end owner scope module_index declaration signature_start signature_count'.split(),
    'E': 'id kind type operator start end left right alternative block binding edge_start edge_count boolean value'.split(),
    'S': 'id kind start end scope expression target binding operator body initial post loop_target'.split(),
    'K': 'id start end owner scope parent_scope edge_start edge_count tail'.split(),
    'A': ['id','value'], 'T': ['id','type'],
}


def source_tokens(source):
    """Byte coordinates only; semantic kinds/types still come from Go HIR."""
    tokens = []
    i = 0
    while i < len(source):
        if source[i] in b' \t\r\n':
            i += 1
        elif source[i:i+2] == b'//':
            while i < len(source) and source[i] not in b'\r\n':
                i += 1
        elif source[i:i+2] == b'/*':
            i += 2
            depth = 1
            while depth:
                assert i < len(source), 'unclosed source comment'
                if source[i:i+2] == b'/*':
                    depth += 1; i += 2
                elif source[i:i+2] == b'*/':
                    depth -= 1; i += 2
                else:
                    i += 1
        else:
            start = i
            if source[i:i+1] == b'"' or source[i:i+2] in (b'b"', b'c"'):
                if source[i] != 34:
                    i += 1
                i += 1
                while i < len(source) and source[i] != 34:
                    i += 2 if source[i] == 92 else 1
                assert i < len(source), 'unclosed source string'
                i += 1
            else:
                word = re.match(rb'[A-Za-z_][A-Za-z_0-9]*', source[i:])
                i += len(word[0]) if word else 1
            tokens.append((source[start:i], start, i))
    return tokens


def source_functions(source):
    """Map scalar function NAME spans to actual top-level declaration ordinals.

    The module/import declarations count too. Counting F/B rows would miss a
    wrong retained declaration target; source coordinates are independent input.
    """
    tokens = source_tokens(source)
    functions = {}
    ordinal = depth = 0
    current = None
    for i, (text, start, end) in enumerate(tokens):
        if depth == 0 and text in (b'module', b'import', b'fn', b'struct', b'enum', b'resource', b'const'):
            ordinal += 1
            if text == b'fn':
                name, name_start, name_end = tokens[i+2]
                assert re.fullmatch(rb'[A-Za-z_][A-Za-z_0-9]*', name) and tokens[i+3][0] == b'('
                begin = tokens[i-1][1] if i and tokens[i-1][0] == b'pub' else start
                current = (name_start, name_end)
                functions[current] = dict(declaration=ordinal, name=name.decode('ascii'), start=begin)
        if text == b'{':
            depth += 1
        elif text == b'}':
            depth -= 1
            assert depth >= 0
            if depth == 0 and current is not None:
                functions[current]['end'] = end
                current = None
    assert depth == 0 and current is None
    return functions


def byte_span(row, source, header=False):
    start, end = row['start'], row['end']
    assert 0 <= start < end <= len(source), row
    # The candidate parser consumes the first for separator with its initializer;
    # Go's For.Initializer Span ends at the last clause token. No other statement
    # terminator or expression/group span is trimmed.
    if header:
        tokens = source_tokens(source[start:end])
        if tokens and tokens[-1][0] == b';':
            assert len(tokens) > 1
            end = start + tokens[-2][2]
    return start, end


def parse_tables(stdout, source):
    lines = stdout.splitlines()
    assert lines and lines[0] == 'fairyc.scalar-hir/v1'
    assert len(lines) >= 2 and lines[1] == 'P\t0', 'single-module HIR requires module index 0'
    tables = {k:{} for k in FIELDS}
    for line in lines[2:]:
        fields = line.split('\t'); tag = fields.pop(0)
        assert tag in FIELDS and len(fields) == len(FIELDS[tag]), line
        row = dict(zip(FIELDS[tag], fields))
        for key in row:
            if key not in ('kind','type','operator') and not (tag=='E' and key=='value'):
                row[key] = int(row[key])
                assert row[key] >= 0, row
        assert row['id'] not in tables[tag], line
        if 'start' in row:
            byte_span(row, source)
        if tag in ('F', 'E'):
            assert row['public' if tag == 'F' else 'boolean'] in (0, 1), row
        tables[tag][row['id']] = row
    for tag, arena in tables.items():
        base = 0 if tag in ('A','T') else 1
        assert sorted(arena) == list(range(base,base+len(arena))), tag
    return tables


def candidate_tree(t, source):
    visited = {k:set() for k in ('F','B','E','S','K','A','T')}
    active = set()
    scope_owners = {}
    declarations = source_functions(source)
    def enter_scope(scope, owner, parent):
        assert scope > 0 and scope not in scope_owners, ('reused scope', scope)
        scope_owners[scope] = (owner, parent)
    def get(tag,id):
        assert id in t[tag], (tag,id)
        visited[tag].add(id); return t[tag][id]
    def identity(id):
        b=get('B',id)
        assert b['end'] > b['start']
        name=source[b['start']:b['end']].decode('ascii')
        if b['kind']=='function':
            declaration = declarations.get((b['start'], b['end']))
            assert b['owner']==0 and b['scope']==0 and b['module_index']==0 and declaration is not None, b
            assert name == declaration['name'] and b['declaration'] == declaration['declaration'], b
            return ('function',name,b['start'],b['end'],b['module_index'],b['declaration'])
        assert b['kind'] in ('parameter','let','var') and b['owner']!=0, b
        assert b['module_index']==0 and b['declaration']==0, b
        owner=identity(b['owner'])
        return ('parameter' if b['kind']=='parameter' else 'local',owner,name,b['start'],b['end'])
    def edges(start,count):
        return [get('A',i)['value'] for i in range(start,start+count)]
    def expr(id,owner,loops,scopes):
        if not id: return None
        key=('E',id); assert key not in active,key
        active.add(key); e=get('E',id); kind=e['kind']; value={'kind':kind,'type':e['type'],'span':byte_span(e, source)}
        if kind=='integer':
            assert re.fullmatch(r'-?(0|[1-9][0-9]*)',e['value']),e
            value['value']=e['value']
        elif kind=='boolean': value['value']=bool(e['boolean'])
        elif kind=='reference':
            b=get('B',e['binding'])
            assert b['kind'] in ('parameter','let','var') and b['owner']==owner and b['scope'] in scopes, (e,b,scopes)
            assert e['type']==b['type'], (e,b)
            value['target']=identity(e['binding'])
        elif kind=='unary': value.update(operator=e['operator'],child=expr(e['left'],owner,loops,scopes))
        elif kind=='binary': value.update(operator=e['operator'],left=expr(e['left'],owner,loops,scopes),right=expr(e['right'],owner,loops,scopes))
        elif kind=='call':
            binding=get('B',e['binding']); assert binding['kind']=='function'
            value.update(target=identity(e['binding']),arguments=[expr(i,owner,loops,scopes) for i in edges(e['edge_start'],e['edge_count'])])
            assert e['type']==binding['type']
            signature=[get('T',i)['type'] for i in range(binding['signature_start'],binding['signature_start']+binding['signature_count'])]
            assert [x['type'] for x in value['arguments']]==signature, (value,signature)
        elif kind=='block': value['block']=block(e['block'],owner,loops,scopes)
        elif kind=='if':
            assert e['right']==0
            value.update(condition=expr(e['left'],owner,loops,scopes),then=block(e['block'],owner,loops,scopes),otherwise=expr(e['alternative'],owner,loops,scopes))
        else: raise AssertionError(e)
        active.remove(key); return value
    def stmt(id,owner,loops,scopes,header=False):
        if not id: return None
        key=('S',id); assert key not in active,key
        active.add(key); s=get('S',id); kind=s['kind']; value={'kind':kind,'span':byte_span(s,source,header)}
        if kind == 'for':
            enter_scope(s['scope'], owner, scopes[-1])
            scopes = scopes + [s['scope']]
        assert s['scope'] == scopes[-1], (s,scopes)
        if kind=='binding':
            b=get('B',s['binding']); assert b['owner']==owner and b['scope']==s['scope'],(s,b)
            value.update(target=identity(s['binding']),expression=expr(s['expression'],owner,loops,scopes))
        elif kind=='assignment':
            target=expr(s['target'],owner,loops,scopes); assert target['kind']=='reference'
            b=get('B',get('E',s['target'])['binding']); assert b['kind']=='var'
            value.update(operator=s['operator'],target=target,expression=expr(s['expression'],owner,loops,scopes))
        elif kind in ('return','expression','assert','discard','defer'):
            value['expression']=expr(s['expression'],owner,loops,scopes)
            if kind=='defer': assert value['expression']['kind']=='call' and value['expression']['type']=='void'
        elif kind in ('while','for'):
            if kind=='for':
                value.update(initial=stmt(s['initial'],owner,loops,scopes,header=True),post=stmt(s['post'],owner,loops,scopes))
            value.update(condition=expr(s['expression'],owner,loops,scopes),body=block(s['body'],owner,loops+[id],scopes))
        elif kind in ('break','continue'):
            assert loops and s['loop_target']==loops[-1],(s,loops)
        else: raise AssertionError(s)
        active.remove(key); return value
    def block(id,owner,loops,scopes):
        key=('K',id); assert key not in active,key
        active.add(key); b=get('K',id); assert b['owner']==owner and b['parent_scope']==scopes[-1],(b,scopes)
        enter_scope(b['scope'], owner, scopes[-1])
        scopes = scopes + [b['scope']]
        value=dict(span=byte_span(b,source),statements=[stmt(i,owner,loops,scopes) for i in edges(b['edge_start'],b['edge_count'])],tail=expr(b['tail'],owner,loops,scopes))
        active.remove(key); return value
    functions=[]
    for id in sorted(t['F']):
        f=get('F',id); b=get('B',f['binding']); name=identity(f['binding'])[1]
        enter_scope(f['scope'], f['binding'], 0)
        parameters=[get('B',i) for i in edges(f['parameter_start'],f['parameter_count'])]
        assert all(p['kind']=='parameter' and p['owner']==f['binding'] and p['scope']==f['scope'] for p in parameters), f
        signatures=[get('T',i)['type'] for i in range(b['signature_start'],b['signature_start']+b['signature_count'])]
        assert [p['type'] for p in parameters]==signatures,(f,signatures)
        locals_=[r for r in t['B'].values() if r['owner']==f['binding'] and r['kind'] in ('let','var')]
        functions.append(dict(name=name,span=byte_span(f,source),public=bool(f['public']),result=b['type'],
            parameters=[dict(target=identity(p['id']),type=p['type'],mutable=False) for p in parameters],
            locals=sorted([dict(target=identity(p['id']),type=p['type'],mutable=p['kind']=='var') for p in locals_],key=lambda x:x['target']),
            body=block(f['body'],f['binding'],[],[f['scope']])))
    for b in t['B'].values():
        if b['kind'] != 'function':
            assert b['scope'] in scope_owners and scope_owners[b['scope']][0] == b['owner'], b
    # Every retained semantic node and edge must have a real consumer. A callee
    # path/group/minimum magnitude must not remain as an unreachable side table.
    for tag in visited:
        assert visited[tag]==set(t[tag]),(tag,set(t[tag])-visited[tag])
    return functions


def oracle_tree(fixture):
    symbols=fixture['symbols']
    source=bytes.fromhex(fixture['inputHex'])
    declarations=source_functions(source)
    module=fixture['hir']['Modules'][0]
    def span(value):
        assert value['Source']==module['SourcePath'] and value['Package']==0, value
        return byte_span(dict(start=value['StartByte'],end=value['EndByte']),source)
    def function_identity(name, name_span):
        start,end=span(name_span)
        declaration=declarations[(start,end)]
        assert name==declaration['name']
        return ('function',name,start,end,0,declaration['declaration'])
    def identity(id):
        s=symbols[str(id)]; declaration=s['declaration']; kind=s['kind']
        assert s['moduleIndex']==0 and s['module']==module['Path'], s
        if kind=='function': return function_identity(s['name'],declaration)
        owner=function_identity(s['owner'].split('::')[-1],s['ownerDeclaration'])
        return (kind,owner,s['name'],*span(declaration))
    def expr(e):
        if e is None: return None
        kind=e['Kind']; children=e['Children'] or []
        if kind=='group': assert len(children)==1; return expr(children[0])
        value={'kind':kind,'type':e['Type'],'span':span(e['Span'])}
        if kind=='literal':
            value.update(kind='boolean' if e['Type']=='bool' else 'integer',value=e['Value']=='true' if e['Type']=='bool' else e['Value'])
        elif kind=='path': value.update(kind='reference',target=identity(e['Symbol']))
        elif kind=='prefix': value.update(kind='unary',operator=e['Operator'] or e['Value'],child=expr(children[0]))
        elif kind=='binary': value.update(operator=e['Operator'] or e['Value'],left=expr(children[0]),right=expr(children[1]))
        elif kind=='call': value.update(target=identity(e['Symbol']),arguments=[expr(c) for c in children[1:]])
        elif kind=='block': value['block']=block(e['Block'])
        elif kind=='if': value.update(condition=expr(children[0]),then=block(children[1]['Block']),otherwise=expr(children[2]) if len(children)>2 else None)
        else: raise AssertionError(e)
        return value
    def stmt(s):
        if s is None: return None
        kind=s['Kind']; value={'kind':'expression' if kind=='expression-statement' else kind,'span':span(s['Span'])}
        if kind=='binding': value.update(target=identity(s['Binding']),expression=expr(s['Expression']))
        elif kind=='assignment': value.update(operator=s['Operator'].removesuffix('=') if s['Operator']!='=' else '=',target=expr(s['Target']),expression=expr(s['Expression']))
        elif kind=='for':
            f=s['For']; value.update(initial=stmt(f['Initializer']),condition=expr(f['Condition']),post=stmt(f['Post']),body=block(f['Body']))
        elif kind=='while':
            e=s['Expression']; value.update(condition=expr(e['Children'][0]),body=block(e['Children'][1]['Block']))
        elif kind not in ('break','continue'): value['expression']=expr(s['Expression'])
        return value
    def block(b): return dict(span=span(b['Span']),statements=[stmt(s) for s in b['Statements'] or []],tail=expr(b['Tail']))
    result=[]
    assert len(fixture['hir']['Modules'])==1
    for f in fixture['hir']['Modules'][0]['Functions']:
        result.append(dict(name=f['Name'],span=span(f['Span']),public=f['Public'],result=f['ResultType'],
            parameters=[dict(target=identity(p['Symbol']),type=p['Type'],mutable=p['Mutable']) for p in f['Parameters']],
            locals=sorted([dict(target=identity(p['Symbol']),type=p['Type'],mutable=p['Mutable']) for p in f['Locals']],key=lambda x:x['target']),body=block(f['Body'])))
    return result


DIAGNOSTIC_MESSAGES = {
    'type mismatch': {'E1006', 'E1009'},
    'integer literal out of range': {'E1013'},
    'expected bool operand': {'E1007'},
    'assignment target is immutable': {'E1008'},
    'call argument count mismatch': {'E1010'},
    'loop control outside loop': {'E1012'},
    'unresolved name': {'E0101', 'E0103'},
    'duplicate name in scope': {'E0102'},
}
UNSUPPORTED_MESSAGES = {
    'missing return value': ('non-void function without a checked return value',),
    'unsupported for header': ('for initializer outside ', 'for condition with ', 'for post clause outside '),
    'unsupported defer call': ('defer outside an internal void call with copy-scalar arguments',),
    'expression is not callable': ('call target that is not a function',),
}


def compare_hir(fixture, row, location=None):
    """Also usable for raw receipt replay without another executable/build run."""
    source=bytes.fromhex(fixture['inputHex'])
    if fixture['outcome']=='pass':
        assert row['exit']==0 and row['stderr']=='',row
        actual=candidate_tree(parse_tables(row['stdout'],source),source)
        expected=oracle_tree(fixture)
        assert actual==expected,dict(case=fixture['name'],actual=actual,expected=expected)
        return
    assert row['exit']==2 and row['stdout']=='' and fixture['hir'] is None,row
    path=row['input']['path']
    match=re.fullmatch(re.escape(path)+r':(\d+):(\d+): error: ([^\r\n]+) \(bytes (\d+)\.\.(\d+)\)\n',row['stderr'])
    assert match,row
    line,column,message,start,end=match.groups()
    line,column,start,end=map(int,(line,column,start,end))
    assert 0 <= start < end <= len(source),row
    if location is None:
        breaks=list(re.finditer(rb'\r\n|\r|\n',source[:start]))
        actual_location=(len(breaks)+1,start-(breaks[-1].end() if breaks else 0)+1)
    else:
        actual_location=location(source,start)
    assert actual_location==(line,column),row
    matching=[]
    for diagnostic in fixture['diagnostics']:
        if diagnostic['Code'] in DIAGNOSTIC_MESSAGES.get(message,set()):
            s=diagnostic['Primary']
            matching.append((s['StartByte'],s['EndByte']))
    for unsupported in fixture['unsupported']:
        if any(unsupported['Feature'].startswith(feature) for feature in UNSUPPORTED_MESSAGES.get(message,())):
            s=unsupported['Span']
            matching.append((s['StartByte'],s['EndByte']))
        # Resolve rejects an unadmitted named type before Go's later literal
        # range error. Its CLI span is the enclosing declaration, not type use.
        # Accept that documented earlier boundary only for an actual Go type
        # unsupported fact in that exact source declaration.
        if message=='unsupported name-check syntax' and unsupported['Feature'].startswith('type '):
            s=unsupported['Span']
            for fn in source_functions(source).values():
                if fn['start'] <= s['StartByte'] < s['EndByte'] <= fn['end']:
                    matching.append((fn['start'],fn['end']))
    assert (start,end) in matching,dict(case=fixture['name'],message=message,span=(start,end),allowed=matching)


def verify_hir(image,reference,directory,rows,run,sha,location):
    counts=dict(fixtures=0,accepted=0,rejected=0,unchangedFiles=0)
    for index,fixture in enumerate(reference['fixtures']):
        source=bytes.fromhex(fixture['inputHex']); path=directory/f'hir-{index}.fy'; path.write_bytes(source); before=sha(path)
        row=run([image,'--hir-file',path],rows,'hir-'+fixture['name'])
        row['input']=dict(path=str(path),beforeSHA256=before,afterSHA256=sha(path),bytes=len(source))
        assert sha(path)==before,row
        counts['fixtures']+=1; counts['unchangedFiles']+=1
        compare_hir(fixture,row,location)
        if fixture['outcome']=='pass':
            counts['accepted']+=1
        else:
            counts['rejected']+=1
    return counts
