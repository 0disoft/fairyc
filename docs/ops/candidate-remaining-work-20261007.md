# Fairy 컴파일러 전환: 후속 작업 목록

2026-10-07 현재 소스와 진행표를 기준으로 정리했다. 14번의 구조체 필드와
배열 실행, 15번의 resource 타입·필드 소유권과 16번의 factory 생성, 17번의 이동 HIR과
18번의 이동 후 사용 검사, 19번의 분기 합류와 20번의 반복·재진입 검사까지 완료했다.
승인된 1~20번은 완료했고, 번호 21~127은 107개 후속 작업이며,
항목마다 별도 커밋이나 전체 테스트를 요구하는 목록은 아니다.
의존성이 맞는 작은 구현 단위로 묶어 순차 커밋한다.
Linux 검증과 빌드 비용 측정은 요청대로 뒤에 묶어서 수행한다.

21~40번은 순차 구현·검증·로컬 커밋을 완료했다. COFF 섹션 상한 수정(ADR0473)을
선행 커밋으로 분리했고, 33~39번의 미뤘던 릴리스 검증은 40번에서 묶어 완료했다.

21~40번의 원래 요구사항·현재 담당 소스·단위 커밋·실행 결과·바이너리 해시 대조는
`evidence/candidate-units-21-40-completion-audit-20261007.json`에 기록했다. 24번의
과거 실패 harness 기록도 최종 후보로 정상 사례 6개·native 12회를 다시 확인해 보완했다.

근거는 candidate-tagged-ownership-progress.md, examples/lexer_candidate/README.md와
실제 Fairy 후보 소스, 기존 Go semantic/MIR/package/CLI 구현,
docs/architecture/self-hosting-readiness-v0.md다.

Go에 이미 있는 기능은 Fairy 후보에 이식·통합할 작업이라는 뜻이다.
후보에 이미 있는 스칼라 배열, 정적 문자열·바이트 리터럴, 바이트 뷰,
스칼라 구조체, 정수·bool enum payload, 스칼라 Option/Result 실행을
처음부터 다시 만들지는 않는다. 파싱과 타입 검사·실행 지원은 구분한다.
마지막 선택 기능은 셀프호스팅 필수 조건에 포함하지 않는다.

## A. 승인된 1~20번 완료

12~13번 중첩 및 구조체·진단 Option/Result는 별도 진행표에서 완료 증거를 관리한다.
현재 nominal leaf는 14비트로 표현하며, 더 큰 합성 타입은 기존 30비트 상한을 유지한다.

14. 완료: 구조체 필드·배열의 enum/Option/Result 저장·복사·인덱싱·대입·초기화와 배열 MIR/native 실행. 상세 증거는 별도 진행표에서 관리.
15. 완료: 선언별 resource 타입 정체성과 필드 소유권·표현 정보 연결. 자원 프로그램의 MIR·native 실행은 아직 거부하며 이동·cleanup 단위에서 연결.
16. 완료: 같은 모듈의 명시적 resource factory와 생성 HIR 연결. 초기화 순서·정확한 필드 타입·중첩 소유권을 보존하며 자원 MIR/native는 아직 거부. 모듈 간 representation 보호는 61~64번에서 연결.
17. 완료: 바인딩·대입·인자·반환의 전체 값 이동 HIR. 소유자 binding·위치를 보존하고 부분 필드 이동을 거부. 실제 수명 상태 검사는 18~20번에서 연결.
18. 완료: 이동 후 사용 E1101과 이동·사용 위치 진단. 새 대입·shadow·도달 불가 사용을 구분. Go의 ? 자원 소비 누락도 재현 후 수정.
19. 완료: 분기 합류에서 Live/Dead 불일치를 E1104로 거부. 종료 분기·동일 이동 상태·새 값 복원을 구분하고 이동 위치 보존. `&&`/`||`의 미평가 경로를 Go 기준 구현에도 반영. 증거는 `evidence/candidate-resource-branches-20261007.json`.
20. 완료: 반복 진입·재진입·break·continue에서 이동 상태 분석. 본문 자원·새 값 복원·중첩 타깃·종료 경로를 구분하고 E1105의 loop-header/move 위치 보존. 조건 재평가의 자원 재사용을 E1101로 거부하며 Go의 누락도 수정. 증거는 `evidence/candidate-resource-loops-20261007.json`.

14번의 실행 완료는 `evidence/candidate-tagged-array-native-20261007.json`에 기록했다.
MIR 읽기·쓰기·검증은 소유 원소 레코드를 사용하며 C 레이아웃은 원소 → 배열 → 포함 구조체
순서로 정의한다. 완전히 추론한 복합 2차원 배열과 1024개 초과 타입 노드도 실제 실행했다.
고정 Go 기준 도구의 태그 배열 미지원은 별도 한계로 기록하며 완료 일치로 계산하지 않는다.
resource 배열은 14번에서 미리 허용하지 않고 이동·정리 계약을 연결한 뒤 처리한다.

15번은 `evidence/candidate-resource-types-20261007.json`에 기록했다. 정상 11개·오류
21개 사례의 두 프로필 CLI 관찰 256개, 프로필별 compiler API native test 5개,
Go 기준 비교 27개와 기존 배열 native 실행 10개가 있다. 자원 순환 레이아웃은
후보가 거부하지만 Go 의미 검사는 허용하는 차이가 있으며, Go의 resource 고정 배열
필드 미지원도 별도로 남겼다. 이는 자원 프로그램 자체의 실행 완료를 뜻하지 않는다.

11번의 비교 차이는 Go의 ? 오류 반환 검사 누락을 수정해 해소했다.
evidence/candidate-result-handling-alignment-20261007.json에 새 검사·도구 해시와
8개 기준 사례의 양쪽 후보 프로필 일치 결과가 있다. 고정 bootstrap EXE는 보존했다.

## B. 이동 검사 이후 resource 수명 관리

