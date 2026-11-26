import json,subprocess
from datetime import datetime
rs=[json.loads(l) for l in open('scan/result.jsonl')]
pr={p['number']:p['headRefOid'] for p in json.load(open('merged.json'))}
fail_any=0
for r in rs:
    sha=pr[r['pr']]
    o=subprocess.run(['gh','api','--paginate',f'repos/Navaladi2019/TicketingSystem/commits/{sha}/check-runs?filter=all&per_page=100','--jq','.check_runs[]|[.name,.conclusion,.started_at,.completed_at,.id]|@tsv'],capture_output=True,text=True).stdout
    by={}
    for l in o.strip().splitlines():
        n,c,s,e,i=l.split('\t'); by.setdefault(n,[]).append((e,int(i),c,s))
    for n,xs in by.items():
        xs.sort()
        ex=[x for x in xs if x[2]!='skipped']
        for x in ex:
            if x[2] in ('failure','timed_out','action_required','startup_failure'): fail_any+=1
        if ex and ex[-1][2] not in ('success','neutral'):
            d=(datetime.fromisoformat(ex[-1][0].replace('Z','+00:00'))-datetime.fromisoformat(ex[-1][3].replace('Z','+00:00'))).total_seconds()
            print(r['pr'],n,ex[-1][2],f'{d:.0f}s','newest='+xs[-1][2])
print('executed failures anywhere (incl. later re-run green):',fail_any)
