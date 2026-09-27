# 명목 스칼라 소스 사용성 평가 v0

- 상태: DEFER (실험용 소스 프론트엔드 포함, 영구 기각 아님)
- 일자: 2026-09-22
- 검토 HEAD: `4cb6653fde8e953a7df0521f699da5cacd825d4c`
- 결정 소유: [ADR 0254](../adr/0254-defer-nominal-scalars-pending-user-domain-evidence.md)

## 실제 호출자와 피호출자

피호출자는 `token_info(ref Stream, usize index)`이며 본문에서
`v::at<Token>(ref stream.tokens, index)`로 접근한다 ([scanner.fy](../../examples/lexer_candidate/src/scanner.fy)).
실제 후보 파서의 호출자는 [body_state.fy](../../examples/lexer_candidate/src/body_state.fy)의
`current`이며 `scanner::token_info(stream, cursor.index)`를 호출한다.
같은 파일의 `advance`는 토큰의 끝 바이트 위치 `.end`를 `cursor.previous_end`에 저장한다.
`Cursor.index`와 `Cursor.previous_end`는 모두 `usize` 필드이다.

[기존 실험 코드](../../internal/cli/token_index_experiment_test.go)의 `tokenIndexClient`에 있는
고정값 `2`/`7`은 실제 API를 호출하는 실험용 호출자이며, 실제 `body_state` 호출부와 구분한다.

## 제안 범위와 제외

[기존 명목 제안](nominal-scalar-prototype-v0.md)은 `usize` 기반의 지역 바인딩과 대입,
동일 패키지 내 일반 직접 호출(모듈 간 호출 포함)을 허용한다.
명목 값의 구조체 필드 저장, 명목 타입을 가리키는 `ref`, `Option`/`Result`/`Vec` 내부의
명목 타입, 직접 산술과 복합 대입은 제외한다.

따라서 현 제안만으로 `Cursor.index` 자체를 타입화할 수 없으며, 지원 범위를 넓히는 설계가 필요하다.
호출 지점에서 `wrap(TokenIndex, cursor.index)`로 감싸더라도
`wrap(TokenIndex, cursor.previous_end)`도 허용된다. 생성은 도메인을 지정할 뿐,
원래 정수가 올바른 도메인에서 왔음을 증명하지 않는다.

## 세 가지 방식 비교

아래에서 `wrap`/`unwrap`은 제안서의 설명용 표기이며 구현되거나 채택된 Fairy 문법이 아니다.
구조체 표기는 [기존 구조체 실험](../ops/token-index-struct-experiment.md)의
`TokenIndex { .raw = ... }`와 `.raw`를 따른다. 타입 이름의 모듈 접두사는 비교에서 생략했다.

| 연산 | 현재 `usize` | 일반 구조체 | 명목 제안(설명용) |
| --- | --- | --- | --- |
| 생성 | `let usize index = cursor.index;` | `TokenIndex { .raw = cursor.index }` | `wrap(TokenIndex, cursor.index)` |
| 피호출자 매개변수 | `usize index` | `TokenIndex index` | `TokenIndex index` |
| 호출 | `token_info(stream, index)` | `token_info(stream, index)` | `token_info(stream, index)` |
| 다른 도메인의 변수 전달 | 둘 다 `usize`면 통과 | `ByteOffset`이면 타입 오류 | `ByteOffset`이면 타입 오류가 요구됨 |
| `v::at<Token>`에 넘기는 값 | `index` | `index.raw` | `unwrap(index)` |
| 증가 연산 | 기존 검사 산술 | `.raw`에 검사 산술 후 재포장 또는 필드 대입 | 직접 산술 거부, 풀고 검사 산술 후 재포장 |
| `Cursor.index`에 타입 유지 | 도메인 구분 없음 | `usize` 한 필드 구조체의 선언·재대입·타입 검사 가능 | 현 제안에서 미지원 |

이미 타입이 붙은 지역 변수의 호출 표기는 세 방식에서 같다. 명목 제안 역시 생성·추출을
명시해야 하므로 철자가 짧아진다는 이유만으로 사용성 우위가 입증되지는 않는다.
일반 구조체로 `Cursor.index`를 선언하는 것은 가능하다. 이후 컴파일러가
`usize` 한 필드의 비-`repr(C)` 구조체 값 저장을 지원하면서 이 파서 경로의
구조체 대안도 검사에 통과한다. 생성물과 실행 비용 비교는 아직 완료되지 않았다.
구조체와 명목 타입 모두 명시적인 풀기/재포장을 막지 못하며, 범위나 스트림 소속을 보장하지 않는다.

## 요구되는 진단