21. 완료: cleanup 본문을 void 문맥에서 이름·타입·지역 자원/Result 흐름 검사. 컴파일러 소유 `this` 수신자는 필드 읽기·직접 스칼라 필드 수정만 허용하며 전체 이동·덮어쓰기는 E1102, 부분 자원 필드 이동은 E1103으로 거부. 자원 receiver의 중첩 필드 쓰기는 Go 대조에서 확인한 기존 금지를 유지. `?`와 값 반환은 거부. 비어 있지 않은 cleanup의 HIR/MIR/C 연결은 22번에서 처리하며 그 전에는 명시적으로 실패. 증거는 `evidence/candidate-resource-cleanup-20261007.json`.
22. 완료: 정상 fallthrough에서 live root를 선언 역순으로 한 번씩 정리. 이동 donor·shadow·소비 인자·빈/비어 있지 않은 정리·직접 scalar receiver 수정·i64 필드를 실행. owned HIR/MIR 연결 및 위조 정리 검사는 별도 부분 receipt에 보존. `evidence/candidate-resource-exit-20261007.json`의 12개 소스 사례, CLI 관찰 81개, 생성 C native 실행 16회와 프로필별 API 테스트 3개가 근거. return/?, 반복 종료, 덮어쓰기, 중첩 소유 필드는 후속 번호에서 연결.
23. 완료: 직접·중첩 return, 반환값 고정, 자원 factory/호출/if/switch 반환과 scalar Result ? 성공·오류 경로의 정리 연결. 오류 반환을 생성한 뒤 성공 경로의 소유 상태를 복원하며 short-circuit의 종료 경로가 미평가 경로 상태를 훼손하지 않음. `evidence/candidate-resource-return-20261007.json`: 13개 사례, CLI 93개, 생성 C native 20회, 프로필별 API 테스트 1개(그래프 3개). 자원 보유 tagged payload는 27번에서 연결.
24. 완료: 반복 본문 fallthrough와 nearest-loop break/continue에서 실제로 빠져나가는 중첩 본문만 정리. 바깥 root 보존, for post 이전 정리, 반복 재생성과 return을 dev/release native 실행. 근거는 `evidence/candidate-resource-loop-exits-20261007.json`, 최종 후보 재실행은 `evidence/candidate-resource-loop-exits-final-20261007.json`.
25. 완료: RHS를 먼저 고정한 뒤 live 이전 세대만 정리하고 새 donor를 설치. 자기 이동·소비 후 복원·factory 호출과 scalar Result ? 실패/성공을 구분하며 MaybeLive는 기존 Flow 검사로 거부. resource-bearing Result의 직접 make()? 성공·오류 실행도 27번에서 검증했다. 근거: `evidence/candidate-resource-overwrite-20261007.json`.
26. 완료: 중첩 resource의 outer cleanup → 필드 역순 정리와 부분 초기화의 완료 임시값만 역순 정리. 이후 필드/인자의 ? 실패, factory 반환 필드, 첫 필드 실패와 donor 이동을 구분. 실패한 바깥 glue는 호출하지 않음. 4096 local/function 상한을 두며 tagged 소유는 27번에서 연결. 근거: `evidence/candidate-resource-partial-20261007.json`.
27. 완료: resource 필드 및 Option/Result의 전체 이동, active payload 추출·무시·중첩 정리와 ? 반환을 실행. 임시 자원도 실제 lexical scope 종료와 source 순서로 정리하며 wildcard arm의 내부 root 및 continue를 구분. `old=make()?`의 양쪽 결과를 직접 검증. 부분 필드 이동과 resource 고정 배열은 기존 제한 유지. 근거: `evidence/candidate-resource-tagged-20261007.json`.
28. 완료: 닫힌 drop/store/key 선언 검사, 재귀 no-drop 정리 금지(E1107/E1108), 같은 소스 모듈의 bare local/parameter burn(E1109), 별도 MIR op 27과 cleanup 없는 C 실행. 18개 정상·오류 fixture 및 두 C 프로필 native 12회. 모듈 import의 선언 소유권 연결은 61~64번에서 처리하며 그 전 외부 resource 입력은 거부. 근거: `evidence/candidate-resource-abilities-20261007.json`.
29. 완료: scope/return/?/break/continue를 공통 종료 정책으로 통합. 값·인자를 먼저 고정하고 종료하는 defer suffix를 역순 실행한 뒤 resource 정리. cleanup 안의 defer와 즉시 인자 캡처를 실행하며 자원/집합/값반환 defer는 거부. 11개 fixture, native 16회. 근거: `evidence/candidate-resource-defer-20261007.json`.
30. 완료: 기존 비-unwind 생성 코드를 유지하고 assertion/overflow/0 나눗셈/부분 생성/defer 내부/cleanup 내부/no-drop trap을 실제 실행. trap 블록은 cleanup 호출과 successor가 없으며 이미 실행한 효과만 남음. 7개 fixture, native 14회. 근거: `evidence/candidate-resource-traps-20261007.json`.

## C. 자기 소스에 필요한 borrow·뷰 확장

