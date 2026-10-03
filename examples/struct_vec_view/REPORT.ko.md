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
