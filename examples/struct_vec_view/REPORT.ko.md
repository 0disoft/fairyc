# 중첩 구조체 스칼라 필드 대입 구현 결과

2026-10-04, ADR 0442에 따라 가변 지역 구조체와 기존 `ref mut` 수신자의 중첩
스칼라 필드를 직접 바꿀 수 있다. 앞선 ADR 0441의 공유 뷰/값 복사 반복도 유지한다.

```fairy
var Token copy = view[0];
copy.position.start = 10;
copy.position.end += 1;
```

`src/tokens.fy`의 `rebase`가 복사한 토큰의 시작/끝 위치에서 구간 원점을 빼고,
`analyze`가 그 지역 위치를 검사한다. 렉서가 저장한 원본 토큰과 출력은 보존된다.
실제 로직은 Fairy C이며 Python은 기존 공개 CLI 검증에만 사용한다.

이름이 있는 가변 루트와 평범한 중첩 구조체의 스칼라 잎만 허용한다. 공유·불변
루트와 살아 있는 루트/필드 대여는 변경을 막는다. 배열 원소·호출 결과·resource
루트, 참조·포인터를 거치는 경로, 구조체 필드 전체 대입과 가변 필드 대여는
이번 확장에 포함되지 않는다. 기존 직접 스칼라 필드/resource와 한-usize 래퍼
대입의 별도 계약은 유지한다.

MIR은 루트와 조상 필드 목록 및 잎 이름을 보존하고 실제 정의·지역 타입을 검증한다.
C는 중첩 lvalue를, 인터프리터는 살아 있는 중첩 셀을 수정한다. 중간 구조체를
복사해 수정 결과를 잃는 하강을 하지 않는다. 복합 대입은 RHS 전에 원래 값을
읽고 RHS를 한 번 실행한다. RHS가 `var ref mut`을 다시 묶어도 선택된 원래 대상에
쓰며, 해당 대여도 최종 쓰기까지 유지한다. 산술 실패는 쓰기 전에 trap이다.

## 검증

- Windows 일반·릴리스: 중첩 대입 11사례씩, 네이티브 실행 22회 통과.
  6종 스칼라, 정수 복합 연산, 복사 독립성, RHS 순서/횟수, 참조 재지정과 지역
  소유자 교체, overflow/underflow·나눗셈·상수 시프트 오류를 확인했다.
- source 거부 12건, 위조 경로/타입 MIR 변조 11건, 인터프리터/codec 저장·복원 및
  중첩 store가 포함된 캐시 all-hit 통과.
- 기존 직접 필드/래퍼/복합 대입과 토큰 예제의 Windows native 회귀 검사 통과.
- 관련 semantic·MIR·interpreter·C backend·codec·packageartifact 검사와 Go vet 통과.
- 공개 CLI check, 잠금 일반·릴리스 build/run 및 portable native 12회 통과.

초기 실행에서 signed 나눗셈 overflow 기대 이름이 기존 계약과 달랐던 테스트를
수정했다. 또한 잘못된 상수 시프트가 MSVC C4293 빌드 오류를 내는 C 코드 생성
결함을 수정했다. 범위 검사는 원래 값을 그대로 검사하고 trap을 유지한다.
검사 뒤 도달 불가능한 C 시프트 식에만 안전한 더미 횟수를 사용한다. 경고 억제나
캐시 우회는 하지 않았다. 최초 실패와 최종 통과를 증거에 구분해 남긴다.

원시 로그와 compiler/SDK/소스/EXE 해시는
`docs/ops/evidence/nested-field-store-20261004.json`에서 연결한다.

## 실행과 한계

```sh
python verify.py /absolute/path/to/fairyc
bun tools/run-go.ts test ./internal/cli -run '^TestNestedFieldStore(Native|Cache)$' -count=1
```

Linux 실제 실행은 이후 일괄 검증으로 미뤘다. Windows 호스트에서
`TestI64HelpersStrictLinuxConsumers`는 Linux/GCC/Clang 필요 조건으로 건너뛰었다.
전체 저장소 스위트와 성능 벤치마크는 실행하지 않았다. SDK stage0.84, 언어 `/20`,
런타임 `/0`, 문법·MIR opcode·wire schema·타깃 성숙도는 유지한다. SDK export나
런타임 심볼, DB·설치·CI·새 runner/PS1·push 변경은 없다.