31. 완료: 스칼라·구조체·resource의 명시적 참조와 역참조, 스칼라 필드 읽기 및 함수 인자 실행. high-half 타입 ID와 MIR 주소/읽기 검증을 연결하고 지역·값 인자 반환, 대여 중 수정·이동을 거부. 14개 fixture, native 14회. 근거: `evidence/candidate-resource-references-20261007.json`. 필드 주소·참조 수정은 다음 단위에서 연결.
32. 완료: 중첩 구조체 필드·이름 있는 배열 원소 및 원소 필드의 공유 주소를 실제 저장 위치에서 생성. whole-owner origin과 검사한 인덱스를 유지하며 지역 escape·대여 중 수정·임시 root를 거부. 10개 fixture, native 12회와 주소 HIR/MIR API test. 필드 배열·다차원 원소의 주소는 현재 명시적 거부. 근거: `evidence/candidate-resource-projections-20261007.json`.
33. 완료: 스칼라 역참조 대입과 plain 필드·배열 원소의 가변 주소, 참조 경유 필드 수정을 연결. 저장 위치를 RHS 이전에 고정하며 immutable/shared 수정·sibling 대여·resource 필드 가변 주소·역참조 복합 대입은 거부. 후보 릴리스 재빌드와 33~39 회귀를 40번에서 묶어 확인 완료. 11개 fixture, 생성 C native 12회. 근거: `evidence/candidate-resource-reference-stores-20261007.json`.
34. 완료: 정확한 referent의 mutable→shared 호출 재대여와 반환된 공유 자식·alias의 whole-owner hold를 연결. 공유 읽기는 공존하고 부모 수정·가변 호출은 자식 마지막 사용까지 금지하며 이후 재개. 직접 바인딩의 암시적 변환은 거부. 7개 fixture, 생성 C native 8회. 근거: `evidence/candidate-resource-shared-reborrows-20261007.json`.
35. 완료: 명시적 ref mut *parent와 반환된 mutable 자식·후손의 세대를 연결. 부모와 부모 alias의 읽기·수정은 자식 마지막 사용까지 금지하고 이후 재개하며 shared→mutable 확대를 거부. 8개 fixture, 생성 C native 8회. 근거: `evidence/candidate-resource-mutable-reborrows-20261007.json`.
36. 완료: 인자와 참조·필드·역참조 consumer의 평가 종료까지 loan을 유지. 앞쪽 shared/mutable 인자와 뒤쪽 경쟁 접근을 양쪽 순서로 거부하고, 앞쪽에서 복사한 scalar 읽기는 이후 수정과 공존. 반환된 중첩 호출 인자도 hold를 유지. 9개 fixture, 생성 C native 8회. 근거: `evidence/candidate-resource-argument-loans-20261007.json`.
37. 완료: 반환 입력 origin을 호출자의 정확한 실제 인자로 치환하며 forward·재배치·wrapper·switch·뷰 반환을 연결. 미선택 입력의 loan은 호출 후 종료하고 선택된 입력 수정·지역 escape·두 입력 반환은 거부. CLI E2103을 명시. 7개 fixture, 생성 C native 8회. 근거: `evidence/candidate-resource-return-origins-20261007.json`.
38. 완료: Unknown과 Static을 분리하고 단일 입력 origin의 직접·상호 재귀 요약을 bounded 고정점으로 증명. Static의 입력 origin 단축은 허용하고 근거 없는 cycle·입력 교환 재귀·지역 escape는 거부. 6개 fixture, 생성 C native 6회. 근거: `evidence/candidate-resource-recursive-origins-20261007.json`.
39. 완료: Option/Result의 단일 safe-borrow leaf를 정체성·레이아웃·C payload·origin·generation에 연결. pattern 캡처와 ? 성공·암시적 오류 반환, mutable none을 검증. 명명 집합/배열에 참조 저장과 두 borrow leaf는 거부하며 30bit 표현 상한은 유지. 12개 fixture, 생성 C native 14회. 근거: `evidence/candidate-resource-tagged-borrows-20261007.json`.
40. 완료: 도달 가능한 분기 arm·조기 종료·fallthrough/break/continue의 대여 상태를 합류. 서로 다른 분기에서 만든 동일 owner의 가변 참조·재대여를 bounded DAG로 함께 보존하며 부모 사용은 모든 자식 종료 뒤 재개. if/switch·중첩 분기·반복·Option none·함수 반환, join 뒤 owner/parent 수정 거부와 dead return/use/continue 뒤 dead break를 검증. 최종 30개 fixture·두 후보 프로필·생성 C native 40회, 33~39 회귀 60개 fixture·native 64회. 근거: `evidence/candidate-resource-borrow-flow-joins-20261007.json`, `evidence/candidate-resource-borrow-joins-regression-20261007.json`. 기존 immutable borrow descriptor·whole-owner 독점 모델을 유지하며 predicate 상관과 가변 descriptor binding 재대입은 기존 제한으로 남김.
41. 부분 완료: str::from_raw_parts·byte_slice::from_raw_parts·from_raw_parts_mut와 CStr::from_raw_parts를 unsafe 검사, owner 기반 HIR/MIR origin·대여와 C descriptor 생성에 연결. CStr 리터럴은 끝 NUL을 포함한 저장 길이를 보존하며 bare/mutable CStr·metadata/index·mutable pointer cast를 거부. 모듈 전달·Option payload와 위조 NUL/unsafe/owner/길이 검사를 일반·릴리스 후보 및 native 실행에서 확인. 근거: evidence/candidate-raw-views-20261008.json, evidence/candidate-raw-view-modules-20261008.json, evidence/candidate-cstr-native-20261008.json, evidence/candidate-cstr-modules-20261008.json, evidence/candidate-cstr-api-20261008.json. 인증된 SDK 본문의 실행 연결은 후속 단위.
42. 완료: 저장된 구조체·중첩 구조체의 rank-one u8 배열 필드 뷰를 원래 저장 공간에서 생성. 공유 구조체 인자에서 반환하는 뷰도 origin을 보존. 임시 owner는 기존 계약에 따라 거부. 일반·릴리스 후보와 생성 C 두 프로필, 정상 5개·오류 7개를 검증. 근거: `evidence/candidate-resource-field-views-20261007.json`.
43. 완료: ADR0415 제한을 유지하고 disjoint sibling·copied parent·빈 child·grandchild·owner 경쟁 접근을 거부. 마지막 사용 후 parent 재개, 끝의 빈 범위, 경계 trap, bounds 평가 순서를 두 후보·C 프로필로 검증. 정상 5개·오류 6개·native 10회. 근거: `evidence/candidate-resource-subview-limit-20261007.json`.

## D. 제네릭·SDK·컴파일러가 사용하는 언어 기능

44. 완료: 같은 소스 모듈의 명시적 scalar 함수 제네릭을 별도 AST arena에서 치환한 뒤 일반 typing/HIR/MIR로 연결. 순서 있는 타입 인자·forwarding·본문·분기·반복·스칼라 정체성 및 오류 20개 fixture, native 20회 검증. public import와 SDK nominal 예외는 46/61~64에서 연결. 근거: `evidence/candidate-resource-scalar-generics-20261007.json`.
45. 완료: build-local 원 선언 ID + 순서 있는 scalar 타입 조합으로 중복 제거. 반복 소스 호출은 그대로 실행. 재귀·depth 32·선언당 16개·전체 64개·분석량 1048576 및 AST arena 상한을 유지. 독립 HIR API 검사 2개, planner 정상/오류 4개·native 4회. 패키지 정체성 연결은 63번. 근거: `evidence/candidate-resource-generic-plan-20261007.json`.
46. 부분 완료: Fairy SHA-256과 bundled SDK 모듈별 원본 digest로 std namespace를 인증. 모듈 타입 그래프의 Vec 원소·공급자·별칭 정체성 및 6필드 storage 형식을 연결. 변조·SDK 사칭·bare Vec·직접/중첩 borrow 원소·위조 공급자를 거부하며 실패한 타입 interning을 되돌림. 근거: `evidence/candidate-sdk-authority-20261008.json`. 스칼라 template의 AST·owned HIR/MIR/C 연결은 아래 후속 근거에 기록했다. 자원 원소는 48번에서 이어간다.

46번의 구체화 요청은 `sdk_vector_plan`이 원본 catalog·canonical graph와 함께 소유하며 모듈 시그니처에 연결했다. 별칭 공유·모듈별 원소 구분·중첩 Vec·파서 owner 종료 뒤 조회와 누락/추가/순서/원소/공급자 변조 거부를 dev/release native API에서 확인했다. 요청 상한은 64개다. 근거: `evidence/candidate-sdk-vector-plan-20261008.json`, `evidence/candidate-sdk-vector-plan-signatures-20261008.json`. 스칼라 SDK AST·원소 레이아웃·HIR/MIR/C 연결은 아래 후속 단위에서 검증했다. 자원 원소와 대여 접근은 남는다.

