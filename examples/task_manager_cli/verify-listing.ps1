# Uses the existing driver; fixtures and fixed ID orders are independent of Fairy sorting.
function Run-ListingCases([string]$Store) {
    $titles=@('z','a','a','é',('e'+[char]0x301),'한',' a','A','10','2','aa',('q'*512))
    $state="FYTASK1`t13`n"
    $rows=@{}
    for($i=1;$i -le $titles.Count;$i++) {
        $done=$i%2 -eq 0; $flag=if($done){1}else{0}; $label=if($done){'done'}else{'open'}
        $state+="$i`t$flag`t$($titles[$i-1])`n"
        $rows[$i]=@{done=$done;line="$i`t$label`t$($titles[$i-1])`n"}
    }
    Write-Text $Store $state
    $bytes=$utf8.GetBytes($state)
    $idOrder=@(1..12); $titleOrder=@(7,9,10,8,2,3,11,5,12,1,4,6)
    $specs=@(@{name='default';status='all';order=$idOrder;args=@()},
        @{name='sort-id';status='all';order=$idOrder;args=@('--sort','id')},
        @{name='sort-title';status='all';order=$titleOrder;args=@('--sort','title')},
        @{name='reverse-options';status='done';order=$titleOrder;args=@('--sort','title','--status','done')},
        @{name='repeat-title';status='all';order=$titleOrder;args=@('--sort','title')})
    foreach($status in @('all','open','done')) {
        $specs+=@{name=('status-'+$status);status=$status;order=$idOrder;args=@('--status',$status)}
        $specs+=@{name=('combined-'+$status);status=$status;order=$titleOrder;args=@('--status',$status,'--sort','title')}
    }
    foreach($spec in $specs) {
        $wanted=''
        foreach($id in $spec.order) {
            $row=$rows[$id]
            if($spec.status -eq 'all' -or ($spec.status -eq 'done' -and $row.done) -or ($spec.status -eq 'open' -and !$row.done)) { $wanted+=$row.line }
        }
        Assert-Case ('list-'+$spec.name) (@($Store,'list')+$spec.args) 0 $wanted '' $Store $bytes -Preserve $Store
    }
    $bad=@(
        @{name='missing-status';args=@('--status');message='wrong command argument count; use --help'},
        @{name='missing-second-value';args=@('--status','open','--sort');message='wrong command argument count; use --help'},
        @{name='unknown-option';args=@('--other','id');message='unknown list option; use --status or --sort'},
        @{name='option-case';args=@('--Status','open');message='unknown list option; use --status or --sort'},
        @{name='option-prefix';args=@('--statusx','open');message='unknown list option; use --status or --sort'},
        @{name='bad-status';args=@('--status','closed');message='status must be open, done or all'},
        @{name='status-case';args=@('--status','Open');message='status must be open, done or all'},
        @{name='empty-status';args=@('--status','');message='status must be open, done or all'},
        @{name='bad-sort';args=@('--sort','name');message='sort must be id or title'},
        @{name='empty-sort';args=@('--sort','');message='sort must be id or title'},
        @{name='duplicate-status';args=@('--status','open','--status','done');message='duplicate --status option'},
        @{name='duplicate-sort';args=@('--sort','id','--sort','title');message='duplicate --sort option'},
        @{name='option-overflow';args=@(('--'+'x'*14),'id');message='cannot read list option (maximum 15 UTF-8 bytes)'},
        @{name='value-overflow';args=@('--status',('x'*16));message='cannot read list option (maximum 15 UTF-8 bytes)'},
        @{name='too-many';args=@('--status','all','--sort','id','extra');message='expected store path and command; use --help'}
    )
    foreach($spec in $bad) {
        Assert-Case ('list-'+$spec.name) (@($Store,'list')+$spec.args) 2 '' "task-manager-cli: error: $($spec.message)`n" $Store $bytes -Preserve $Store
    }
    Assert-Case 'mutation-refuses-list-options' @($Store,'add','title','--sort','title') 2 '' "task-manager-cli: error: wrong command argument count; use --help`n" $Store $bytes -Preserve $Store
    $none=$Store+'.none'; Write-Text $none "FYTASK1`t2`n1`t0`topen only`n"
    Assert-Case 'list-no-match' @($none,'list','--status','done','--sort','title') 0 "empty`n" '' $none -Preserve $none
    $empty=$Store+'.empty'; Write-Text $empty "FYTASK1`t1`n"
    Assert-Case 'list-empty-options' @($empty,'list','--sort','title','--status','open') 0 "empty`n" '' $empty -Preserve $empty
    $missing=$Store+'.missing'
    Assert-Case 'list-missing-options' @($missing,'list','--status','done','--sort','title') 0 "empty`n" '' $missing -Absent
    $full=$Store+'.full'; $content="FYTASK1`t257`n"; $expected=''
    for($i=1;$i -le 256;$i++) { $title=(257-$i).ToString('D3'); $content+="$i`t0`t$title`n" }
    for($i=256;$i -ge 1;$i--) { $title=(257-$i).ToString('D3'); $expected+="$i`topen`t$title`n" }
    Write-Text $full $content
    Assert-Case 'list-sort-256' @($full,'list','--sort','title') 0 $expected '' $full -Preserve $full
    # A corrupt done row is still rejected when the request selects only open tasks.
    $corrupt=$Store+'.corrupt'; Write-Text $corrupt "FYTASK1`t3`n1`t0`tvalid`n2`t1`t   `n"
    Assert-Case 'list-hidden-corruption' @($corrupt,'list','--status','open','--sort','title') 1 '' "${corrupt}: error: invalid task title in store`n" $corrupt -Preserve $corrupt
    foreach($suffix in @('.tmp','.bak','.lock')) {
        Write-Text ($Store+$suffix) 'do not change'
        $message=if($suffix -eq '.lock'){"${Store}.lock: error: create lock failed; another run or stale lock (status=1 os_code=80)`n"}else{"${Store}${suffix}: error: recovery sidecar exists; inspect .tmp/.bak before retry`n"}
        Assert-Case ('list-pending-'+$suffix) @($Store,'list','--status','open','--sort','title') 1 '' $message $Store -Preserve @($Store,($Store+$suffix)) -Pending
        Remove-Item -LiteralPath (Local ($Store+$suffix))
    }
}