## ADR 0443: 구조체 필드 전체 대입

`rebase`는 이제 `token.position = Span { ... }`으로 복사한 토큰의 구간을
한 번에 바꾼다. 직접·중첩 필드의 정확한 타입 대입을 허용하며, 원본 벡터와
치환에 사용한 값은 독립적인 복사 값이다. 참조 수신자는 RHS 전에 선택하고
재바인딩 뒤에도 원래 대상에 저장한다. RHS 실패는 최종 치환을 수행하지
않으며, 실패 이전에 RHS가 수행한 부수 효과를 되돌리지는 않는다.

Windows 일반·릴리스에서 복사/평가 순서/참조 선택 성공 프로그램과 런타임
RHS overflow 프로그램을 실행해 4개 결과가 통과했다. 공개 CLI `check`,
일반·릴리스 `build`/`run`과 정상·오류 입력 12회도 통과했다. 소스 거부,
잘못된 MIR, 인터프리터, codec roundtrip, package warm-cache 및 관련
패키지 검사와 Go vet를 확인했다. 증거와 이미지 해시는
`docs/ops/evidence/plain-struct-field-store-20261004.json`에 기록했다.

상수로 확정된 overflow 뒤 구조체를 반환하는 함수는 `check`에 성공하지만
MSVC 릴리스 빌드에서 C4702 도달 불가 코드 경고로 실패하는 한계가 남았다.
이전 컴파일러에서도 필드 대입 없이 같은 문제를 재현했다. 경고를 억제하지
않았으며, 해당 실패는 완료 검증과 구분해 증거에 보존했다.

Linux 실제 실행은 일괄 검증으로 미뤘다. 전체 저장소·성능 검사는 수행하지
않았고, 범위를 넘어선 컴파일러 전체 패키지 검사는 중단한 뒤 관련 검사로
좁혔다. SDK stage0.84, 언어 /20, 런타임 /0 및 문법·wire schema를 유지한다.
새 공개 SDK API, DB·설치·CI·runner·PS1 변경과 push는 없다.


## ADR 0444 후속 수정

이전 절의 상수 overflow RHS 관련 MSVC C4702 실패는 ADR 0444에서 수정했다.
같은 입력 바이트의 공개 CLI `check`와 일반·릴리스 빌드가 통과하고,
실행은 정상적으로 `integer-overflow` trap(종료 코드 5)을 반환한다.
Windows 네이티브 10사례씩, 20회 결과를 확인했다. 상수 증명은 같은 블록의
스칼라 단순 복사로 제한하고, 간접 쓰기·미지 연산·재지정을 만나면 필요한
상수 정보를 폐기한다. 부수 효과 fixture는 trap 이전의 `before` 출력만
남고 이후의 `after` 출력이 없음을 확인한다. 원래 MIR은 바꾸지 않는다.
경고 억제·SDK/런타임/API·DB·CI·runner 변경은 없으며 Linux 실제 실행은
이후 일괄 검증으로 남긴다. 증거는
`docs/ops/evidence/constant-checked-c-20261004.json`이다.


## ADR 0445: 가변 필드 대여 적용

현재 `analyze`는 복사한 지역 토큰의 `local.position`을 `ref mut Span`으로
대여하고 `rebase(position, origin)`에 넘긴다. `rebase`가 시작/끝 필드에
checked 복합 대입을 수행한다. 함수 호출 뒤 자식 대여의 마지막 사용이
끝나면 지역 토큰의 필드를 다시 읽어 검사한다. 원본 공유 벡터와 출력은
유지하며, 제품 계산은 Fairy C 소스가 수행한다.

직접·중첩·그룹 필드 대여는 평범한 구조체 그래프와 이름 있는 가변 지역
루트/기존 `ref mut` 수신자에 한정한다. 자식/별칭/후손 대여가 살아 있는
동안 부모·루트·형제 접근은 E2101이며, 지역 소유자에서 유래한 반환은
E2102이다. 반환 함수와 Option/Result를 통한 자식도 같은 보호를 유지한다.
resource, 배열 원소, 호출 결과, 포인터/역참조 루트, 비평범한 구조체와
동시에 보유하는 여러 가변 sibling 대여는 허용하지 않는다.