44/45/61~64번의 imported scalar generic 함수는 provider 선언과 순서 있는 타입 인자로
구체화하며 같은 공급자의 별칭 호출을 중복 제거한다. 같은 이름의 caller/provider 함수와
provider-local helper 및 nominal 인자는 원래 선언 정체성을 보존한다. 일반 API 3개와
릴리스 대표 API 1개, CLI 정상 2개·오류 4개 및 C native 4회가 근거다:
`evidence/candidate-module-generics-api-20261008.json`,
`evidence/candidate-module-generics-cli-20261008.json`. Vec/aggregate template와
영속 package 정체성은 아직 남는다.
46번의 스칼라 Vec AST 구체화까지 완료: 별칭은 하나의 resource 인스턴스를 공유하며 6필드와 비어 있지 않은 cleanup 본문을 함께 복제한다. 원본 SDK 바이트·공급자 위치를 보존하고, 관계없는 함수의 동명 매개변수 오탐을 수정했다. dev/release native API 근거: `evidence/candidate-sdk-vector-expand-20261008.json`. storage intrinsic의 타입 검사·owned HIR/MIR/C와 실제 Vec 실행은 아래 후속 근거에 기록했다.

47. 완료: 스칼라 Vec 할당·초기화된 prefix·push/pop·reserve/clear·cleanup과 `at/at_mut` 원소 대여 실행. 여섯 스칼라 타입의 공유·가변 뷰와 이름 있는 정수 뷰도 연결했다.
47번의 선행 연결로 Vec의 원본 extern 선언 6개와 정확한 linker/result/parameter 권한을 owned HIR/MIR 검사에 연결했다. cleanup의 선행 호출을 extern 선언으로 인정하지 않으며, dev/release native API에서 C prototype 6개와 권한·소스·시그니처·효과·선언 위치 변조 거부를 확인했다. 근거: `evidence/candidate-sdk-vector-foreign-20261008.json`, `evidence/candidate-sdk-vector-process-regression-20261008.json`. 스칼라 Vec의 storage 연산과 실제 실행은 아래 후속 근거에서 확인했다.
47번 스칼라 실행 근거: dev/release 후보가 정상 프로그램 4개와 거부 사례 7개를 처리했고, 생성 C를 MSVC low/optimized에서 16회 실행했다. 스칼라 6타입, 첫 할당·growth 실패 후 donor/값/길이/용량 보존, 역순 prefix cleanup과 누수 0개를 확인했다. 프로필별 native API는 storage MIR 변조·SDK 변조와 출력 보존을 확인했다. 템플릿 제거 뒤 필드 ID를 다시 연결하고 concrete resource별 cleanup receiver를 구분했다. 근거: `evidence/candidate-sdk-vector-storage-api-20261008.json`, `evidence/candidate-sdk-vector-storage-cli-20261008.json`.

47번 스칼라 원소 대여: 인증된 SDK `at/at_mut` 본문을 AST 구체화 → owned HIR/MIR → C 실행에 연결했다. raw slot과 별개로 Vec 소유자 origin을 보존하며, 원소 크기로 주소를 계산하기 전에 SDK의 초기화 prefix 경계를 검사한다. 공유·가변 원소 대여, 호출로 반환한 alias와 child 재대여, 마지막 사용 뒤 owner 접근 재개, 대여 중 구조 변경·덮어쓰기·이동 금지를 연결했다. 특수화의 같은 원본 source span 때문에 다른 함수의 반환 origin이 섞이던 문제는 함수별 scope로 구분했다. 근거: `evidence/candidate-sdk-vector-element-api-20261008.json`, `evidence/candidate-sdk-vector-element-cli-20261008.json`, `evidence/candidate-sdk-vector-element-storage-regression-20261008.json`. 비바이트 Vec 뷰와 resource/nominal 원소는 남음.

47번 검증은 dev/release 후보 빌드, 정상 4개·경계 trap 3개·거부 16개와 C native 28회, 이전 저장 기능 native 16회가 완료됐다. 개발 native API 테스트 2개도 통과했다. 릴리스 API 파일 묶음은 C 최적화 중 관찰자 360초 제한으로 자식 프로세스까지 종료했으며 완료로 계산하지 않는다. 관련 변조 검사의 릴리스 실행은 후속 묶음에서 확인한다.

47번 바이트 뷰: 원본 SDK `view<u8>/view_mut<u8>`와 요청된 `view_u8/view_mut_u8`를 연결했다. initialized prefix의 원소 복사·할당 없이 owner/data/length를 순서대로 평가하고 owned HIR/MIR에 보존한다. MIR의 길이 edge 소유·타입·초기화를 검사하며, 빈 Vec의 하위 뷰에서는 null 포인터 덧셈을 피한다. 공유 sibling·반환 alias·중첩/가변 하위 뷰·부모 접근 재개·경계 trap을 실행하고, 대여 중 구조 변경·덮어쓰기·이동·인자/범위 평가의 변경과 local escape를 거부한다. 근거: `evidence/candidate-sdk-vector-byte-view-api-20261008.json`, `evidence/candidate-sdk-vector-byte-view-cli-20261008.json`. 이전 원소 대여·저장 실행은 `evidence/candidate-sdk-vector-byte-view-element-regression-20261008.json`, `evidence/candidate-sdk-vector-byte-view-storage-regression-20261008.json`에서 현재 개발 후보로 확인했다. 비바이트 스칼라·nominal 뷰와 resource 원소, 릴리스 변조 API 실행은 남음.

47번 비바이트 스칼라 뷰: 원본 SDK `view<T>/view_mut<T>`와 요청된 정수별 export를 i32·i64·u32·usize·bool에 연결했다. bool/i32를 구분하는 타입 ID와 정확한 C 원소 포인터를 쓰며, 길이는 원소 수다. 읽기·쓰기·반환/태그/부분 뷰·빈 뷰·대여 해제 후 growth와 고정 배열 뷰를 실행했다. 개발/릴리스 후보 각각 정상·경계 프로그램 22개, 거부 34개, MSVC low/optimized 실행 44회가 통과했다. 개발 native API는 저장 뷰 MIR 변조 10개와 출력 보존, Option/Result 타입 왕복을 확인했다. 근거: `evidence/candidate-sdk-vector-scalar-view-api-20261008.json`, `evidence/candidate-sdk-vector-scalar-view-cli-dev-20261008.json`, `evidence/candidate-sdk-vector-scalar-view-cli-20261008.json`. 릴리스 변조 API, resource/nominal 원소와 뷰, Linux·self-hosting은 후속 범위다.