`scanner::token_info`가 `TokenIndex`를 받도록 설계된 상태에서, `body_state::current`가
지역 변수 `ByteOffset wrong`을 그대로 넘기는 오류를 대상으로 한다.
진단에는 기대 타입 `TokenIndex`, 실제 타입 `ByteOffset`, 잘못된 인자 범위와 두 타입의
선언 위치가 필요하다. 동명의 타입도 소유 모듈로 구별해야 한다. 새 진단 코드는 예약하지 않는다.
현재 구조체 실험은 잘못된 도메인과 원시 정수 전달을 `E1006`으로 거부한다.
일반 직접 호출의 구조체 불일치는 이제 모듈명이 붙은 기대·실제 타입 이름과 두 구조체 선언 위치를 함께 보여준다. 기본형·참조·컨테이너에는 선언 위치를 추가하지 않는다. 명목 타입 소스 프론트엔드는 여전히 보류하며, 이 진단 개선은 런타임이나 생성 C를 변경하지 않는다.
`wrap(TokenIndex, cursor.previous_end)`처럼 원시 정수를 명시적으로 감싸는 경우는
이 타입 불일치 진단으로 거부할 수 없다는 한계도 구분한다.

## 결정과 재개 조건

이 경계에서는 구조체 대비 명목 타입의 소스 사용성 우위가 확인되지 않았다.
실험용 소스 프론트엔드 구현도 보류한다. 명목 타입을 영구 기각하는 결정은 아니다.
[전체 빌드 비용 실험](../ops/nominal-scanner-total-experiment.md)의 불확실한 결과를
뒤집기 위한 반복 측정은 하지 않는다. 실제 운영 버그를 발견한 것으로 주장하지 않는다.
`TestTokenIndexStructBoundary`와 HIR 프로토타입은 이미 구현돼 있으며 이번에는 재실행하지 않았다.

재개하려면 실제 커서 필드부터 타입을 유지하는 설계가 호출 직전 감싸기보다 어떤 이점을 주는지,
그리고 기존 구조체보다 무엇이 나은지 보여야 한다. 필요한 필드·참조·컨테이너·산술 상호작용만
선별해 지원 또는 거부를 정하고, 소스 진단·타입 식별·비용 검증 범위를 함께 제시해야 한다.
모든 상호작용을 한꺼번에 지원하라는 조건은 아니다.
C 대비 누적 8%는 기능마다 추가 8%를 허용한다는 뜻이 아니며,
과거 Fairy 스칼라 대비 크기 수치가 이 기준의 충족을 증명하지 않는다.

## 실제 레코드 파서 커서 실험 (2026-09-27)

`TestParserCursorStructBoundary`는 체크인된 `records.fy`와 의존 모듈을
임시 워크스페이스에 복사한다. 원본 `Cursor.index: usize`는 검사에 통과한다.
대안은 `TokenIndex { usize raw; }`와 `ByteOffset { usize raw; }`를 선언하고,
`Cursor.index`의 29개 사용처, 두 증감 위치, 두 파서 반환값 대입, 생성 위치를
명시적으로 변환한다. 잘못된 `ByteOffset` 대입은 `E1006`으로 거부되고,
올바른 `TokenIndex` 재대입은 검사에 통과한다.
필드 대입의 `E1006`도 기대·실제 구조체 타입 이름과 두 선언 위치를 보여준다.

이 실험은 실제 파서 커서에 일반 구조체 대안을 적용할 수 있음을 확인한다.
명목 스칼라 구현 필요성을 입증하지는 않는다. 원본 파서 소스와
안정 언어 문법은 변경하지 않았다.

`TestWindowsParserCursorStructFieldStoreNative`는 양쪽 파서를 동일한
`module main; struct R { u32 x; }` 입력으로 실행한다. 두 MSVC 프로필에서
종료 코드 0, 빈 표준 출력·오류, 필드 한 개를 확인했다. 한 번씩 빌드한
산출물 크기는 다음과 같다(바이트).

| 프로필·산출물 | 원본 | 구조체 커서 | 차이 |
| --- | ---: | ---: | ---: |
| 기본 EXE | 349696 | 350720 | +1024 |
| 기본 `unit.obj` | 291915 | 292491 | +576 |
| 기본 `runtime.obj` | 15219 | 15219 | 0 |
| 최적화 EXE | 200704 | 200704 | 0 |
| 최적화 `unit.obj` | 201816 | 201816 | 0 |
| 최적화 `runtime.obj` | 22607 | 22607 | 0 |

이는 동일 Fairy 프로그램 두 형태의 단일 호스트 산출물 비교다. 최적화 EXE와
오브젝트 크기가 같아도 실행 시간이나 생성 명령어가 같다는 뜻은 아니다.
생성 C, 컴파일 시간, 반복 측정, 독립 C 구현 대비 누적 8%는 검증하지 않았다.

## 2026-09-22 검토 범위

사용성 비교와 보류 근거를 기록한 문서 변경이다. 컴파일러·문법·공개 API·언어 및 SDK 버전은
변경하지 않는다. 기존 실험을 재사용하며, 컴파일러 테스트와 벤치마크는 다시 실행하지 않는다.
