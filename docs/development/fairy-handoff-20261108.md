# Fairy C 새 스레드 인계 — 2026-11-08

사용자가 현재 클로저 캐시·의미 실행 배치까지만 마무리하고 새 스레드로
옮기기로 했다. 새 로드맵 기능에는 착수하지 않았다. 저장소는
`projects/hobby/opensource/fairyc`, 작업 브랜치는 `main`이다.

## 새 스레드의 CLI 후속 검증

아래 본문은 인계 당시 상태를 보존한다. 이후 실제 Fairy CLI의 지원된
클로저 값 실행은 [CLI 검증 기록](fairy-closure-value-cli-20261108.md),
소스·패키지 ID 변경과 손상 복구 및 기존 scalar/aggregate 캐시는
[CLI 캐시 검증 기록](fairy-closure-value-cli-cache-20261108.md)을 우선한다.
실제 실행 프로필과 별도 미검증 경계는 각 기록에 명시했다. 기존 비지원
copied-closure 입력의 Go/Fairy 진단·종료 코드 차이와 전체 self-host,
Linux, CI, 배포 검증은 이 후속 검증으로 완료 처리하지 않는다.

## 이번 배치의 범위

- 프로젝트 디스크 캐시: `FYCACHE2` / `FYCKEYS2` / `FYPRJ002`, 현재 소스에서
  유도한 키와 명시적인 aggregate 비트, 실제 cached wrapper의 cold/warm 연결.
- 혼합 단위 캐시: 현재 시그니처와 단위의 safe-fn 타입으로 closure 경로를
  선택하고, 전체 Complete와 인터페이스가 성공한 뒤에만 cold 단위를 게시.
- Complete C 게시: 봉인된 Complete의 closure 플래그에 따라 명시적 어댑터를
  선택. 기존 raw C API의 거부 조건을 완화하지 않음.
- 의미 실행: 실제 Fairy 인터프리터가 생성·복사·인자를 통한 전달·반환·호출을
  실행. 캡처는 불변 힙 셀에 복사하며 원래 스택을 참조하지 않음.

프로젝트 캐시와 Complete C 게시 커밋은 `b1fb3eee`, 의미 실행 커밋은
`a3567a8d`, 혼합 단위 캐시 커밋은 `4ecca976`다. 세 구현 배치는 완료했다.
이 문서와 소스 잠금 갱신은 별도 마무리 커밋에 포함한다.

## 검증 기록

| 범위 | 실제 확인한 결과 |
| --- | --- |
| 프로젝트 캐시 Source | 통과, 23.317초. fixture와 import 소스 검사이며 native 실행과 구별 |
| 프로젝트 캐시 Native | 통과, 518.368초. low 295.01초, optimized 187.53초 |
| 프로젝트 캐시 생성 C | 두 producer에서 성공한 30개 패킷 × 두 consumer = 60회 실행 |
| 혼합 캐시 최종 Source | 통과, 17.02초(test body). 일반·local 입력과 명시적인 private generic package oracle 포함 |
| 혼합 캐시 Native | 통과, 361.87초(test body). low 203.05초, optimized 141.25초; Source와 합친 package 379.000초 |
| 혼합 캐시 실제 실행 | 새 producer 프로세스 30회 중 성공 패킷 26개를 대응하는 consumer 프로파일에서 컴파일·실행 |
| 의미 실행 Source | 통과, 27.150초. 독립 Go 결과 19건과 Fairy fixture 소스 검사 |
| 의미 실행 Native | 통과, 167.002초. low 31.41초, optimized 113.56초; 의미 결과 38건과 제한 검사 8건 |
| 전체 후보 `check` / `check --locked` | 둘 다 통과, 2 instances / 776 sources / 9791 MIR functions |
| `go vet -mod=readonly ./internal/cli` | 최종 Go fixture 수정 뒤 통과 |
| 형식·diff | 대상 Go 파일 `gofmt -l` 출력 없음, `git diff --check` 통과 |

실패 기록도 보존한다. 최초 혼합 Native의 low는 마지막 aggregate 입력에서
저장 수 `0/3/0`으로 실패했다. 일반 구조체만 있는 입력으로 aggregate 단위
저장을 요구한 fixture 문제였으며, 실제 `Cell<i32>` 사용으로 수정했다.
안전 검사나 기대한 hit/miss/store 수를 낮추지 않았다. 새 private aggregate
입력을 stable Go `check`로 검사한 시도도 실패 기록에 남아 있다. 자세한 근거는
[혼합 캐시 기록](fairy-closure-mixed-cache-20261108.md)에 있다.
이후 다른 UNIT 분할 사이의 private C 필드 번호 차이도 분리했다. 기본
독립 Complete의 인터페이스·검증 IR 확인은 유지하고, aggregate C의 정확한
바이트 비교는 같은 분할로 독립 cold-lowering한 Complete를 기준으로 한다.
실패한 원래 비교와 진단 로그도 같은 기록에 남겼다.
최종 실행은 aggregate를 먼저 확인한 뒤 기존 ordinary/local 시나리오를 모두
실행했다. 두 프로파일 모두 성공했고 어떤 검사도 skip하지 않았다.

전체 `go test ./...`, 실제 Fairy CLI closure 값 회귀, native/self-host/배포와
hosted CI는 이번에 실행하지 않았다. 현재 배치만 마무리하라는 사용자 범위와
검증 비용에 따라 후속으로 남긴다. Linux는 사용자의 누적 검증 요청으로 보류했다.
의미 실행 fixture의 직접 힙 셀 변조와 helper 인증의 정확한 step 비용 경계
테스트도 이 배치의 실행 범위에 포함되지 않았다.

## 다음 스레드에서 먼저 할 일