48. Vec의 resource 원소, 할당 실패 시 donor 반환, 토큰·진단의 중첩 자원 정리.
49. Buffer/String 할당·크기 변경·바이트 접근·소유권 전달 연결.
50. 부분 완료: 인증된 std::process Fairy 본문과 원본 Windows 인자 runtime을 owned HIR/MIR/C에 연결. 인자 수·EXE 이름 복사·버퍼 부족의 정확한 needed·오류 시 버퍼 보존·범위 초과·한글/빈 인자를 두 후보·C 프로필과 Go 기준으로 실행. SDK 소스/심벌 사칭과 owned authority/result/layout 변조를 거부. 조합 검증에서 드러난 else 없는 if의 대여 검사 중단도 수정. 근거: evidence/candidate-sdk-foreign-api-20261008.json, evidence/candidate-sdk-process-cli-20261008.json, evidence/candidate-borrow-if-no-else-20261008.json. UTF-8 검사·콘솔 SDK 및 실제 프로젝트 CLI 통합은 남음.
51. File open/read/write/sync/close/replace 연결.
52. 완료: SDK enum·오류 타입을 일반 import 경로에서 해석하고 Option/Result payload에 선언 정체성을 보존. HIR은 정확한 enum 레이아웃 보존을 요구하며 MIR도 payload 타입을 검증. 인증된 std::num 원본의 숫자 파서·변환 함수와 로컬 enum 저장·분기를 실제 실행. 일반/릴리스 API 3개, CLI 정상 3개·오류 3개, 생성 C 6회 및 Go 기준 native 6회가 근거: `evidence/candidate-enum-tags-api-20261008.json`, `evidence/candidate-sdk-num-cli-20261008.json`. 30비트 복합 타입 상한은 유지하며 enum leaf는 16비트다. 두 enum leaf 등 표현 상한을 넘는 조합은 계속 거부한다. 다른 SDK 본문의 runtime 연결은 46~51번에서 진행.
53. 부분 완료: 닫힌 language/11 효과 집합과 일반·scalar generic 함수 선언, 직접 호출·forward/재귀·defer의 권한 포함 관계 연결. owned HIR binding/MIR target에 effect bitset을 보존하며 MIR call의 제거된 권한·미지 비트를 거부. 17개 정상/오류 fixture·두 후보 프로필·native 12회 및 MIR mutation API test 통과. 근거: `evidence/candidate-resource-effects-calls-20261007.json`. 자원 cleanup 효과도 연결: 정리 권한과 중첩·Option/Result union을 정상 종료/return/덮어쓰기/반복 및 부분 임시값의 실제 drop에 적용. 이동 sender·trap·no-drop burn은 정리 권한을 요구하지 않으며 abilities/effects의 순서는 무관. 19개 fixture·native 22회·tagged drop 권한 위조 API test 통과. Go 대표 6개 중 5개 일치, literal assert(false) 뒤 효과 요구 여부는 차이로 보존해 83번에서 검토. 근거: `evidence/candidate-resource-effects-cleanup-20261007.json`. imported scalar effects 전파는 63번 부분 단위에서 완료; nominal/resource의 모듈 전파는 61~64번에서 연결.
54. 부분 완료: unsafe Fairy 함수와 lexical unsafe 블록의 호출 권한 검사. imported scalar metadata·generic AST 복제·defer/cleanup까지 보존하고, effects·must-use·이동·대여 검사는 그대로 유지. owned HIR은 함수/블록 트리를 독립 순회하며 범위 제거를 거부; MIR은 exact target과 호출 표시가 불일치하면 E5101. 17 fixture·native 18회와 변조 API 2개 통과. 근거: `evidence/candidate-resource-unsafe-scopes-20261007.json`. 단일 모듈 extern-C 선언·ASCII link identity와 스칼라 실행은 아래 55번에서 연결했다. raw-pointer 전달과 안전 참조·byte/text 뷰의 명시적 pointer cast를 unsafe 검사·owned HIR/MIR·C 실행에 연결했다. 근거: `evidence/candidate-resource-pointer-casts-20261008.json`. 제한된 void** 출력 슬롯은 완료: 정확한 mutable opaque pointer의 직접 지역 borrow/참조 변수만 unsafe cast하며 shared·다른 pointee·3중 포인터·tagged ref는 거부. 10개 fixture, 두 후보/C 프로필의 native 6회, 일반 FFI API 6개·릴리스 출력 슬롯 API 1개 및 pointer/cast/C scalar 회귀 통과. 릴리스 API 파일 전체 검사는 관찰자 360초 제한으로 중단하여 완료로 세지 않음. 근거: `evidence/candidate-resource-output-slot-20261008.json`. imported nominal/C ABI 시그니처는 남음.
55. 부분 완료: 단일 모듈 unsafe extern C 함수와 @link_name을 i32/i64/u8/u32/usize 인자·반환 및 void 반환에 연결. exact linker 식별자·중복 alias 시그니처·effects/must-use·unsafe 권한을 owned HIR/MIR에 보존하고 C prototype/호출을 생성. 검증용 provider를 별도 C 객체로 컴파일·링크해 두 프로필 native 18회 통과. 정상/오류 24 fixture, linker/ABI 위조 4개를 다루는 native API test 1개, 기존 unsafe C 출력 4개 동일·오류 4개 유지 확인. 근거: `evidence/candidate-resource-scalar-foreign-20261008.json`. 현재 link_name은 escape 없는 ASCII 이름, 최대 256바이트; SDK fy_ 이름은 인증된 SDK admission 이후 연결. raw pointer의 scalar/void 타입·null·복사·인자·반환·필드·동일성 비교와 C ABI 전달을 연결했다. 근거: `evidence/candidate-resource-pointer-transport-20261008.json`. borrow→pointer cast도 별도 단위에서 연결했다. C scalar 15종 전달과 i32↔c::int 명시 변환도 연결했다. 근거: `evidence/candidate-resource-c-scalars-20261008.json`. repr(C) 선언과 소유된 ABI 정보의 HIR/MIR 보존·위조 거부는 별도 단위에서 완료했다. 근거: evidence/candidate-module-repr-20261008.json. C 이름·필드 출력과 별도 C 객체의 작은/큰 집합 반환값, imported repr(C) 결과를 연결했다. 두 후보·두 C 프로필, 정상 6개·오류 9개·native 12회 및 모듈 프로그램 7개·오류 6개·native 14회와 ABI 정보 제거 거부 API 3개 통과. 큰 조건식에서 드러난 MIR 재귀 스택 초과는 분기 배열을 helper로 분리해 해결했다. 근거: evidence/candidate-module-repr-results-20261008.json, evidence/candidate-module-repr-cli-20261008.json. 타깃별 c::ssize_t 및 File/Buffer/Vec 연결은 아직 미완료.
56. 부분 완료: @must_use의 단일 무인자 선언·inert 반환 eligibility·명시 discard와 소비 문맥 연결. 괄호/블록/if/switch 끝값으로 숨긴 무시 호출은 E4202, resource/Result/borrow/void·중복·인자·잘못된 대상은 E4201. scalar generic와 effects 조합을 지원하며 HIR에 marker를 보존하고 MIR에서는 기존 계약대로 제거. 21개 fixture·native 18회·HIR marker API test·Go 대표 6개 일치. 근거: `evidence/candidate-resource-must-use-20261007.json`. 외부 scalar 함수 marker 전파는 63번에서, extern-C marker는 55번에서 연결. @test/@const_eval 조합은 후속 단위.
57. 모듈 상수·제한된 스칼라 컴파일 타임 계산·static_assert 연결.
58. 정확한 source_location 전개와 선언 문서 주석 연결.
59. 도달 가능한 소스에서 쓰는 경우 바이트 읽기 반복·region 할당 의미 연결.
60. 부분 완료: 값 보존 scalar `as` 캐스트를 이름·타입·owned HIR·MIR 검증·C 실행까지 연결. u8→i32/i64/u32/usize, i32→i64, u32→i64/usize만 허용. narrowing·부호 변경·bool/enum/raw-pointer·동일 타입은 기존 계약에 따라 거부. 12개 fixture·두 후보 프로필·native 12회 및 위조 MIR 거부 1개 API test 통과. 근거: `evidence/candidate-resource-scalar-casts-20261007.json`. 전체 자기 소스 inventory·모듈·unsafe/포인터 연결은 후속 단위에서 마무리하며 스크립트 대체 컴파일로 우회하지 않음.

