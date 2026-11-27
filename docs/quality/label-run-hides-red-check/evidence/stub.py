import re,sys
p=sys.argv[1]; L=open(p,encoding='utf-8',newline='').read().split('\n')
stub={'lint','unit','temporal-server','integration','isolation','commit-check-budget','stack','coverage','sdk'}
out=[];job=None;skip=False;injobs=False;done=set()
for l in L:
    if l.rstrip()=='jobs:': injobs=True
    m=re.match(r'^  ([a-z-]+):\s*$',l) if injobs else None
    if m: job=m.group(1); skip=False
    if skip: continue
    out.append(l)
    if injobs and job in stub and l.rstrip()=='    steps:':
        out.append('      - run: exit %d # TICK-1146 spike stub'%(1 if job=='isolation' else 0)); out.append('')
        skip=True; done.add(job)
assert done==stub,stub-done
open(p,'w',encoding='utf-8',newline='').write('\n'.join(out))