공개 CLI `check`, 잠금 일반·릴리스 `build`/`run`, 정상·오류 native 입력
12회가 통과했다. 이번 구현 fixture도 일반·릴리스 성공/overflow trap
4회, 인터프리터/codec 저장·복원, 캐시 cold/warm all-hit, 소스 거부와
위조 MIR 변조 검사를 통과했다. 관련 패키지의 Go vet를 확인했다.
원시 argv/cwd/stdout/stderr/exit와 compiler identity·소스·잠금·EXE 해시는
`docs/ops/evidence/mutable-field-example-20261004.json`, 구현 검증은
`docs/ops/evidence/mutable-field-borrow-20261004.json`에 남겼다.

계약은 ADR 0445와 대여·MIR·C backend·버전 문서에 반영했다. SDK stage0.84,
언어 /20, 런타임 /0 및 문법·wire schema·타깃 성숙도는 유지한다.
Linux 실제 실행은 요청대로 이후 일괄 검증에 남겼다. 전체 저장소 스위트와
성능 벤치마크를 실행하지 않았고 SDK export·DB·CI·설치·runner·PS1 변경과
push는 없다. 구현과 실제 예제 적용을 별도 로컬 커밋으로 나눴다.


## ADR 0446: 가변 구조체 참조에서 공유 필드 대여

현재 `rebase(ref mut Span position, usize origin)`는 시작·끝 필드를 `ref usize`로
공유 대여하고 원래 길이를 읽는다. 마지막 공유 사용 뒤 부모 필드를 보정하고
길이가 유지되는지 검사한다. 원본 공유 벡터와 CLI 출력은 유지한다. 제품 계산과
필드 대여는 Fairy C이며 기존 Python 파일은 공개 CLI 검증만 수행한다.

직접·중첩·그룹 필드 투영은 이름 있는 가변 참조와 ADR 0441의 평범한 구조체
그래프에 한정한다. 살아 있는 공유 자식/별칭/반환값·Option/Result가 있으면
부모·부모 별칭·형제·원본의 쓰기는 E2101이다. 공유 읽기와 여러 공유 형제는
함께 사용할 수 있다. 참조 변수 재지정은 기존 규칙을 유지하며, 먼저 만든
자식은 원래 소유자를 계속 가리키고 보호한다. 지역/다중 원점 반환은 E2102/
E2103이고 resource·비평범한/배열 원소/호출 결과/포인터/역참조 루트와 전체
참조의 암시적 공유 바인딩 전환은 이번 허용 범위에 포함하지 않는다.

관련 semantic/MIR/C emission/interpreter 86개, compiler/codec 및 기존 가변
필드·공유 재대여 회귀 64개가 통과했다. 네이티브 성공과 실행 시 입력에 따른
checked overflow를 일반·릴리스에서 실행해 4개 결과를 확인했고, 캐시 cold/
warm all-hit와 관련 Go vet도 통과했다. 이 예제의 공개 CLI check, 잠금 일반·
릴리스 build/run 및 정상·오류 native 입력 12회가 통과했다. 원시 argv/cwd/
stdout/stderr/exit, 소스·잠금·SDK·compiler identity·EXE 해시는
`docs/ops/evidence/shared-field-example-20261004.json`과 구현 증거
`docs/ops/evidence/shared-mutable-field-borrow-20261004.json`에 기록했다.

초기 상수 overflow 오류 사례는 MSVC C4702/C2220으로 빌드가 실패했다. 새 필드
투영 대신 기존 공유 호출 재대여를 쓴 재현도 이전의 깨끗한 251d15b0 컴파일러에서
같은 릴리스 빌드 실패(종료 6)를 보였다. 기존 C 도달성 한계로 남았으며, 현재
동적 오류 사례의 통과가 이 문제를 해결했다고 주장하지 않는다. 초기/이전
컴파일러 실패와 정상적인 runtime trap 결과를 구분해 보존했다.

ADR 0446과 대여/MIR/C backend/버전 계약을 갱신했다. 기존 공유 재대여와 공유
필드 opcode를 조합하며 verifier/backend 규칙을 약화하지 않는다. SDK stage0.84,
언어 /20, 런타임 /0, 문법·wire schema·타깃 성숙도를 유지한다. Linux 실제 실행은
요청대로 일괄 검증에 남겼으며 전체 저장소 스위트와 성능 벤치마크는 생략했다.
SDK export·DB·CI·설치·새 runner/PS1·push 변경은 없다. 구현과 실제 예제 적용은
별도 로컬 커밋으로 나눴다.