현재 30비트 tagged 타입 표현은 nominal/resource leaf 하나에 14비트를 쓴다.
따라서 예시 조합 `Result<Option<Token>,Problem>`에서 양쪽 leaf가 nominal/resource라면
32비트가 필요해 표현하지 못한다. 이 예시가 현재 자기 소스에 존재한다고 주장하지 않는다.
실제 `Result<Graph,state::Problem>` 등 도달 가능한 조합부터 inventory에 넣고,
표현할 수 없는 필수 조합이 확인되면 선언 정체성을 유지하는 소유 타입 테이블 등으로 바꾼다.
중첩 깊이·레이아웃·분석량 상한은 타입 ID의 비트 상한과 별도로 유지한다.

## E. 모듈·패키지·실제로 쓸 수 있는 컴파일러 명령

61. 부분 완료: `module_catalog`와 `module_types`가 실제 Parsed source set의 모듈/선언 좌표와 nominal/tagged/borrow descriptor를 소유. 서로 다른 모듈의 같은 로컬 선언 번호는 구분하고 같은 공급자의 별칭은 canonical ID로 합침. exact source set 검증·private 접근·오류 시 interning rollback·위조 graph publication 거부를 일반/릴리스 native API에서 확인. 근거: `evidence/candidate-module-type-graph-20261008.json`. nominal 본문 typing과 joined HIR/MIR 연결은 64번에서 추가했다. 기존 scalar-only CLI·SDK·영속 package 연결은 남음.
62. 부분 완료: 새 declaration catalog의 qualified lookup에 public/private와 module/declaration 정체성 검사 적용. joined 본문 checker와 imported resource field/representation 접근 거부는 64번에서 연결했다. SDK·패키지별 정체성과 더 넓은 import admission은 남음. 근거: `evidence/candidate-module-type-graph-20261008.json`.
63. 부분 완료: imported scalar 함수의 effects·must-use를 Registry → Resolved → owned HIR에 복사해 보존. 공급자 선언 오류와 caller 권한 누락·반환값 무시를 거부하며 Registry 정리 후에도 metadata가 유지됨. dev/release native API test 각 2개 통과. 근거: `evidence/candidate-resource-module-contracts-20261007.json`. 새 owned source-set signature graph에서 nominal/resource/enum/tagged/borrow와 C scalar/output-slot 시그니처 및 exact linker bytes를 보존. provider/caller 타입 정체성 일치, parser owner 종료 뒤 조회, private 타입·함수·잘못된 선언과 cross-module foreign alias ABI 충돌을 일반/릴리스 native API에서 검증. 근거: `evidence/candidate-module-signatures-20261008.json`. nominal 본문 typing과 joined MIR/C 연결은 64번에서 추가했다. generic/반환 origin·SDK·프로젝트 CLI 연결은 남음.
64. 부분 완료: source-set AST 결합과 catalog 좌표 바인딩을 이름·타입 검사 → owned HIR → MIR → 검증된 C 생성에 연결. 서로 다른 모듈의 같은 타입/함수 이름은 구분하고 같은 공급자의 별칭은 합친다. imported struct/resource/enum, private resource 접근 거부, effects/unsafe/must-use와 원래 파일 오류 위치를 일반·릴리스 native API에서 검증. 근거: `evidence/candidate-module-join-20261008.json`, `evidence/candidate-module-program-20261008.json`. 현재 하나의 joined HIR/MIR owner와 일시적인 선언 번호를 사용한다. source-list 기반 --c-modules-file과 dev/release 후보 → low/optimized C 실제 실행도 연결했다. imported struct/resource/enum 및 provider cleanup의 실제 값·정확히 한 번 정리를 검증. 근거: evidence/candidate-module-cli-20261008.json. SDK/generic/receiver, 영속 package 정체성 및 파일별 debug map은 후속 단위.
65. manifest 기반 소스 탐색과 source-root/module 경로 검사.
66. 패키지 의존성·version/UID/language 정체성·순환 오류 처리.
67. lock 생성·locked 입력 검사와 정확한 stale-lock 종료 코드 연결.
68. bundled SDK 선택과 source/compiler/runtime identity digest 연결.
69. entry 선택·도달성·runtime/C 라이브러리 링크 연결.
70. 후보 C 출력을 지원되는 native compiler driver에 연결.
71. 소스·C 컴파일·링크·결과 공개 실패 시 기존 실행 파일 보존.
72. check/build/run, dev/release 프로필과 프로그램 인자 통합.
73. 구조화 진단과 실패 종류별 기존 종료 코드 유지.
74. native test 탐색·정확한 선택·격리 실행·JSON 결과 통합.
75. 컴파일러 프로세스 timeout과 자식 종료 정책 정리. 현재 release 검증이 60초 bootstrap 제한을 120초로 늘린 검증 전용 overlay를 사용한다. 실제 프로젝트 빌드 경로에서는 timeout 설정·진단·종료 처리를 연결해 검증 전용 EXE 의존을 제거.
76. 파일·핸들·할당 오류를 후보 CLI까지 전달.
77. 플랫폼 공통 검증 진입점 유지. PowerShell이 유일한 기여자 실행 경로가 되지 않게 정리.

## F. 새 기능을 연결하며 함께 마무리할 불변식

