import json, subprocess, sys
prs = json.load(open(sys.argv[1]))
out = []
for p in sorted(prs, key=lambda x: x['mergedAt']):
    sha = p['headRefOid']
    r = subprocess.run(['gh','api','--paginate',f'repos/Navaladi2019/TicketingSystem/commits/{sha}/check-runs?filter=all&per_page=100','--jq','.check_runs[] | [.name,.status,.conclusion,.started_at,.completed_at,.id] | @tsv'],capture_output=True,text=True)
    runs=[l.split('\t') for l in r.stdout.strip().splitlines() if l]
    iso=[x for x in runs if x[0]=='isolation']
    iso.sort(key=lambda x:(x[4] or '', x[5]))
    executed=[x for x in iso if x[2] not in ('skipped','') ]
    last_exec = executed[-1][2] if executed else None
    newest = iso[-1][2] if iso else None
    reds=[x for x in executed if x[2]!='success']
    # also any required-name failures hidden (other jobs) for info
    names={}
    for x in runs:
        names.setdefault(x[0],[]).append(x)
    hidden_other=[]
    for n,xs in names.items():
        xs.sort(key=lambda x:(x[4] or '',x[5]))
        ex=[x for x in xs if x[2] not in ('skipped','')]
        if ex and ex[-1][2] not in ('success','neutral') and xs[-1][2]=='skipped':
            hidden_other.append(f"{n}:{ex[-1][2]}")
    labels=','.join(l['name'] for l in p['labels'])
    out.append(dict(pr=p['number'],merged=p['mergedAt'],sha=sha[:8],iso_count=len(iso),iso_exec=len(executed),iso_last_exec=last_exec,iso_newest=newest,iso_reds=len(reds),hidden=hidden_other,labels=labels,err=r.stderr.strip()[:80]))
    print(json.dumps(out[-1]),flush=True)