1. **새 closure 값의 실제 Fairy CLI 회귀를 끝낸다.** `check`, locked check,
   `build`, `run`, interpreter/native `test`, JSON/출력 캡처, low/optimized를
   동일한 원본 소스로 비교한다. 이번에 확인한 private Complete 및 캐시
   경로가 CLI 전체의 통과를 대신하지 않는다. 기존 direct-call copied closure
   CLI 검증과 ADR0551의 반환 가능한 값 검증도 구별한다.
2. **CLI의 영속 캐시 회귀를 연결한다.** 새 프로세스의 cold/hit, 소스·provider·
   패키지 정체성 변경, 손상 복구, 실패 소스의 게시 방지를 CLI에서 확인한다.
   기존 scalar/aggregate 경로의 독립 native 회귀도 함께 확인한다.
3. **전체 candidate native/self-host 수용 검사를 마친다.** 지원 프로파일,
   원본·prepared 워크스페이스, 생성 C와 런타임, 재생성/고정점 검증을 묶는다.
   이전 monolithic optimized 빌드의 기본 60초 초과 기록은 여전히 실패다.
   분할된 probe의 통과로 그 기록을 통과 처리하지 않는다.
4. **CI 실행 범위와 시간을 확인한다.** 현재 Windows core는 job 20분,
   Go package test 15분이다. 새 native probe들에 기존 검사를 합치면 이
   예산을 넘을 수 있다. 실제 선택 범위·중복·실행 시간을 확인하되, 근거 없이
   테스트를 건너뛰거나 제한만 늘리지 않는다. 이번 배치에서 workflow와
   runner는 변경하지 않았고 hosted CI도 실행하지 않았다.
5. **Linux 변경분을 한 번에 검증한다.** GCC/Clang × low/optimized로 새
   closure·cache·CLI 경로와 누적 대기 항목을 같은 소스에서 실행한다.
   Fairy GNU companion의 실제 Linux 실행·배포 검증도 이 범위에 포함된다.
   과거 `62f2a006`의 Debian CLI 통과는 완료된 기준선으로 유지한다.
   새 코드의 증거로 재사용하지도, 기존 구현을 미구현으로 되돌리지도 않는다.
6. **배포 후보 ZIP과 clean-machine 설치를 검증한다.** 최신 소스의 Windows
   이미지·SDK·런타임 DLL·PDB 보관/조회·체크섬 및 설치 후 핵심 명령을 확인한다.
   푸시·게시·릴리스·배포는 별도 사용자 지시에 따른다.

CLI 초안은 로컬 ignored 파일
`.cache/closure-handoff-20261108/candidate_closure_value_cli_test.go.draft`에 있다.
이번 배치에서는 실행·커밋하지 않았다. 새 스레드에서 현재 helper와 맞춰 검토한
뒤 정식 테스트로 연결해야 한다. 원격이나 다른 컴퓨터에는 이 초안이 없을 수 있다.

## 설계 확장으로 남은 범위

아래는 지원을 선언한 좁은 구현의 결함 목록이 아니라 별도 확장 작업이다.
이전 순서표에서 Windows 검증이 끝난 ABI·COFF·weak external·CRT·COMDAT·unwind·
debug·DLL 검사를 전부 미구현으로 다시 열지 않는다. 최신 완료 기록이 우선이다.

- 언어: 현재 허용된 범위를 넘는 generics·interfaces·closure 시그니처/저장 형태,
  inline·배열·포인터·resource 형태 확대. ADR0551은 `fn i32(i32)`, i32 캡처
  0~4개, 일반 함수 인자/반환·불변 복사·일반 구조체 필드까지만 허용한다.
  보관형·비동기·cross-thread C callback은 계속 제외한다.
- 백엔드/타깃: C 기준선을 유지한다. textual LLVM IR은 측정할 후보이고 QBE는
  보류 상태다. 새 타깃 선택은 UNDECIDED이며 구현 완료로 표시하지 않는다.
- 원격 패키지: pinned HTTPS와 Windows 온라인/오프라인 캐시는 완료 범위를
  유지한다. 버전 범위 해결, registry, Git/signing/자동 업그레이드와 Linux
  어댑터·전송·CLI 실행은 별도 남은 범위다.
- Atlas: 넓은 소스와 generic map 동등성, 전체 semantic diff, inferred rename,
  여러 파일 구조 편집과 MCP 쓰기. 현재 읽기/영향 탐색·좁은 본문 편집은
  기존 완료 기록에 따른다.
- 상태 실행: generated-C durable persistence와 power-loss 검증, 실제 chain/VM
  선택·어댑터·배포. 기존 Go MIR 기반 로컬 실행·영속 저장은 구현된 범위다.

상세 기준은 [순서표](ordered-work-20261031.md),
[이전 캐시 통합](fairy-closure-cache-integration-20261108.md),
[ADR0551](../adr/0551-copied-closure-values.md) 및 이번 세 실행 기록을 함께 읽는다.
오래된 remaining 문구는 뒤에 기록된 구체적인 완료 증거가 대체한다.

## 작업 환경과 원격 경계

작은 단위로 대상 파일만 커밋한다. Linux는 사용자 요청대로 모아 두었다.
이번 배치에서 공개 CLI/SDK ABI, DB, 의존성, normative 문서, 저장소 위생 설정,
warning/timeout 정책과 CI/runner 설정은 변경하지 않았다. `fairy.lock`은 새
현재 소스에 대해 정상 `check`로 갱신했고 locked check로 확인했다.
푸시·workflow dispatch·릴리스·배포는 실행하지 않았다.

현재 배치의 모든 native 검사는 종료했다. 새 스레드에서는 `git status --short`,
`git log -4 --oneline`과 이 문서부터 확인하고 첫 CLI 회귀 항목부터 이어가면 된다.