78. 새 값·참조 종류에 owned HIR 검증 확장.
79. MIR 레이아웃·피연산자·초기화·지배 관계 검증 확장.
80. move/borrow/cleanup 연산 검증과 위조 소유권 정보 거부.
81. 호출 시그니처·활성 tagged payload 접근·cleanup 종료 경로 검증.
82. 원래 소유자 제거와 다중 모듈 재연결 이후 source span 유지.
83. 해당 범위의 후보 결과·오류·위치를 Go 기준 구현과 비교.
84. 새 집합·SDK 경로에서도 정수 trap·cast·경계 검사 의미 유지.
85. 노드·할당·깊이·분석량 상한과 초과 시 실패 처리. 부분 성공 금지. 아래 실측 소스 크기·파일 수·배열 길이 blocker를 해당 구현과 함께 해소.
86. 실제 누락에 직접 대응하는 회귀 추가. 전체 검사 반복과 구현을 베낀 테스트 줄이기.
87. 진행표·readiness 문서를 실제로 유지되는 구현과 실행 결과에 맞춰 갱신. 현재 약 156KB인 후보 README는 기능별 문서와 짧은 진입 안내로 분리하고, 40KB를 넘은 `type_check.fy`는 타입·식·문장 책임을 기준으로 분리할 후보로 검토. 문서 정리는 구현 단위와 묶고 별도 전체 검사 반복은 피하기.

2026-10-08: 65/85번의 입력 상한 연결은 파일당 64KiB·프로그램 1MiB·256파일로 구현했다.
스캐너·catalog·AST 재결합·HIR/MIR의 상한과 소스 위치 표현을 맞췄다.
API 3개를 두 native 프로필로, CLI 정상 2개·오류 5개와 C native 4회로 확인했다.
근거: `evidence/candidate-source-limits-20261008.json`,
`evidence/candidate-source-limits-cli-20261008.json`. 별도 노드/분석량 상한과
SDK 본문 연결과 나머지 분석량 제한은 계속 남는다.
60/78/79/84/85번의 큰 rank-one 배열은 최대 4096개 원소까지 연결했다.
일반/릴리스 API와 정상 4개·오류 6개·native 8회, 기존 작은 배열 C 출력 일치를 확인했다.
근거: `evidence/candidate-large-arrays-api-20261008.json`,
`evidence/candidate-large-arrays-cli-20261008.json`. 큰 rank-two 및 자원 원소 배열은 제외한다. 87번 README 분리는 기존 60개 절을
7개 guide 문서로 옮겨 원문·상대 링크·기존 anchor를 확인했다.

### 실제 소스로 더 구체화한 선행 작업

2026-10-07 점검에서 후보 `src/*.fy`는 84개다. 아래는 기존 60/61/65/78~85/88번의
세부 작업이며, 같은 원인을 새 작업 번호로 중복 계산하지 않는다.

| 현재 소스에서 확인한 제한 | 실제 필요한 후속 작업 | 담당 |
| --- | --- | --- |
| `file_cli::load`와 `hir_lower::prepare`의 입력 상한은 32768바이트. `type_check.fy` 42757바이트, `mir_lower.fy` 39787바이트, `hir_model.fy` 34980바이트는 이미 이를 초과 | 책임에 맞는 파일 분리와 bounded 로더/분석 budget 조정을 함께 설계. 로더만 늘리고 HIR에서 다시 실패하는 불일치 제거. 메모리·곱셈 overflow와 초과 오류 유지 | 60, 85, 87, 88 |
| `module_cli::paths`와 `module_registry::register`는 각각 32파일/모듈 상한. 자기 소스는 84파일이며 bootstrap check의 SDK 포함 집합은 92파일 | package/module별 source set과 총 budget을 연결하고 두 단계의 상한을 일치시킴. 새 파일 추가 때 fixture-only manifest로 자기 소스를 빼는 우회 금지 | 61, 65, 85, 88 |
| `array_types::make`는 차원 길이 1..64만 표현. 자기 소스는 `u8[512]`, `u8[1024]`, `usize[1153]`, `usize[4096]` 등을 사용 | 실제 fixed-array 길이를 담는 타입/레이아웃 표현으로 수정. 반복 초기화·필드 저장·인덱싱·복사와 크기 계산 overflow까지 기존 배열 경로에 연결 | 60, 78, 79, 84, 85 |
| `module_registry::register`는 scalar 함수 시그니처만 등록하며 attribute/generic/receiver/unsafe를 거부 | imported nominal/resource/borrow/Vec 시그니처와 정의 정체성 등록. 이름이 같은 다른 모듈 타입을 섞지 않고, 공개 factory와 private representation 검사까지 연결 | 44, 52~56, 61~64 |
| `name_resolution::resolve`는 `*` 역참조·일부 표현·attribute/generic 선언을 거부하고 resource cleanup은 빈 본문만 허용 | 파서는 받아들이지만 이름·타입 단계가 거부하는 자기 소스 형태를 위치별로 분류. 단계마다 하나씩 닫힌 거부 경로를 열고 HIR/MIR까지 연결 | 21, 31, 44, 53~60, 88 |
| `hir_view_check`는 단일 exact origin을 보존하며 mixed/recursive 반환을 거부 | 도달 가능한 호출 관계로 필요한 origin을 먼저 확정. 필드/인덱스/재대여 경로와 호출 인자 보유 기간을 통합하고 원래 소유자의 수명을 보존 | 32~40 |
| `c_cli::run`은 단일 파일의 C를 stdout으로 내보내며 `fn i32 main()`과 2MiB 출력 상한을 사용 | 전 프로그램 연결·entry 선택·runtime 심벌·native driver를 연결. 파일 출력 실패와 C/link 실패를 구분하고 기존 EXE 공개를 마지막 성공 단계로 유지 | 64, 69~76, 85 |

이 제한들은 현재 소스의 admission 조건에서 확인한 사실이다. 자기 소스 전체를 후보로
컴파일해 얻은 런타임 실패 목록이나 완전한 blocker 목록으로 표현하지 않는다.
기능 inventory는 오류 하나를 고칠 때마다 다음 오류만 찾아가는 방식 대신,
전체 도달 소스를 한 번 분류해 구현 의존성을 미리 드러내는 데 사용한다.

## G. 셀프호스팅과 기준 컴파일러 전환

88. 자기 소스 기능 목록과 도달 가능한 blocker 확정.
89. Go stage0로 후보 전체 소스를 컴파일해 stage1 생성.
90. 같은 소스와 잠긴 입력을 stage1으로 컴파일해 stage2 생성.
91. 컴포넌트 결과·오류·위치·결정적 산출물 비교.
92. 정상·잘못된 소스·mutation corpus에서 stage1/stage2 동작 비교.
93. 결정적인 경우 stage 해시 비교. 도구 메타데이터 차이는 재현해 설명.
94. 재현 가능한 bootstrap receipt와 돌아갈 수 있는 Go bootstrap 경로 유지.
95. 기존 promotion gate 충족 후 기준 컴파일러 전환 판단.
96. Linux·backend·패키지 차이를 전환 완료 주장에 정확히 반영.

