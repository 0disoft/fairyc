# 중첩 구조체 공유 뷰 구현 결과

2026-10-04, ADR 0441에 따라 중첩 구조체를 공유 뷰와 값 복사 for-in의 원소로
허용했다. 토큰 예제는 다음 형태를 실제로 사용한다.

```fairy
pub struct Span { usize start; usize end; }
pub struct Token { u32 kind; Span position; i64 value; }
```

`src/tokens.fy`의 렉서가 토큰의 바이트 범위 `[start,end)`를 `Span`에 저장하고,
`analyze`는 구조체 for-in에서 중첩 범위를 읽고 검사한다. 출력 형식은 유지했다.

semantic과 MIR의 구조체 판정은 `internal/hir/views.go`의 같은 그래프 검사를
사용한다. 스칼라 또는 비어 있지 않은 구조체 필드만 허용하고, 반복되는 형제
타입은 재사용한다. 순환·누락 정의·빈 중첩 구조체와 참조·포인터·resource·enum·
배열·Option·Result 필드는 모든 깊이에서 거부한다. 명시적 스택을 써서 이 판정
자체가 호스트의 재귀 스택에 의존하지 않는다.

인덱싱과 루프 항목은 중첩 필드를 포함한 원소 전체를 복사한다. 소유자 전체
대여, 명목 타입 구분, 범위 검사와 루프 종료 규칙은 기존 계약을 따른다.
구조체 필드에 구조체 값을 대입하는 기능은 현재 범위에 포함되지 않는다.
복사 독립성 검증은 스칼라 필드를 바꾼 뒤 구조체 전체 값을 대입하는 방식이다.

## 검증

- Windows 일반·릴리스 각각 중첩 구조체 5건, 기존 구조체 반복 7건, 기존 스칼라
  구조체 뷰 8건, 변경된 토큰 분석기 12건: 실제 실행 64회 통과.
- 중첩 금지 필드·순환 및 루프 중 소유자 변경/이동·뷰 필드 쓰기 거부 13건 통과.
  기존 구조체 뷰·반복 거부 검사도 유지했다.
- 중첩 MIR 변조 11건 거부, 스칼라·중첩 반복문 MIR 저장/복원 및 캐시 all-hit 통과.
- HIR 그래프 검사: 10,000단계 체인, 공유 자식 그래프, 순환·금지 필드 검증 통과.
- 관련 HIR·의미 분석·compiler/MIR 검사, packageartifact·semanticsnapshot·atlasmap 및
  Go vet 통과. C 동작은 위 네이티브 실행으로 검증했다.
  `TestMapStoreCrashHelper`는 부모 프로세스에서 의도적으로 건너뛰는 자식 진입점이며,
  제어된 자식 테스트들은 통과했다. mir·mircodec·cbackend·genericsource 패키지는
  초기 이름 필터에 대응하는 unit test가 없어 해당 명령에서는 컴파일만 확인했다.
- 공개 CLI check, 잠금 일반·릴리스 build/run와 portable verify의 native 12회 통과.

첫 native 실행에서 테스트가 미지원인 aggregate 필드 대입을 사용해 한 사례가
실패했다. 제품 소스 변경 없이 테스트를 전체 값 대입으로 고친 뒤 해당 native
묶음 5건을 두 프로필에서 재검증해 모두 통과했다. 최초 실패와 해결 결과를
`docs/ops/evidence/struct-nested-20261004.json`에 구분해 남겼다.
원시 실행·EXE·컴파일러·SDK·소스 해시도 같은 증거 파일에서 연결한다.

예제 디렉터리에서 `python verify.py /절대/경로/fairyc`를 사용한다. 관련 Go 검사:

```sh
bun tools/run-go.ts test ./internal/cli -run '^TestStruct(Nested|VecView|For|Token)' -count=1
```

## 한계

Linux 실제 실행은 사용자의 이후 일괄 검증으로 미뤘다. 전체 저장소 스위트와
성능 벤치마크는 실행하지 않았다. SDK stage0.84, 언어 `/20`, 런타임 `/0`,
문법·MIR·인터페이스 스키마와 타깃 성숙도는 그대로다. SDK/API export나 런타임
심볼, DB, 새 runner·PS1, 설치·CI·push 변경은 없다.