## H. 나중에 묶어서 할 Linux 검증

97. Drimi 수정 패치를 현재 소스와 먼저 대조.
98. 현재 구현된 `.i` 입력과 Clang의 `-Wno-error=gnu-line-marker` 한정 처리를 Linux에서 확인. 소스 수정은 이미 있으므로 미구현으로 다시 세지 않고 실제 GCC/Clang 캐시 실행 결과로 판단.
99. GCC/Clang × dev/release를 제한된 한 묶음으로 실행.
100. i64 helper와 정수·tagged·resource 실제 native 동작 확인.
101. cache miss/hit/무효화/손상과 소스 오류 후 이미지 보존 확인.
102. stale-lock과 unlocked 의미 오류의 종료 코드를 별도로 확인.
103. 저장 write/sync/publication 실패와 제어된 자식 중단 확인. 주입과 실제 OS/SDK 오류 구분.
104. 이식 소스·runtime의 의도치 않은 Windows 전용 경로 확인.
105. 공통 명령·도구 identity·원시 실패 증거 제공. 물리 전원 차단·디스크 고갈 실험 제외.

## I. 처음 목표였던 빌드 비용과 배포 준비

106. clean build 후 생성 C·object·EXE·debug·cache·검증 부산물 용량을 나눠 측정.
107. 함수 본문·interface·SDK 수정별 재빌드 범위와 시간 측정. 경과 시간과 CPU/cycle 구분.
108. 반복 개발 cache 최대 용량·보관·정리 정책 설정.
109. compiler/profile/toolchain identity가 맞는 runtime object 재사용.
110. 패키지·함수 단위 컴파일의 비용과 실행 성능 절충 평가.
111. 제네릭 특수화 코드 증가량 측정과 도달하지 않는 인스턴스 제거.
112. 선언한 비용 목표에 따라 debug 정보·최적화·LTO 평가.
113. 과거 검증 복사본 줄이기. 단위별 짧은 receipt와 실패 재현 중심 보관.
114. CI runner 시간·업로드 artifact·cache 저장 비용을 구분하고 승인된 push 묶기.
115. 지원 플랫폼별 compiler·SDK·digest·bootstrap 안내 패키징.
116. 설치·첫 프로젝트·지원 범위·알려진 제한 안내 정리.
117. 대표 유틸리티를 후보로 빌드해 실제 사용 경로 확인.
118. formatter·편집기 grammar·색상·의미 진단을 지원 문법과 맞추기.
119. 호환성·버전 전환을 검토하고 호출자가 옮겨간 뒤 구형 scaffolding 정리.

## J. 필요가 입증될 때 판단할 선택 기능

120. 기존 바이트 뷰와 ADR0430의 공유 i64 배열 뷰를 넘어 다른 원소 타입·가변 i64·Vec 비소유 뷰 확장 여부 판단. 기존 지원 이식과 새 언어 기능 도입을 구분.
121. 현재 SDK 예외를 넘어 일반 사용자 제네릭 집합 타입 확장.
122. 실제 여러 타입 알고리즘에 필요한 작은 정적 제약.
123. 실제 요구와 빌드 비용이 뒷받침하는 컴파일 타임 계산 확장.
124. 일반 구조체로 해결하기 어려운 도메인 혼동이 입증된 뒤 nominal scalar 축약 판단.
125. 더 넓은 public tagged-aggregate FFI·추가 target·편집기 LSP 기능.
126. C 출력 경로가 측정된 목표를 막을 때 직접 native backend·추가 최적화 판단.
127. 별도 제품 목표가 선택된 뒤 contract/on-chain profile 확장.

## 실행 순서와 묶음 완료 기준

| 묶음 | 항목 | 구현 결과 |
| --- | --- | --- |
| 자원 실행 | 21~30 | resource MIR/C와 cleanup을 연결해 정상 종료·반환·덮어쓰기·중첩 자원의 실제 실행을 확인한다. MIR 검증 항목 78~86을 해당 변경과 함께 처리한다. |
| 자기 소스 접근 | 31~43 | 필요한 borrow·재대여·뷰와 호출 origin을 연결한다. 실제 compiler 소스에서 쓰는 형태부터 처리한다. |
| 자기 소스 의존성 | 44~60 | 특수화·SDK 컨테이너·파일·효과·FFI·캐스트를 연결한다. 모듈 항목 61~64와 서로 필요한 범위를 함께 구현한다. |
| 프로젝트 명령 | 61~77 | 여러 파일과 SDK가 포함된 프로젝트를 후보로 check/build/run/test한다. 실패하면 이전 실행 파일을 보존한다. |
| 단계 전환 | 88~96 | 같은 소스로 stage1과 stage2를 만들고 결과·진단·bootstrap 복구 경로를 비교한다. |
| Linux 일괄 확인 | 97~105 | Drimi의 기존 수정과 GCC/Clang 두 프로필, 캐시·저장 실패를 한 묶음으로 확인한다. |
| 원래 제품 목표 | 106~119 | 빌드 시간·부산물·캐시 용량을 측정하고 실제 기여자 실행·편집기·패키징 경로를 마무리한다. |
| 별도 선택 | 120~127 | 실제 요구가 입증되면 새 언어 기능·추가 backend·target 도입을 판단한다. |

78~87은 매번 전체 검증을 추가하는 별도 단계가 아니라 해당 구현의 검증·한계·문서 작업이다.
새로운 미지원 사례가 발견되면 담당 항목에 재현을 붙이며, 동일한 원인을 새 할 일로 중복 계산하지 않는다.
20번 완료 후 주요 작업 99개(21~119)와 선택 검토 8개(120~127)를 구분한다. 주요 작업 모두가
셀프호스팅 전에 끝나야 한다는 뜻은 아니며, 자기 소스의 도달 가능한 기능이 단계 전환을 결정한다.

즉시 다음 작업은 **47번 비바이트 스칼라 Vec 뷰와 48번 자원 원소 실행**이다. 41번 CStr 표현형과 50번 process runtime의 bounded 실행은 연결했으며, 다른 SDK 본문 및 프로젝트 driver는 남는다. 61~64번 nominal source-set와 54~55번 inert repr(C) 반환값 연결은 완료했으며, SDK/영속 package 정체성 등 남은 범위는 각 항목에 유지했다.
선택 기능은 별도 도입 판단을 거친다. 이 목록은 push·PR·설치·release를 승인하지 않는다.
