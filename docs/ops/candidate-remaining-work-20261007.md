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

48. 완료: 인증된 SDK AST에서 선언에 바인딩된 resource 원소를 구체화한다. 서로 다른 resource의 Vec·take-slot 정체성과 원본 SDK 바이트를 보존하며, 개발 native API 2개를 통과했다. 근거: `evidence/candidate-sdk-vector-resource-expand-20261008.json`. resource 원소의 저장·꺼내기·대여·실패 시 donor 반환·역순/중첩 정리도 실행했다. 개발/릴리스 각각 프로그램 5개·거부 4개, 생성 C 20회와 개발 storage API 2개, 스칼라 저장 회귀 8회가 통과했다. 단락 평가의 RHS에서만 생긴 임시 자원도 합류 전에 정리한다. 근거: `evidence/candidate-sdk-vector-resource-api-20261008.json`, `evidence/candidate-sdk-vector-resource-cli-dev-20261008.json`, `evidence/candidate-sdk-vector-resource-cli-release-20261008.json`, `evidence/candidate-sdk-vector-resource-scalar-regression-20261008.json`. struct 원소 Vec도 연결했다. 토큰 형태의 5필드 구조체, 원본 복사 보존, 할당 실패, 패딩·중첩 태그, 원소 참조를 두 후보와 생성 C 16회로 확인했다. 개발 native API 2개도 통과했다. 근거: `evidence/candidate-sdk-vector-struct-api-20261008.json`, `evidence/candidate-sdk-vector-struct-cli-dev-20261008.json`, `evidence/candidate-sdk-vector-struct-cli-release-20261008.json`. 실제 scanner 조합에 필요한 프로그램 AST/HIR 표현식 상한 16384와 owned unsafe 방문표, 전역 심벌 조회를 연결했다. 큰 프로그램·대여 충돌·상한 초과를 개발 후보와 native 2회로 검증했다. 근거: `evidence/candidate-program-capacity-dev-20261009.json`. 큰 프로그램은 최종 개발/릴리스 후보와 native 4회로 재확인했다. 근거: `evidence/candidate-program-capacity-profiles-20261009.json`. 59번 바이트 반복과 구조체 배열 필드 subrange 대여를 연결했다. 배열 필드 대여는 개발/릴리스 각각 프로그램 6개·거부 7개·native 합계 24회와 개발 owned API 1개, 바이트 반복 회귀 native 16회를 통과했다. 근거: `evidence/candidate-array-field-views-dev-20261009.json`, `evidence/candidate-array-field-views-release-20261009.json`, `evidence/candidate-array-field-views-api-20261009.json`, `evidence/candidate-array-field-byte-loop-regression-20261009.json`. 입력 parameter와 cooked 저장소 중 하나를 고르는 local 혼합 origin을 HIR 및 독립 MIR에 연결했다. 두 후보 각각 프로그램 4개·거부 8개·native 합계 16회, 개발 API 2개와 text 회귀 native 12회가 통과했다. 모호한 다중 입력 반환은 계속 E2103으로 거부한다. 근거: `evidence/candidate-mixed-origins-dev-20261009.json`, `evidence/candidate-mixed-origins-release-20261009.json`, `evidence/candidate-mixed-origins-api-20261009.json`, `evidence/candidate-mixed-origin-text-regression-20261009.json`. 도달성/origin 캐시와 소유자별 충돌 인덱스로 HIR 상한을 해소했고 MIR 식 함수 분리로 기본 스택 초과를 수정했다. 개발/릴리스 후보가 원본 scanner·SDK를 컴파일하고 Token/raw/cooked/진단·E0010·중첩 정리를 실행했다. 303개 fixture·실제 소스 2개의 현재/이전 문법·변조·예산 조합 7670개를 native 두 프로필씩 Go lexer와 비교했다. 총 30680개 관측과 잘못된 프레임 92회가 일치하며 생성 C도 동일하다. 근거: `evidence/candidate-scanner-corpus-parity-20261009.json`. 전체 컴파일러 승격·Linux·자기 소스 전체 컴파일은 별도 항목이다.
49. 완료: 원본 인증 SDK의 Buffer 할당·reserve/append·zeroed 크기 변경·truncate/clear·공유/가변 접근·copy/move·정리를 연결했다. 개발/릴리스 각각 정상 4개·거부 5개와 생성 C 16회, 개발 foreign 변조·mutable 충돌 API 2개, 자원 Vec 회귀 native 10회를 통과했다. 전체 SDK 분석의 반복 origin/path 계산과 중첩 if MIR의 스택 초과도 수정했다. 근거: `evidence/candidate-sdk-buffer-api-20261008.json`, `evidence/candidate-sdk-buffer-cli-20261008.json`, `evidence/candidate-sdk-buffer-resource-regression-20261008.json`. 원본 인증 String/CString·UTF-8 SDK도 연결했다. 개발/릴리스 각각 정상 6개·거부 7개와 생성 C 24회, text 서명/구조체/인증 변조 11개 및 pointer ABI 회귀 API 2개, Buffer 회귀 8회가 통과했다. 무할당 Buffer↔String 포인터 보존, 잘못된 UTF-8 donor 반환, CString NUL·할당 실패와 대여 수명을 확인했다. Go 기준이 허용하는 borrowed switch 결과의 오래된 거부 조건도 제거했다. 근거: `evidence/candidate-sdk-text-api-20261009.json`, `evidence/candidate-sdk-text-cli-20261009.json`, `evidence/candidate-sdk-text-reference-20261009.json`, `evidence/candidate-sdk-text-buffer-regression-20261009.json`. 실제 scanner 토큰·진단 통합은 48번에서 검증했다.
50. 부분 완료: 인증된 std::process Fairy 본문과 원본 Windows 인자 runtime을 owned HIR/MIR/C에 연결. 인자 수·EXE 이름 복사·버퍼 부족의 정확한 needed·오류 시 버퍼 보존·범위 초과·한글/빈 인자를 두 후보·C 프로필과 Go 기준으로 실행. SDK 소스/심벌 사칭과 owned authority/result/layout 변조를 거부. 조합 검증에서 드러난 else 없는 if의 대여 검사 중단도 수정. 근거: evidence/candidate-sdk-foreign-api-20261008.json, evidence/candidate-sdk-process-cli-20261008.json, evidence/candidate-borrow-if-no-else-20261008.json. UTF-8 검사와 borrowed 결과 추출은 49번 String/text 실행에서 연결했다. 콘솔 SDK도 원본 Windows runtime에 연결했다. 개발/릴리스 각각 프로그램 5개·거부 4개·native 합계 20회, 개발/릴리스 각각 권한·효과·시그니처·결과 구조체 변조 11개를 확인했다. 근거: `evidence/candidate-sdk-io-cli-dev-20261009.json`, `evidence/candidate-sdk-io-cli-release-20261009.json`, `evidence/candidate-sdk-io-api-20261009.json`, `evidence/candidate-sdk-io-api-release-20261009.json`. 원본 파일/manifest 로더를 `file_source.fy`로 분리해 file_cli↔module_cli 순환을 제거하고 후보가 원본 File/text/Buffer SDK와 함께 컴파일·실행했다. 두 후보·native 두 프로필에서 빈 파일·64KiB 정확한 경계·초과·잘못된 UTF-8/없는 경로·할당 실패와 파일/버퍼 정리를 확인했다. 기존 CLI 5개 사례는 이전 개발/새 개발/새 릴리스 이미지 간 stdout/stderr/exit가 같고 Go CLI 의존성 검사는 실제 source 목록으로 통과했다. 근거: `evidence/candidate-file-source-dev-20261009.json`, `evidence/candidate-file-source-release-20261009.json`, `evidence/candidate-file-source-cli-regression-20261009.json`. token 경로는 `file_tokens.fy`로 분리했고 기존 명령 14개 사례의 stdout/stderr/exit를 유지했다. 원본 SDK들과의 후보 C 생성은 AST join의 entry 1244개가 기존 1024개 상한을 넘어 거부됨을 native Fairy 파서로 실측했다. 근거: `evidence/candidate-file-token-route-regression-20261009.json`, `evidence/candidate-file-token-join-budget-20261009.json` (실패 probe는 `evidence/candidate-file-tokens-dev-20261009.json`). 인증된 SDK 함수 도달성과 AST 압축을 별도 컴파일 API에 연결해 이 join 제한을 해소했다. 이름 조회 캐시로 기존 1048576 작업량 예산을 유지했고, 바이트 반복문의 숨은 인덱스가 실제 호출 인자를 전체 반복 끝까지 붙잡던 위치 오류도 수정했다. 두 후보가 같은 C 769317바이트를 만들고 두 native 프로필에서 정상/문자열 오류/크기 초과/없는 파일 4개 입력의 stdout·stderr와 개별 종료 상태가 기존 CLI와 일치했다. 후보별 미사용 사용자 오류·비공개 SDK 호출·효과 누락 3개도 계속 거부한다. 바이트 반복 회귀는 후보별 정상 2개·거부 1개, native 합계 8회로 확인했다. 근거: `evidence/candidate-sdk-ast-compact-api-20261009.json`, `evidence/candidate-file-tokens-fixed-dev-20261009.json`, `evidence/candidate-file-tokens-fixed-release-20261009.json`, `evidence/candidate-byte-loop-calls-dev-20261009.json`, `evidence/candidate-byte-loop-calls-release-20261009.json`. 다음은 전체 명령 dispatcher를 이 후보 경로에서 컴파일·실행하는 작업이며, 50번 전체는 아직 부분 완료다. 전체 dispatcher의 실제 137파일 입력은 파서 상한에서 거부됐다. 큰 자기 소스 12개 중 `hir_model.fy`, `mir_lower.fy`, `type_check.fy`, `hir_lower.fy`가 같은 제한에 걸림을 실제 Fairy 선언 CLI로 확인했다. MIR 변환은 기존 13개 함수 본문을 유지해 6개 파일(최대 15044바이트)로 나눴고 모두 기존 파서 상한에서 읽힌다. 이전 릴리스/새 일반 후보의 생성 C 3개가 같고 native 12회·대여 충돌 거부가 통과했다. 근거: `evidence/candidate-mir-source-split-20261009.json`, `evidence/candidate-mir-source-split-cli-20261009.json`. 새 전체 입력 142파일·1041199바이트는 아직 파서에서 거부된다. HIR 모델은 검사 조건과 순서를 유지한 9개 파일로 분리해 파서 제한을 해소했다. 기존 정리/receiver 위조 13개가 계속 거부되고, 이전 후보와 같은 C 3개·native 12회가 통과했다. 근거: `evidence/candidate-hir-model-source-split-20261009.json`, `evidence/candidate-hir-model-split-mutation-20261009.json`, `evidence/candidate-hir-model-split-cli-20261009.json`. 전체 입력은 150파일·구분자 포함 1047406바이트로 1MiB 이내이며 아직 파서에서 거부된다. 다음은 타입 검사·HIR 변환 자기 소스의 파서 제한을 해소하는 작업이다.
51. 완료: 인증된 std::fs의 외부 함수 37개와 C 구조체 9개를 owned MIR/C에 연결. 함수 이름·심벌 위치·filesystem 효과·스칼라/구조체 포인터·반환 구조체를 검증하며, 권한·소스·ABI·시그니처 변조 14개와 사용자 SDK 심벌 사칭을 거부. 개발/릴리스 native API 검증 근거: `evidence/candidate-sdk-file-api-20261009.json`. enum/구조체 및 Option/구조체 Result를 별도 private 인코딩으로 연결해 기존 타입 ID를 유지하고 전체 SDK의 C 생성 차단을 해소했다. 512개 ID 왕복과 native API 6개, 두 후보의 정상 3개·오류 2개 fixture·native 6회 통과. 근거: `evidence/candidate-result-pairs-20261009.json`. 원본 Windows runtime에서 open/write/write_all/sync/replace/read/EOF/close/remove와 FileKind Result를 실제 실행했다. 한글 경로·NUL 포함 데이터·짧은 읽기 뒤 버퍼 보존·잘못된 경로·사라진 파일, filesystem 효과 누락과 SDK 심벌 사칭을 두 후보·native 두 프로필로 확인. 근거: `evidence/candidate-sdk-file-lifecycle-20261009.json` (정상 프로그램 1개, 거부 2개, native 4회). `file_cli::load`의 후보 컴파일과 프로젝트 명령 통합은 50번에 계속 남음.
52. 완료: SDK enum·오류 타입을 일반 import 경로에서 해석하고 Option/Result payload에 선언 정체성을 보존. HIR은 정확한 enum 레이아웃 보존을 요구하며 MIR도 payload 타입을 검증. 인증된 std::num 원본의 숫자 파서·변환 함수와 로컬 enum 저장·분기를 실제 실행. 일반/릴리스 API 3개, CLI 정상 3개·오류 3개, 생성 C 6회 및 Go 기준 native 6회가 근거: `evidence/candidate-enum-tags-api-20261008.json`, `evidence/candidate-sdk-num-cli-20261008.json`. 30비트 복합 타입 상한은 유지하며 enum leaf는 16비트다. 기존 prefix 표현 상한을 넘는 enum/record 및 Option-record 쌍은 51번의 별도 Result 인코딩으로 연결했다. 새 pair 자체의 추가 중첩은 계속 거부한다. 다른 SDK 본문의 runtime 연결은 46~51번에서 진행.
53. 부분 완료: 닫힌 language/11 효과 집합과 일반·scalar generic 함수 선언, 직접 호출·forward/재귀·defer의 권한 포함 관계 연결. owned HIR binding/MIR target에 effect bitset을 보존하며 MIR call의 제거된 권한·미지 비트를 거부. 17개 정상/오류 fixture·두 후보 프로필·native 12회 및 MIR mutation API test 통과. 근거: `evidence/candidate-resource-effects-calls-20261007.json`. 자원 cleanup 효과도 연결: 정리 권한과 중첩·Option/Result union을 정상 종료/return/덮어쓰기/반복 및 부분 임시값의 실제 drop에 적용. 이동 sender·trap·no-drop burn은 정리 권한을 요구하지 않으며 abilities/effects의 순서는 무관. 19개 fixture·native 22회·tagged drop 권한 위조 API test 통과. Go 대표 6개 중 5개 일치, literal assert(false) 뒤 효과 요구 여부는 차이로 보존해 83번에서 검토. 근거: `evidence/candidate-resource-effects-cleanup-20261007.json`. imported scalar effects 전파는 63번 부분 단위에서 완료; nominal/resource의 모듈 전파는 61~64번에서 연결.
54. 부분 완료: unsafe Fairy 함수와 lexical unsafe 블록의 호출 권한 검사. imported scalar metadata·generic AST 복제·defer/cleanup까지 보존하고, effects·must-use·이동·대여 검사는 그대로 유지. owned HIR은 함수/블록 트리를 독립 순회하며 범위 제거를 거부; MIR은 exact target과 호출 표시가 불일치하면 E5101. 17 fixture·native 18회와 변조 API 2개 통과. 근거: `evidence/candidate-resource-unsafe-scopes-20261007.json`. 단일 모듈 extern-C 선언·ASCII link identity와 스칼라 실행은 아래 55번에서 연결했다. raw-pointer 전달과 안전 참조·byte/text 뷰의 명시적 pointer cast를 unsafe 검사·owned HIR/MIR·C 실행에 연결했다. 근거: `evidence/candidate-resource-pointer-casts-20261008.json`. 제한된 void** 출력 슬롯은 완료: 정확한 mutable opaque pointer의 직접 지역 borrow/참조 변수만 unsafe cast하며 shared·다른 pointee·3중 포인터·tagged ref는 거부. 10개 fixture, 두 후보/C 프로필의 native 6회, 일반 FFI API 6개·릴리스 출력 슬롯 API 1개 및 pointer/cast/C scalar 회귀 통과. 릴리스 API 파일 전체 검사는 관찰자 360초 제한으로 중단하여 완료로 세지 않음. 근거: `evidence/candidate-resource-output-slot-20261008.json`. imported nominal/C ABI 시그니처는 남음.
55. 부분 완료: 단일 모듈 unsafe extern C 함수와 @link_name을 i32/i64/u8/u32/usize 인자·반환 및 void 반환에 연결. exact linker 식별자·중복 alias 시그니처·effects/must-use·unsafe 권한을 owned HIR/MIR에 보존하고 C prototype/호출을 생성. 검증용 provider를 별도 C 객체로 컴파일·링크해 두 프로필 native 18회 통과. 정상/오류 24 fixture, linker/ABI 위조 4개를 다루는 native API test 1개, 기존 unsafe C 출력 4개 동일·오류 4개 유지 확인. 근거: `evidence/candidate-resource-scalar-foreign-20261008.json`. 현재 link_name은 escape 없는 ASCII 이름, 최대 256바이트; SDK fy_ 이름은 인증된 SDK admission 이후 연결. raw pointer의 scalar/void 타입·null·복사·인자·반환·필드·동일성 비교와 C ABI 전달을 연결했다. 근거: `evidence/candidate-resource-pointer-transport-20261008.json`. borrow→pointer cast도 별도 단위에서 연결했다. C scalar 15종 전달과 i32↔c::int 명시 변환도 연결했다. 근거: `evidence/candidate-resource-c-scalars-20261008.json`. repr(C) 선언과 소유된 ABI 정보의 HIR/MIR 보존·위조 거부는 별도 단위에서 완료했다. 근거: evidence/candidate-module-repr-20261008.json. C 이름·필드 출력과 별도 C 객체의 작은/큰 집합 반환값, imported repr(C) 결과를 연결했다. 두 후보·두 C 프로필, 정상 6개·오류 9개·native 12회 및 모듈 프로그램 7개·오류 6개·native 14회와 ABI 정보 제거 거부 API 3개 통과. 큰 조건식에서 드러난 MIR 재귀 스택 초과는 분기 배열을 helper로 분리해 해결했다. 근거: evidence/candidate-module-repr-results-20261008.json, evidence/candidate-module-repr-cli-20261008.json. 타깃별 c::ssize_t 및 File/Buffer/Vec 연결은 아직 미완료.
54~55번 추가 선행 연결: inert struct 원시 포인터의 선언 정체성·모듈 전달·safe-ref cast와 repr(C) pointee의 외부 함수 인자/반환을 연결했다. 개발/릴리스 각각 정상 5개·거부 5개와 생성 C 20회, ABI 증거 제거·포인터 namespace API 2개, Buffer 회귀 8회가 통과했다. 근거: `evidence/candidate-nominal-pointer-api-20261008.json`, `evidence/candidate-nominal-pointer-cli-20261008.json`, `evidence/candidate-nominal-pointer-buffer-regression-20261008.json`. 이 선행 구현만으로 String SDK 실행이나 resource 포인터를 완료 처리하지 않는다.

56. 부분 완료: @must_use의 단일 무인자 선언·inert 반환 eligibility·명시 discard와 소비 문맥 연결. 괄호/블록/if/switch 끝값으로 숨긴 무시 호출은 E4202, resource/Result/borrow/void·중복·인자·잘못된 대상은 E4201. scalar generic와 effects 조합을 지원하며 HIR에 marker를 보존하고 MIR에서는 기존 계약대로 제거. 21개 fixture·native 18회·HIR marker API test·Go 대표 6개 일치. 근거: `evidence/candidate-resource-must-use-20261007.json`. 외부 scalar 함수 marker 전파는 63번에서, extern-C marker는 55번에서 연결. @test/@const_eval 조합은 후속 단위.
57. 모듈 상수·제한된 스칼라 컴파일 타임 계산·static_assert 연결.
58. 정확한 source_location 전개와 선언 문서 주석 연결.
59. 부분 완료: named immutable shared byte descriptor와 직접 슬라이스의 copied u8 반복을 owned HIR의 capture/For/Index에 연결했다. 경계 한 번 평가, nested/generic 반복, shadowing, continue/break/return과 cleanup을 개발/릴리스 각각 프로그램 4개·거부 7개·native 합계 16회 및 개발 owned API 1개로 확인했다. 근거: `evidence/candidate-byte-loops-dev-20261009.json`, `evidence/candidate-byte-loops-release-20261009.json`, `evidence/candidate-byte-loops-api-20261009.json`. 다른 반복 형태와 region 할당은 도달 소스의 요구에 따라 남는다.
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

이전 단계에서 다음 작업으로 정했던 항목은 **51번 파일 SDK의 후보 실행과 50번 프로젝트 명령의 SDK 연결**이었다. local 혼합 origin은 HIR/독립 MIR와 두 후보 실행에 연결했다. 구조체 배열 필드 subrange 대여는 두 프로필로 검증했다. 50번 콘솔 SDK와 59번 copied byte 반복은 실행 검증을 마쳤다. 47번 스칼라 Vec와 48번 resource/struct 원소 실행, 49번 Buffer/String/CString·UTF-8 실행은 연결했다. 41번 CStr 표현형과 50번 process runtime의 bounded 실행도 연결했으며, 다른 SDK 본문 및 프로젝트 driver는 남는다. 61~64번 nominal source-set와 54~55번 inert repr(C) 반환값 연결은 완료했으며, SDK/영속 package 정체성 등 남은 범위는 각 항목에 유지했다.
선택 기능은 별도 도입 판단을 거친다. 이 목록은 push·PR·설치·release를 승인하지 않는다.

2026-10-09: 85번 HIR 도달성 구간 인덱스와 반환 summary 확정 후 origin 캐시를 연결했다. 개발/릴리스 혼합 origin 정상 4개·거부 10개·native 합계 16회, String/CString/UTF-8 정상 6개·거부 7개·native 합계 24회와 개발 API 2개가 통과했다. 깊이 초과 alias와 미확정 recursive 반환은 계속 거부한다. Scanner는 초기 origin 상한을 지나 가변 대여의 전체 식 충돌 비교 상한에서 멈춘다. 분석량 상한은 올리지 않았다. 근거: `evidence/candidate-scanner-reach-limit-profile-20261009.json`, `evidence/candidate-borrow-index-mixed-dev-20261009.json`, `evidence/candidate-borrow-index-mixed-release-20261009.json`, `evidence/candidate-borrow-index-text-dev-20261009.json`, `evidence/candidate-borrow-index-text-release-20261009.json`.

2026-10-09: 85번 가변 대여 충돌 비교를 소유자별 식 인덱스로 줄여 scanner HIR 상한을 해소했다. 릴리스 후보가 원본 scanner와 SDK를 C로 컴파일했고 Token/raw/cooked/진단 저장·E0010·중첩 자원 정리를 native 두 프로필에서 실행했다. 개발 후보의 scanner 컴파일은 기본 Windows 스택에서 0xC00000FD로 종료하며 전체 차등 검증도 남는다. 근거: `evidence/candidate-scanner-sdk-release-20261009.json`, `evidence/candidate-borrow-access-api-20261009.json`, `evidence/candidate-borrow-access-mixed-small-dev-20261009.json`, `evidence/candidate-borrow-access-mixed-small-release-20261009.json`.

2026-10-09: 개발 scanner의 기본 스택 초과가 MIR 식 변환에서 발생함을 단계별로 확인했고, 식 종류별 함수 분리로 해소했다. 개발/릴리스 후보가 같은 원본 scanner와 SDK를 컴파일해 native 두 프로필에서 Token/raw/cooked/진단·E0010·자원 정리를 실행했다. Scanner 및 혼합 origin 프로그램 4개의 C 출력은 후보 프로필 사이에서 동일하며 스택·분석량·깊이 상한은 올리지 않았다. 전체 차등 corpus와 프로젝트 명령, Linux·stage1/stage2는 계속 남는다. 근거: `evidence/candidate-scanner-sdk-dev-20261009.json`, `evidence/candidate-scanner-sdk-final-release-20261009.json`, `evidence/candidate-mir-frames-parity-20261009.json`, `evidence/candidate-mir-frames-text-20261009.json`.

2026-10-09: 60/85/87/88번 타입 검사 소스를 coordinator와 7개 helper로 분리했다. 8개 파일 파싱, 개발 후보 빌드, 기존 본문 보존, 정상 3개 C 출력 일치·native 12회와 타입 오류 8개의 종료 코드·진단 일치를 확인했다. 전체 dispatcher 157파일·1054271바이트는 아직 parser limit으로 거부된다. `hir_lower.fy` 분리와 1MiB를 넘긴 import/helper 중복 축소가 다음 작업이다. 상한은 올리지 않았다. 근거: `evidence/candidate-type-source-split-20261009.json`, `evidence/candidate-type-source-split-cli-20261009.json`, `evidence/candidate-dispatcher-admission-after-type-split-20261009.json`.

2026-10-09: `hir_lower.fy`를 block/program coordinator와 bindings/expression/statement helper로 분리했다. 4개 파일 파싱, 5개 함수 본문 보존, 개발 후보 빌드와 C 출력 3개 일치·native 12회를 확인했다. 전체 dispatcher는 파싱을 통과하고 module program phase 1 code 10의 누적 소스 상한에 도달했다. 현재 160파일·1055400바이트이며, 다음은 중복 축소로 1MiB 안에 맞추는 작업이다. 근거: `evidence/candidate-hir-lower-source-split-20261009.json`, `evidence/candidate-hir-lower-source-split-cli-20261009.json`, `evidence/candidate-dispatcher-admission-after-hir-lower-split-20261009.json`.

2026-10-09: 공통 CLI 분석·출력과 HIR 기본 생성자를 재사용하고 중복 import를 정리해 전체 입력을 6884바이트 줄였다. 160파일·1048516바이트로 기존 1MiB 상한과 파싱을 통과했다. 개발 CLI 46개 사례의 stdout/stderr/종료 코드, 타입 오류 8개, C 출력 5개 일치와 native 20회를 확인했다. 다음 blocker는 `mir_output::row`의 `usize[20]` 값 인자를 module signature 단계가 거부하는 phase 1 code 9다. 60/61/65/85/88번 source-set 고정 배열 타입 정체성·시그니처·후속 join/bind 연결이 남으며, 새 구현을 넣기 위한 소스 여유도 60바이트뿐이다. 근거: `evidence/candidate-cli-source-dedup-20261009.json`, `evidence/candidate-source-dedup-native-20261009.json`, `evidence/candidate-dispatcher-admission-after-source-dedup-20261009.json`.

2026-10-09: AST 재결합의 긴 설명을 별도 설계 문서로 옮겨 구현 여유를 확보하고 source-set 고정 배열 시그니처를 연결했다. 원소는 canonical 그래프 ID로, 길이는 기존 rank-one 1..4096·rank-two 1..64 범위로 보존한다. 두 native 프로필의 API 3개, 두 후보의 정상 5개·거부 10개와 C 출력 일치·native 20회가 통과했다. 실제 dispatcher 160파일·1046965바이트는 `usize[20]` 시그니처를 통과하고 SDK compaction phase 8 code 11에 도달했다. 다음은 85/88번 도달성의 이름 참조 인덱스 상한을 중복·불필요한 참조 축소로 해결하는 작업이며, 현재 상한 초과를 할당 실패로 표시하는 구분도 바로잡아야 한다. 상한은 올리지 않았다. 근거: `evidence/candidate-module-array-signatures-api-20261009.json`, `evidence/candidate-module-array-signatures-parity-20261009.json`, `evidence/candidate-dispatcher-admission-after-array-signatures-20261009.json`.

2026-10-09: SDK 도달성 참조 인덱스의 16384개 상한 초과는 code 10, 실제 Vec 할당 실패는 code 11로 분리했다. 실제 dispatcher 160파일·1047016바이트가 같은 위치에서 phase 8 code 10으로 거부되어 인덱스 상한 초과를 확인했다. 기존 SDK compaction native 검사 2개도 올바른 SDK 경로 인자로 통과했다. 다음은 인덱스에서 중복·불필요한 이름 참조를 줄이는 작업이다. 근거: `evidence/candidate-reachability-limit-classification-20261009.json`, `evidence/candidate-dispatcher-admission-reach-limit-classified-20261009.json`.

2026-10-09: 참조별 Use 목록을 선언별 first/stop 식 구간으로 바꿨다. 22개 실제 파싱 모듈의 이름 참조 17600개를 원문·토큰·사용자 본문 보존과 함께 통과시켰고, SDK native 검사 3개와 파일 CLI 4개 사례의 C 출력 일치·native 4회를 확인했다. 실제 dispatcher 160파일·1046901바이트는 이전 참조 목록 제한을 지나 `numbers.fy`의 이름 조회 분석량 상한에서 phase 8 code 10으로 멈춘다. 다음은 catalog의 전체 테이블 선형 조회를 모듈 구간 조회로 줄이고 실제 조회량을 계산하는 작업이다. 근거: `evidence/candidate-reachability-ranges-20261009.json`, `evidence/candidate-dispatcher-admission-reach-ranges-20261009.json`.

2026-10-09: catalog 조회를 모듈별 정렬 구간으로 제한하고 실제 방문 횟수를 계산했다. 도달성 분석은 선언 소유자를 재사용하며 모든 사용자 선언을 보존하므로 비한정 사용자 이름의 SDK 조회를 생략한다. catalog native 검사 2개와 SDK 검사 3개 통과. dispatcher 160파일·1048411바이트는 이름 조회를 지나 `std/vec.fy`의 AST 복사 중 phase 8 code 10으로 멈춘다. 다음은 복사 노드별 반복 소유자 조회와 복사 비용 축소다. 근거: `evidence/candidate-catalog-ranges-20261009.json`, `evidence/candidate-dispatcher-admission-catalog-ranges-20261009.json`.

2026-10-09: 비한정 이름을 토큰 끝 위치로 구분하고 AST 매핑의 함수 소유자를 재사용했다. 긴 로컬 이름 17600개의 도달성 작업량을 50000 미만으로 검사하는 native 사례를 포함해 SDK 검사 3개 통과. dispatcher는 1048534바이트로 상한 안에 있으며 `bodies.fy` 복사까지 진행한 뒤 phase 8 code 10으로 멈춘다. 다음은 CLI의 사용자 AST 중복 복사 제거다. 근거: `evidence/candidate-ast-owner-cache-20261009.json`, `evidence/candidate-dispatcher-admission-owner-cache-20261009.json`.

2026-10-09: 사용자 AST·스캐너 소유자를 그대로 옮기고 SDK만 복사하는 in-place 경로를 opt-in CLI에 연결했다. SDK 교체본과 이동 공간을 먼저 확보해 그 전의 실패는 원본을 유지한다. native 검사 3개 통과. 실제 dispatcher 160파일·1048460바이트는 SDK compaction을 통과하고 AST 결합의 scanner 저장량 거부로 phase 2 code 10에서 멈춘다. 다음은 결합 scanner의 65536바이트 cooked 저장량을 중복 제거 등으로 줄이는 작업이며, 전체 joined AST/HIR/MIR와 stage1/stage2는 계속 남는다. 근거: `evidence/candidate-ast-owner-transfer-20261009.json`, `evidence/candidate-dispatcher-admission-in-place-20261009.json`.

2026-10-09: 결합 scanner의 같은 cooked 값은 하나의 Buffer로 공유하고 65536바이트 상한을 실제 소유량에 적용했다. 256칸 인덱스의 충돌·가득 찬 뒤 검색도 정확한 비교와 기존 work budget으로 처리한다. mutable record 인자의 scalar 고정 배열 필드 뷰를 연결하고, 분석량 초과 E0010 stream은 AST 결합에서 부분 성공 없이 거부한다. 개발 check·SDK driver build, native API 8개, 배열 뷰 정상 7개·거부 9개·C native 14회, 파일 토큰 4개 사례의 정확한 출력·종료 코드와 거부 3개·C native 2회가 통과했다. 실제 160파일·1045810바이트의 logical cooked 69949바이트는 소유량 41226바이트로 줄었고 scanner 진단은 0개다. 다음 blocker는 module 18 `module_cli.fy`에서 누적 AST entry 1049개가 기존 1024개 상한을 넘는 phase 2 code 10이다. 60/61/65/85/88번의 다음 작업은 상한 증가 없이 arena를 분할하거나 줄이는 것이며, 전체 joined AST/HIR/MIR·stage1/stage2는 남는다. Release 전체 재빌드·Linux는 미실행. 근거: `evidence/candidate-cooked-storage-20261009.json`, `evidence/candidate-dispatcher-admission-cooked-intern-20261009.json`, `evidence/candidate-file-tokens-cooked-intern-20261009.json`, `evidence/candidate-mutable-record-array-view-native-20261009.json`.

2026-10-09: AST를 하나로 합치는 대신 실제 arena 수를 계산해 원래 모듈 순서대로 묶는 `module_units`와 범위 결합 `join_range`를 추가했다. 개별 파일·전체 소스·joined arena 상한은 유지하며 사용자 선언을 삭제하지 않는다. 개발 check·probe build와 native API 8개 통과. 실제 dispatcher 160모듈·1046245바이트 전체가 12개 bounded AST owner로 결합됐고 식 152020개·entry 11509개를 유지했다. 개별 묶음의 식 최대 16234개·entry 최대 1017개로 기존 상한 안에 있다. 60/61/64/65/85/88번의 다음 단계는 원래 module/declaration 좌표를 유지하는 묶음 간 이름·타입 연결이며, HIR/MIR·C 출력·최종 링크는 아직 연결되지 않았다. 현재 CLI는 계속 단일 join을 사용하므로 full-dispatcher 컴파일·stage1/stage2 완료를 주장하지 않는다. Release 재빌드·Linux·push는 미실행. 근거: `evidence/candidate-module-units-20261009.json`; 설계: `../architecture/candidate-compilation-units-v0.md`.

2026-10-09: `module_unit_bind`가 각 묶음의 선언과 imported 타입·값 참조를 원래 module/declaration/span 좌표로 소유한다. 같은 provider의 별칭은 같은 target으로 합치고 다른 모듈의 같은 선언 번호는 구분한다. 전체 소스와 metadata를 먼저 비교해 묶음 밖 provider의 변경도 거부하며 private 타입·함수와 generic parameter 경계를 유지한다. 개발 check·probe build와 native 회귀 5개 통과. 실제 160모듈·1046447바이트의 12개 묶음 모두 binding을 통과했고 식 152032개·entry 11513개를 유지했다. 묶음별 unique target 합계 2146개·최대 252개, 작업량 최대 134589로 기존 1048576 상한 안에 있다. 60/61/63/64/65/85/88번의 다음은 이 좌표를 실제 본문 이름·타입 검사와 provider layout/signature에 연결하는 것이다. HIR/MIR·C·CLI·stage1/stage2는 남아 있으며 Release·Linux·push는 미실행. 근거: `evidence/candidate-module-unit-bind-20261009.json`.

2026-10-09: `module_unit_interfaces`가 필요한 provider 타입 의존성만 따라가 필드·enum payload·함수 인자/반환 타입·effects/must-use/unsafe·foreign link name·repr(C)·resource abilities와 선언된 cleanup 권한을 소유한다. Generic 함수는 구체 signature가 없는 template으로 남긴다. Canonical 필드 조회는 resource 모듈 privacy를 유지하고 enum payload의 일반 필드 접근을 거부한다. 오래된 Bound도 현재 caller 바이트와 새 catalog 조회로 다시 확인한다. 개발 check·probe build와 native 회귀 6개 통과. 실제 160모듈·1046575바이트의 13개 묶음 모두 interface 수집을 통과했다. 묶음별 최대 shape 131·field 910·callable 198·type graph 401, projection 작업량 최대 10629로 기존 상한을 유지했다. 60/61/63/64/65/85/88번의 다음은 로컬 annotation·template과 본문 타입 검사를 이 interface에 연결하는 것이다. Target byte layout·transitive drop 효과·HIR/MIR·C·CLI·stage1/stage2는 아직 남는다. Release·Linux·push 미실행. 근거: `evidence/candidate-module-unit-interfaces-20261009.json`.

2026-10-09: `module_unit_interfaces`에 로컬 타입 표기와 canonical 타입 그래프를 연결했다. 선언 반환값·매개변수·지역 변수·필드·캐스트·명시적 타입 인자·바이트 반복문 타입을 원래 Parsed와 모듈 좌표로 해석하고 오류의 원본 위치를 유지한다. Generic 선언 안의 타입 표기는 구체화 대기로 표시하며, 해석된 타입의 provider 필드도 함께 수집한다. 개발 check·probe build와 native 회귀 8개 통과. 실제 dispatcher 160모듈·1046575바이트의 13개 묶음에서 타입 표기 10289개 해석, 69개 구체화 대기. 묶음별 canonical graph 최대 409개, shape 131·field 910·callable 198·projection 작업량 27474로 기존 상한을 유지했다. 다음은 generic 구체화와 지역 이름·식·본문 타입 검사 연결이다. 초기값·인자·반환식의 타입 일치 검사, layout·drop 효과·HIR/MIR·C·CLI·stage1/stage2와 Release·Linux는 남는다. Push 미실행. 근거: `evidence/candidate-module-unit-annotations-20261009.json`.

2026-10-09: `module_unit_generics`가 명시적 generic 호출의 구체 signature를 소유한다. 원래 provider module/declaration과 순서 있는 canonical 타입 인자가 같으면 재사용하고, 반환값·매개변수·effects/must-use/unsafe 및 식별 호출 위치를 유지한다. 원본 AST를 바꾸지 않는 타입 치환과 실패 시 graph rollback을 연결했다. 사용자 함수는 기존 builtin scalar만, 정확한 SDK Vec 템플릿은 명목 struct/resource도 허용한다. 개발 check·probe build와 native 회귀 13개 통과. 실제 160모듈·1048186바이트는 1MiB 안에 있고, 기존 AST 13개 묶음 중 처음 두 묶음은 각각 인스턴스 12·61개로 통과했지만 세 번째는 원래 module 37의 4953..4970에서 E4503으로 거부됐다. 다음은 기존 generic 인스턴스 상한을 유지하며 묶음 범위를 더 나누는 것이다. 그 후 본문 복제·중첩 구체화·재귀 검사·본문 타입 검사와 HIR/MIR/C/CLI 연결을 진행한다. Release·Linux·push 미실행. 근거: `evidence/candidate-module-unit-generics-20261009.json`.

2026-10-09: `module_unit_plan`으로 generic 수요를 확인해 과한 묶음만 더 나누도록 연결했다. 오류 10/E4503인 범위만 나누며, 일반 의미·privacy·stale 오류와 단일 모듈에서도 남는 오류는 원래 위치로 반환한다. 새 native 테스트 2개가 통과했고 이전 kernel/interface/type 회귀 13개도 소스·컴파일러·SDK·테스트 해시가 일치한다. 실제 160모듈·1048186바이트 전체는 분할 2회·검사 17회로 15개 범위에서 통과했다. 단위별 인스턴스 최대 61개로 기존 64개 상한을 유지했고, 단위별 항목 합계 476개로 호출 요청 1281개를 연결했다. Generic 본문 안의 14개 요청은 본문 복제 단계에 남는다. Canonical graph 최대 433·shape 131·field 907·callable 164, 이름 binding 작업량 최대 146311·interface 작업량 최대 81469. 초기 E4503 압박은 해소했으며 다음은 본문 복제·중첩 구체화·재귀 검사다. Graph ID는 owner 내부 값이므로 묶음 간 중복 제거는 구조적 타입 정체성을 비교해야 한다. 식·본문 타입 검사와 layout/drop/HIR/MIR/C/CLI·stage1/stage2는 남고 Release·Linux·push는 미실행이다. 근거: `evidence/candidate-module-unit-plan-20261009.json`.

2026-10-09: `module_unit_generics`가 함수 인스턴스마다 원래 provider 본문의 중첩 generic 요청을 읽어 전달된 타입 인자와 private helper를 연결한다. 직접·간접·타입 인자가 바뀌는 재귀와 깊이 32 초과를 거부하며, 함수 호출·정확한 SDK storage intrinsic·Vec<T> 타입 생성을 구분해 원래 식 ID로 소유한다. SDK Vec 생성식을 함수로 취급하던 빠진 처리를 수정했다. 최종 native 회귀 10개와 개발 check·probe build 통과. 실제 160모듈·1048277바이트 전체는 분할 5회·검사 23회로 18개 단위에서 통과했고 함수 인스턴스 최대 57개로 기존 상한을 유지했다. 단위별 함수 인스턴스 합계 592개, 초기 호출 1281개, 중첩 함수 요청 116개·intrinsic 594개·Vec 생성 100개를 원래 좌표에 연결했다. 원본 템플릿 구문의 14개 보류 표시는 유지하며 구체 인스턴스별 요청 기록을 따로 사용한다. 다음은 provider 함수와 SDK Vec cleanup 본문의 AST 소유, resource cleanup 구체화 및 암시적 drop 의존성 연결이다. 식·본문 타입 검사·묶음 간 구조적 중복 제거·layout/drop 효과·HIR/MIR/C/CLI·stage1/stage2는 계속 남는다. Release·Linux·push 미실행. 근거: `evidence/candidate-module-unit-generic-dependencies-20261009.json`.

2026-10-09: `module_unit_bodies`가 요청된 제네릭 함수 본문의 AST를 소유한다. 원래 모듈 위치와 토큰, hex cooked 리터럴을 보존하고 제공자 scanner는 모듈마다 한 번만 복사한다. 순서 있는 타입 인자, 매개변수와 반환 시그니처, 지역 변수·캐스트·명시적 타입 인자 표기의 canonical 타입을 소유한다. 중첩 호출과 SDK Vec 생성·storage intrinsic의 요청을 복제된 식 ID와 연결한다. 원본 AST와 복제 AST의 합산 크기가 상한을 넘으면 기존 분할 계획이 나눈다. Native 7개와 개발 probe check/build를 통과했다. 실제 dispatcher 160모듈·1048277바이트는 검사 37회·분할 12회로 25개 유닛에서 통과했다. 유닛별 본문 합계 615개, 타입 표기 3249개, 요청 연결 838개이며, 인스턴스 최대 57개·본문 작업량 최대 93777·합산 식 최대 16293개·entry 최대 951개로 상한을 유지한다. 다음은 SDK Vec cleanup AST 소유와 resource cleanup 컨텍스트 구체화 및 암시적 drop 의존성이다. 본문 식 타입 검사, 새 지역 타입 필드 shape 확장, 묶음 간 구조적 심벌 중복 제거, layout/drop 효과, HIR/MIR/C/CLI, stage1/stage2가 남는다. Release·Linux·push 미실행. 근거: `evidence/candidate-module-unit-bodies-20261009.json`.

2026-10-09: 필요한 resource와 SDK Vec 타입에 cleanup 컨텍스트를 연결했다. 컴파일러가 소유하는 정리 receiver(kind 10), 자원 타입, 선언된 effects/abilities와 Vec 원소 타입을 보존한다. 정리 AST는 필요한 노드만 복사하고 원본 위치와 토큰, 제네릭 호출과 지역 타입 표기를 연결한다. 가변 resource 필드 대여 계약을 넓히지 않고 지역 벡터를 조립해 새 계획을 반환한다. 단일 모듈에서 `take_slot<T>`의 템플릿당 16개 상한에 걸린 문제는 인증된 SDK 래퍼의 AST를 검사하고 cleanup에서만 `storage_take<T>`로 인라인해 해결했다. 원래 래퍼 선언과 unsafe 출처도 기록한다. Native 7개와 개발 check/probe build 통과. 실제 160모듈·1048472바이트는 검사 39회·분할 13회로 26개 유닛에서 통과했고, 유닛별 함수 본문 합계 625개·cleanup 컨텍스트 763개·정리 타입 표기 3624개·정리 요청 연결 595개다. 유닛별 최대 함수 57개·cleanup 59개·본문 작업량 495691·합산 entry 1015개로 상한을 유지한다. 다음은 암시적·중첩 drop 의존성과 정리 효과 전파다. 본문 이름·식 타입 검사와 인라인 출처 확인, 새 지역 타입 필드 확장, 묶음 간 중복 제거·layout·HIR/MIR/C/CLI와 stage1/stage2는 남는다. Release·Linux·push 미실행. SDK 소스·공개 언어/SDK ABI·DB·CI·러너 변경 없음. 근거: `evidence/candidate-module-unit-cleanups-20261009.json`.

2026-10-09: `module_unit_drops`로 소유 인터페이스의 타입 수준 drop 그래프를 연결했다. 필드와 Vec 원소의 공급자 좌표를 유지하고, Option·Result·배열·Vec에 포함된 값의 cleanup effects와 no-drop 능력을 전파한다. 참조·포인터에는 drop 의존성이 없으며, 재귀 Vec 관계는 작업량 상한 안에서 고정점을 구한다. 아직 수집하지 않은 nominal 타입은 `complete=false`로 두고 `required()`가 code 9로 거부한다. Admission은 기존 code 10 분할 경로를 사용한다. 개발 check·probe check/build와 native 회귀 7개가 통과했다. 현재 dispatcher 160모듈·1048472바이트는 검사 39회·분할 13회·26유닛을 유지했다. 유닛별 drop 엣지 합계 9878개, 최대 그래프 노드 425개·엣지 660개·작업량 6036이며, 합산 entry 최대 1015개로 기존 상한을 유지한다. 다음은 cleanup 지역 변수·암시적 식 타입과 이동 여부를 본문 검사에 연결하고 새 지역 타입의 필드·cleanup 수요를 완성하는 것이다. 인라인 출처 소비, 묶음 간 중복 제거·layout·HIR/MIR/C/CLI·전체 dispatcher native·stage1/stage2는 남는다. 이번 단위의 Release·Linux·push는 미실행이며 아직 연결되지 않은 전체 컴파일 경로의 완료를 주장하지 않는다. 공개 언어/SDK ABI·SDK 소스·DB·CI·러너 변경 없음. 근거: `evidence/candidate-module-unit-drops-20261009.json`.

2026-10-09: 요청된 generic 함수와 resource/SDK Vec cleanup의 지역 타입 수요를 본문 복제 전에 수집한다. 원래 provider AST의 명시적 타입 표기와 named record 생성식에서 canonical 타입을 찾고, 컨텍스트·module/declaration·원래 root/expression ID·span을 보존한다. 새 private 타입의 필드·cleanup과 그 cleanup이 요청하는 helper 본문까지 기존 pending queue에서 반복 수집한다. 호출하지 않은 template은 구체화하지 않으며, 잘못된 cleanup 지역 타입을 원래 위치로 보고하고 실패 시 graph 추가를 되돌린다. 개발 check·probe check/build와 native 회귀 8개가 통과했다. 실제 160모듈·1048472바이트는 39회 검사·13회 분할·26유닛을 유지했고, 함수 타입 수요 3289개·cleanup 수요 3624개로 합계 6913개를 연결했다. 유닛당 타입 수요 최대 443개·interface 작업량 최대 265344이며 기존 1MiB 작업량 상한을 유지한다. 모든 유닛에서 본문 복제 후 canonical graph 크기가 그대로임을 확인했다. Named constructor 발견은 native fixture에서 확인했고 이 dispatcher 입력에는 추가 constructor 수요가 없다. 다음은 일반 함수 호출 결과 추론과 본문 이름·식 타입 검사, 이동 여부를 반영한 cleanup 의무와 인라인 출처 소비다. 구조적 묶음 간 중복 제거·layout·HIR/MIR/C/CLI·전체 dispatcher native·stage1/stage2는 남는다. 이번 단위의 Release·Linux·push는 미실행. 설계·준비상태·진행표를 갱신했으며 공개 언어/SDK ABI·SDK 소스·DB·CI·러너 변경 없음. 근거: `evidence/candidate-module-unit-body-types-20261009.json`.

## Owned lexical binding for concrete bodies (2026-10-09)

`module_unit_names`, `module_unit_name_model` and `module_unit_name_walk` resolve
requested generic function and resource/SDK Vec cleanup ASTs. Current source
stamps, canonical graphs, ordered arguments and provider streams are checked
before publishing any result. Parameters, let/var declarations, payload patterns,
byte iteration, regions and the dropping receiver have separate symbol kinds.
Explicit annotation types remain canonical; inferred locals and payload types
remain unresolved. Entry/pattern/use tables and provider module/declaration
coordinates retain original expression IDs and spans.

Cleanup `this` is kind 10 with a resource type and no ordinary by-value parameter.
References record lexical unsafe function/block context; SDK storage names are
classified only in authenticated std::vec source. The safe `take_slot<T>` wrapper
has an unsafe intrinsic in its body: its source call can have `unsafe_context`
false while the separate request edge retains `inline_unsafe` provenance.
Binding records these facts without granting unsafe authority or performing
expression/flow checks. Ordinary unit bodies, expression typing, implicit drop
ownership/effect checks, inline provenance consumption, structural links,
layouts, HIR/MIR/C/CLI and full dispatcher native/stage1/stage2 remain pending.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-names-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus | 160 modules; 1048472 bytes |
| Admitted units | 26; 39 attempts; 13 splits |
| Per-unit function / cleanup totals | 625 / 763 |
| Per-unit symbol / reference totals | 3963 / 10161 |
| Maximum symbols / references | 260 / 648 |
| Maximum resolved-unit name work | 10639 / 1048576 |
| Maximum combined entries | 1015 / 1024 |

## 일반 함수 본문의 이름 바인딩 (2026-10-09)

module_unit_ordinary_names는 검사할 모듈 범위 내의 모든 비제네릭 함수 본문을 바인딩한다. 참조한 원본 스캐너 스트림과 함께 module_unit_body_ast와 module_unit_name_walk를 재사용한다. 임시 본문 AST는 폐기하지만, 불변의 해석된 이름 테이블은 원본 모듈 및 선언, 표현식, 매개변수·지역 변수 ID와 스팬을 유지한다. 명시적 정규 타입 표기 루트는 정확히 동일한 상한이 지정된 인터페이스 범위에서 읽으며, 인터페이스 범위 불일치와 오래된 소스 해시는 결과 반환 전에 거부한다. 제네릭 템플릿은 계속 보류하고, 요청된 제네릭 및 정리 바인딩은 module_unit_names에 남긴다.

단위 승인은 기존 16384, 16384, 1048576 상한 아래에서 결합된 일반, 제네릭, 정리 심볼, 참조, 이름 작업을 고려하며, 기존 128개의 활성 로컬과 깊이 64는 변경하지 않는다. 표현식 타입 검사, 일반 호출 결과 추론, 이동 인식 드롭 의무, unsafe 및 inline 출처 소비, 단위 간 구조적 중복 제거 및 레이아웃, HIR, MIR, C, CLI, 그리고 전체 디스패처 네이티브, stage1, stage2는 미완료 상태로 남는다. 공개 언어 및 SDK ABI, SDK 소스, DB, CI, 러너는 변경하지 않는다.

근거: `evidence/candidate-module-unit-ordinary-names-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Ordinary functions / symbols / references | 1175 / 7479 / 48136 |
| Maximum combined symbols / references | 723 / 4094 (each <= 16384) |
| Maximum combined name work | 751853 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Observation execution timeout | Initial 300s run timed out after 23 units; final 600s selection passed all 26 |

## 일반 함수 호출 결과와 필드·cleanup 수요 (2026-10-09)

fairy 컴파일러의 module_unit_calls는 선언된 호출의 canonical 반환 타입과 매개변수, 제공자 및 원래 식 좌표를 기록한다. 또한 이름 바인딩 이후의 효과, unsafe, foreign 플래그도 기록한다. module_unit_call_closure는 아직 수집하지 않은 일반 함수 시그니처를 기존 필드·cleanup 수집 경로에 반복해서 전달한다. cleanup helper 호출은 더 많은 결과 타입을 요청할 수 있다. 지역 변수로 함수 이름을 가린 호출와 오래된 입력은 거부된다. 제네릭 호출, 허용된 builtin, variant 생성식는 분리되어 유지된다. 기존 arena 상한는 변경되지 않으며, 반복 횟수는 64회, 호출 수집 작업량은 1048576로 제한한다. 완전한 식 및 인수 타입 대조, 추론된 지역 변수 타입, 소유권과 효과, unit HIR/MIR/C/CLI, 그리고 셀프호스팅은 모두 아직 보류 중이다.

공개 언어/SDK ABI·SDK 소스·DB·CI·러너 변경 없음.
근거: `evidence/candidate-module-unit-call-results-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Per-unit declared / specialized call-site totals | 11189 / 2622 |
| Per-unit variant / closed-builtin totals | 610 / 4 |
| Maximum closure rounds / work | 2 / 64; 305763 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Native closure-chain fixture | New call-result fields and cleanup-returned resources close after at least 3 rounds |
| Array reference regression | Kind-24 element links use right; a declared call beneath a cast retains its callee binding |
| Remaining expression consumers | Argument/operator matching, inferred-local symbol types, ownership and unsafe/inline checks |
| Deferred checks | Release, Linux, full dispatcher native and stage1/stage2; no push |

## 알려진 값의 지역 타입 추론과 반환값 검사 (2026-10-09)

`module_unit_values`는 일반 함수, 요청된 제네릭 함수와 cleanup 본문에
불변 값·지역 변수·반환값 테이블을 연결한다. 제공자 모듈·선언, 원래 식·entry ID와
소스 위치를 보존한다. 선언된 호출 및 명시적으로 특수화된 함수 결과를 받아 지역
변수 타입을 추론하고, 지역 변수 참조와 괄호로 알려진 타입을 전달한다. 직접
리터럴은 지역 변수·반환 타입을 문맥으로 사용하며 정수 범위 오류를 유지한다.
타입이 정해진 반환값만 대조한다. 연산자·캐스트·필드·인수·variant·분기 합류와
builtin 결과는 보류 상태로 남기며, enum의 payload 없는 값도 값이 아닌 구문으로
잘못 거부하지 않는다. 기존 canonical ID를 보존하고 빠진 기본 타입만 별도 그래프에
추가한다. 오래된 입력은 결과 반환 전에 거부하고, 기존 arena와 64회 고정점·1MiB
작업량 상한을 유지한다. 상한 초과는 기존 유닛 분할 경로를 사용한다.

| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher input snapshot | 160 modules; 1048562 bytes |
| Admitted units / attempts / splits | 26 / 39 / 13 |
| Per-unit context / expression totals | 2563 / 186137 |
| Known / unresolved / non-value expression totals | 58300 / 68976 / 58861 |
| Local / inferred local totals | 11442 / 539 |
| Return / checked return totals | 4783 / 2876 |
| Maximum value work / graph | 754886 / 1048576; 427 / 4096 |
| Maximum combined entries / type syntax nodes | 1015 / 1024; 2242 / 4096 |
| Preserved local / iteration bounds | 128 active locals; 64 fixed-point rounds |


비교에는 기존 160모듈 입력 스냅샷을 사용했다. 새 값 분석 모듈은 개발 check와
native 실행의 구현 코드로 검사했으며, 1MiB 입력 스냅샷에 추가하지 않았다.
따라서 이번 관찰을 최신 자기 소스 전체의 dispatcher 컴파일로 취급하지 않는다.
다음은 연산자·인수 및 나머지 식 타입 검사다. 이동 여부에 따른 cleanup 의무,
unsafe·inline 출처 소비, 묶음 간 구조적 중복 제거·layout·HIR/MIR/C/CLI도 남는다.
공개 언어/SDK ABI·SDK 소스·DB·CI·러너 변경 없음. Release·Linux·전체 dispatcher
native·stage1/stage2·push는 실행하지 않았다.
근거: `evidence/candidate-module-unit-values-20261009.json`.

## 스칼라 연산의 타입 문맥과 진단 (2026-10-09)

`module_unit_value_scalar`가 지역 변수 초기화와 반환값의 정수 산술·비트·시프트,
비교·bool 논리·단항 연산 타입을 검사한다. 정수 리터럴 트리는 타입 문맥에 따라
구체화하며, 타입이 정해진 상대 피연산자를 먼저 참조하더라도 원래 소스의 실행
순서와 식 ID·위치를 보존한다. 직접적인 i32·i64 최솟값 단항식에는 별도 표식을
남긴다. 필드·캐스트·builtin의 타입이 아직 정해지지 않았다면 상대 리터럴도
보류하여 기본 i32 범위로 성급하게 거부하지 않는다. 값 계획마다 기본 타입 캐시를
한 번 만들어 반복 그래프 검색을 제거한다. 일반·제네릭·cleanup 문맥에서 기존
canonical ID, arena, 64회 반복 및 1MiB 작업량 상한을 유지한다.

| Observation | Result |
| --- | --- |
| Development check | 185 source files; 2298 MIR functions; passed |
| Selected Fairy native tests | 8/8 passed |
| Reference fixture acceptance / selected primary code and span | 8/8 matched; additional cascade output retained |
| Unsupported reference case | Experimental exit 2 and span, distinct from an E diagnostic |
| Integer range correction | E1013, replacing the earlier internal 1008 |

다음은 선언된 호출 및 제네릭 호출의 인수 타입·개수·이름표와 shared 재대여
기록이다. 나머지 식·제어·place·소유권, unsafe·inline 출처와 HIR/MIR/C/CLI는
미완료다. 전체 dispatcher 입력 관찰은 다음 인수 단위와 묶어 실행하며 이번
단위에서 새 전체 코퍼스 통과를 주장하지 않는다. Release·Linux·전체 dispatcher
native·stage1/stage2·push는 미실행이다. 공개 언어/SDK ABI·SDK 소스·DB·CI·러너
변경 없음. 근거: `evidence/candidate-module-unit-operators-20261009.json`.

## 호출 인자의 개수와 알려진 타입 검사 (2026-10-09)

Fairy C의 module_unit_argument_collect/types는 이제 제네릭, 정리, 일반 컨텍스트 전반에 걸쳐 불변 호출 및 인자 레코드를 추가하며, 제공자 모듈/선언, 구체 함수 인스턴스, 인자 순서와 원래 식 ID, 원래 소스 위치를 그대로 보존한다. 선언된 일반 호출과 명시적 제네릭 호출은 개수(E1010)와 알려진 인자 타입(E1006)을 검증하고, 컨텍스트에 따른 스칼라 리터럴/연산자와 범위(E1013)도 함께 확인한다. 인자 이름표는 ADR0396에 따라 같은 모듈의 공개되지 않은 일반 함수에만 허용되며, 같은 위치에 정확한 이름이 있어야 하고 그렇지 않으면 E1014가 발생한다. ref mut T를 ref T로 변환하면 공유 재대여가 기록되지만 소유권·저장 위치 검증은 아직 남아 있고, 알 수 없는 필드·캐스트·대여·내장 함수·variant 타입은 여전히 미해결 상태로 유지된다. 기존 arena·작업량·깊이·반복 횟수 상한은 변경되지 않으며, code 10일 때만 상한을 넘은 유닛을 분할한다.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2338 MIR functions; passed |
| Selected development native tests | 9/9 passed |
| Reference acceptance/code/span | 10 matching cases; one recorded callable-value difference |
| Fixed dispatcher input snapshot | 160 modules; 1048562 bytes |
| Units / attempts / splits | 26 / 39 / 13 |
| Invocation / checked invocation totals | 12590 / 6167 |
| Argument / checked argument totals | 29822 / 20527 |
| Shared reborrow argument records | 402 |
| Maximum calls / arguments | 919 / 2687; each bounded to 16384 |
| Maximum value work / graph | 737134 / 1048576; 427 / 4096 |


기존 160모듈·1MiB 입력 스냅샷을 재사용했다. 새 분석 모듈은 구현 코드로 빌드하며,
최신 자기 소스 전체의 dispatcher 컴파일을 통과했다고 주장하지 않는다. 호출의
반환 타입을 알아도 필드·캐스트·대여·variant·builtin 때문에 인자 타입이 아직
정해지지 않으면 해당 인자를 보류한다. 함수 이름을 정수 인자로 넘기는 사례는
기준 컴파일러가 허용하지만 후보는 기존의 값이 아닌 함수 이름 분류에 따라
E1006으로 거부한다. 이 사례는 일치한 10개와 구분해 기록했다.
다음은 나머지 식·제어·place 타입 검사와 이동 여부에 따른 cleanup 의무,
unsafe·inline 출처 소비다. 묶음 간 구조적 중복 제거·layout·HIR/MIR/C/CLI,
전체 dispatcher native·stage1/stage2도 남는다. Release·Linux·push는 미실행이다.
공개 언어/SDK ABI·SDK 소스·DB·CI·러너 변경 없음.
근거: `evidence/candidate-module-unit-arguments-20261009.json`.

## 원래 피연산자 타입을 유지하는 스칼라 변환 (2026-10-09)

module_unit_value_model.Expression는 이제 정규 캐스트 대상과 문맥상의 기대/결과 타입을 분리해 보관합니다. module_unit_value_collect는 소유한 제네릭/정리 주석이나 원래의 일반 소스 타입 오프셋에서 대상을 초기화하며, 프로바이더의 원래 표현식/타입 ID와 스팬은 유지하고 오래된 입력 검사는 그대로 둡니다. module_unit_value_scalar는 기존 scalar_casts::allowed 규칙을 사용하며, i64로의 확장과 정확한 i32 / c::int 연결을 포함합니다. 피연산자는 대상 문맥 없이 검사되므로, 접미사 없는 과대 리터럴은 여전히 E1013으로 실패합니다. 지원되지 않는 캐스트 쌍은 새로 만들어낸 E1052 대신 실험적 code2와 전체 캐스트 스팬을 유지합니다. 알려진 캐스트는 추론된 지역 변수, 스칼라 동료 연산자, 호출 인자, 반환값으로 전파됩니다. 알 수 없는 피연산자와 원시 포인터 변환은 필드/플레이스/unsafe 작업 전까지 보류됩니다. 기존 graph/arena/depth64/round64/work1MiB 경계는 변경되지 않았습니다. 전체 디스패처 관찰은 다음 필드/플레이스 단위로 미루며, 새로운 코퍼스 통과나 stage1/stage2, Release/Linux/push를 주장하지 않습니다.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2340 MIR functions; passed |
| Selected development Fairy native tests | 8/8 passed |
| Reference acceptance/code/span | 7/7 matched |
| Unsupported conversion | Experimental code 2 and full cast span; no invented E1052 |
| Retained bounds | 4096 types; 16384 expressions; depth/rounds 64; work 1048576 |


Full dispatcher observation is bundled with the next field/place unit; this unit
claims no new corpus pass. Field/place, raw-pointer casts and unsafe provenance,
move-aware cleanup, structural cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. Release, Linux, full dispatcher native, stage1/stage2 and push were not
run. Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-casts-20261009.json`.

## 제공자 좌표를 유지하는 필드 타입 검사 (2026-10-09)

module_unit_value_field는 소유 인터페이스에서 알려진 nominal 필드를 해석하며, canonical owner/type, provider 모듈과 선언, 원래 필드 인덱스를 원래 식 ID와 span과 함께 불변 값 테이블에 보존한다. shared/mutable receiver borrow는 nominal 필드 조회를 위해 unwrap하고, provider 밖의 resource 필드는 E0103으로 거부하며, 내부 헬퍼 6201은 consumer에서만 정규화한다. 알 수 없는 operand는 pending으로 남고, missing/nonaggregate/지원되지 않는 member 사용은 전체 식 span에 실험적 code2를 사용한다. 닫힌 builtin 필드는 ref str, byte/scalar view의 len/is_empty, fixed-array len, Result is_ok를 포함하며, scalar peer context, cast, 추론된 local, argument, return과 통합된다. 측정된 shape/name 비교는 기존 work budget에 청구되고, 유효한 알려진 return 불일치는 이제 E1009이며 local과 argument는 E1006을 유지한다. borrow/place/ownership/unsafe와 pointer-cast consumer는 아직 pending이고, 기존 arena/depth64/round64/work1MiB 한도는 그대로이며 code10 range splitting은 유지된다.

| Observation | Result |
| --- | --- |
| Development check | 188 source files; 2344 MIR functions; passed |
| Selected development native cases | 11 passed; 9 unchanged cases reused and 2 corrected-coordinate cases rerun |
| Reference acceptance/code/span | 9/9 matched, including imported field privacy |
| Fixed dispatcher snapshot | 160 modules; 1048562 bytes |
| Detailed observations | 24 units before the 900s process timeout; remaining 2 units passed separately |
| Units / admission attempts / splits | 26 / 39 / 13 |
| Field / checked field totals | 14031 / 6085 |
| Nominal provider field records | 6015 |
| Cast / checked scalar cast totals | 1789 / 52 |
| Argument / checked argument totals | 29822 / 25740 |
| Maximum value work / graph | 747467 / 1048576; 427 / 4096 |


The native reuse receipt retains the original 9-pass/2-failure report and proves
that the only test-file change corrected two expected declaration IDs from 1 to
2. The parser counts the module declaration; compiler/implementation/SDK/lock
identities and the other nine fixtures are unchanged. Both corrected cases were
rerun natively. No failed execution is labeled as a passing execution.
The first process admitted all 26 units, then completed detailed assertions for
24 before its 900-second deadline. The final two ranges (145..152 and 152..160)
passed with the same compiler/manifest/implementation. All ranges continuously
cover modules 0..160; the initial timeout is retained as a failed execution.
This combined observation uses the same bounded 160-module input snapshot; new unit
modules are compiled as implementation code. Full current-source dispatcher
native and stage1/stage2 remain pending. Borrow/dereference/index/slice and
place/ownership/unsafe consumers, raw-pointer casts, move-aware cleanup,
structural cross-unit links/layouts and HIR/MIR/C/CLI remain unfinished.
Release, Linux and push were not run. Public language/SDK ABI, SDK sources,
DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-fields-20261009.json`.

## 이름 있는 참조와 역참조의 타입·소유자 추적 (2026-10-09)

module_unit_value_place 프로젝트는 주로 scalar, nominal, resource 참조와 중첩 nominal 필드를 대상으로 합니다. 안전한 참조 borrow/dereference 타입은 추론된 로컬, 선언/제네릭 호출 인자, scalar 캐스트, 반환까지 도달하며, canonical shared/mutable 타입은 기존 4096 타입 그래프에 추가됩니다. place 메타데이터는 전체 로컬 owner root와 scope, expression projection ID, mutability, indirect access, resource-field 경계를 유지하고, 참조 alias는 bounded fixed-point 라운드에서 owner provenance를 전파합니다. immutable/nonplace mutable borrow와 mutable resource-field borrow는 대상 span에서 E1008을 사용하고, 지원되지 않는 shared nonplace borrow와 비참조 dereference는 전체 expression span에서 실험적 code2를 사용합니다. ref *r과 ref (x)는 참조 컴파일러에서 지원되지 않으므로 일반적인 reborrow 지원을 주장하지 않습니다. 필드 그룹화는 명명된 root를 유지하고, array/view/indexed borrow와 unsafe pointer dereference는 명시적으로 보류 상태입니다. 이는 lifetime/conflict/move 검증이 아니라 type/provenance projection입니다. 기존 program/work 1MiB, entries 1024, types 4096, activeLocals 128, depth 64, rounds 64는 변경되지 않았습니다.

| Observation | Result |
| --- | --- |
| Development check | 189 source files; 2350 MIR functions; passed |
| Selected development native cases | 11 passed; unchanged ten cases reused with a source/hash proof, corrected cleanup case rerun |
| Reference acceptance/code/span | 14/14 matched, including generic and cleanup provider context |
| Bounds | program/work 1048576; types 4096; entries 1024; active locals 128; depth/rounds 64 |

The unit only consumes initializer, call-argument and return expressions. Standalone
expression/discard typing, lifetime/conflict/move checking, array/view/index/slice
borrows, unsafe pointer consumers, cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. The standalone implementation receipt excludes corpus telemetry; the
separate full fixed-corpus observation is documented below. Full current dispatcher native and stage1/stage2, release,
Linux and push were not run. Public language/SDK ABI, SDK sources, DB, CI and runners
are unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-places-20261009.json`.


## Safe-reference corpus observation (2026-10-09)

The fixed dispatcher snapshot completed admission and all detailed assertions in
one development execution. Original expression IDs/spans, nominal provider field
coordinates, canonical borrow types and whole-owner root/scope metadata were
checked. The input snapshot hashes are unchanged from the field-unit observation;
new unit modules are compiled as implementation inputs outside this 1MiB snapshot.

| Observation | Result |
| --- | --- |
| Input / units | 160 modules; 1048562 bytes; 26 units |
| Admission attempts / splits | 39 / 13 |
| Borrow expressions / typed borrows | 2891 / 1422 |
| Dereferences / typed safe dereferences | 388 / 254 |
| Expressions retaining owner root/scope | 51532 |
| Checked arguments | 27324 / 29822; previous field-unit result 25740 |
| Checked fields / nominal provider records | 6486 / 6416; previous checked fields 6085 |
| Checked scalar casts | 52 / 1789 |
| Maximum value work / graph | 823678 / 1048576; 427 / 4096 |
| Detailed coverage | All ranges continuously cover 0..160; execution passed without timeout |

Array/view/index/slice borrows, standalone expression/discard typing, lifetime,
conflict and move checking, unsafe/pointer provenance, cross-unit links/layouts,
unit HIR/MIR/C/CLI, full current-source dispatcher native and stage1/stage2 remain
pending. Release, Linux and push were not run. Public language/SDK ABI, SDK source,
DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-places-corpus-20261009.json`.

## Indexed array and scalar-view values (2026-10-09)

module_unit_value_index는 알려진 스칼라 뷰와 허용된 고정 배열 인덱스 값을 정규 원소 타입으로 해석한다. 인덱스 피연산자는 usize를 요구한다. E1006은 피연산자 범위를 보존하고, 지원하지 않는 수신자는 수신자 범위에서 실험 코드2가 되며, 반환 불일치는 E1009로 유지된다. 인덱스 자리는 원래 소유자 루트와 범위, 식 투영을 그대로 두고, 뷰 가변성은 바인딩이 아니라 뷰 타입에서 나온다.

이름 있는 배열과 뷰 원소 대여는 정규 공유 및 가변 참조, 캐스트, 추론된 지역 변수, 인자와 반환으로 이어진다. 임시, 호출, 원시 포인터 루트는 대여 주소 정체성을 얻지 않는다. 중첩 배열은 현재 언어 계약 밖에 있으며, 오래된 내부 그래프가 표현할 수 있더라도 값 소비자가 거부한다. 일반 제공자와 소유 정리 문맥이 이를 재사용한다. 기존 한도는 그대로이고 컴파일러 로직은 전부 Fairy다. 배열 리터럴, 뷰 생성, 슬라이스, 명목 뷰 인터페이스, 수명과 충돌과 이동, 안전하지 않은 코드와 포인터, HIR/MIR/C/CLI와 stage1/stage2는 아직 남아 있다.

| Observation | Result |
| --- | --- |
| Development check | 190 source files; 2352 MIR functions; passed |
| Selected development native cases | 5 passed; four identical cases reused and the corrected provider cleanup case rerun |
| Reference acceptance / code / span | 12/12 matched |
| Retained bounds | types 4096; expressions/calls/arguments 16384; entries 1024; active locals 128; depth/rounds 64; work 1048576 |

This observation only covers known receiver types in initializer, argument and
return consumers. Declaration/shape admission remains separate: resource array
fields and nested array type annotations still need earlier interface rejection.
No new full-corpus observation is claimed; it is bundled with array/view/slice
construction. Release, Linux, full current dispatcher native, stage1/stage2 and
push were not run. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-indexes-20261009.json`.

## Canonical array literal projection (2026-10-09)

새 `module_unit_value_array` 모듈은 목록 리터럴과 반복 배열 리터럴을 정규 그래프 타입에 대해 검사한다. 문맥이 없는 첫 원소가 타입을 정한다. 명시적 배열 기대는 각 원소에 문맥을 부여하되, 배열 길이 불일치는 기존 지역/인자 E1006 또는 반환 E1009 소비자가 보고한다. 개수는 1..4096 범위의 리터럴이어야 한다. 지원되지 않는 개수는 개수 구간에서 code2를 내고, 빈 원소 배열과 지원되지 않는 원소 배열은 리터럴 전체에서 code2를 내며, 원소 불일치는 E1006, 스칼라 오버플로는 E1013을 내고 둘 다 원소 구간을 유지한다. 목록 간선은 한 번 수집되어 소스 순서로 평가되고, 반복 초기화는 한 번 평가된다. `module_unit_value_shapes`는 비어 있지 않은 비순환 명목 필드 그래프를 검증하며 허용된 인라인 배열을 포함하고, 자원·원시 포인터·대여·태그·중첩 배열 원소를 거부한다. 64 깊이와 1 MiB 작업 한도는 유지된다. 비교는 청구되고 정규 그래프는 4096으로 유지된다. 알려진 배열은 추론된 지역 변수, 인덱싱, 선언 및 제네릭 호출 인자, 반환으로 흐르며 제공자 표현식 ID와 구간은 그대로다. 알 수 없는 레코드 초기화는 아직 남아 있고, 일반적인 독립 표현식·수명·이동 완성은 주장하지 않는다. 뷰 구성, 슬라이스, 명목 뷰 표현, 이전 선언 거부, unsafe·포인터, HIR/MIR/C/CLI, stage1/stage2는 남아 있다. 새로운 전체 코퍼스 통과, 릴리스, Linux 빌드, 푸시는 없다.

| Observation | Result |
| --- | --- |
| Development check | 192 source files; 2355 MIR functions; passed |
| Selected development native cases | 5 passed; four identical cases reused and the corrected shape-rejection case rerun |
| Reference acceptance / code / span | 15/15 matched |
| Literal order / identity | Source-order list edges, one repeated initializer, canonical element and length retained |
| Bounds | graph 4096; literal elements 1..4096; work 1048576; active shape depth 64 |

The original resource-bearing struct negative fixture was rejected earlier by
reference declaration analysis with E1002. It was replaced with a legal
pointer-bearing struct whose array literal is rejected by the value-shape gate.
General declaration rejection, record construction, standalone expression,
loan lifetime/conflict/move, view construction/slices, nominal view interfaces,
unsafe/pointers and downstream HIR/MIR/C/CLI remain incomplete. Full corpus
observation is bundled with the next view/slice unit. Release, Linux, full current
dispatcher native, stage1/stage2 and push were not run. Public language/SDK ABI,
SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-arrays-20261009.json`.

## Scalar array views and slices (2026-10-09)

module_unit_value_view는 이제 있는 바이트, 논리값, 정수 보기 갈래를 모두 다룬다. 이름 붙은 스칼라 배열을 그대로 ref 또는 ref mut 하면 문맥에 맞는 정확한 보기 형을 쓰고, 맞는 보기 문맥이 없으면 배열을 가리키는 기본 참조를 그대로 둔다. 묶인 이름 배열이나 평범한 구조체 배열 칸 범위를 명시적으로 ref 하면 보기 조각이 된다. 이미 있는 보기의 보통 조각은 공유/배타 형과 별칭을 거친 전체 소유자 뿌리와 범위를 그대로 지킨다. 생략한 경계는 0번 원천 번호로 남고, 적은 경계는 받는값, 시작, 끝 차례로 usize이며, 반열린 경계는 나중 실행 때 확인하고 정적 E를 지어내지 않는다. ref view[..]는 지원하지 않는다. 컴파일러가 임시 서술자를 빌리지 않기 때문이다. 가변에서 공유로 줄이기는 호출 인자에서만 되고 지역 선언에서는 안 된다.

E1008은 고칠 수 없는 가변 빌림과 평범하지 않은 칸 하위보기의 대상 구간을, E1006은 경계/인자/지역 구간을 가리키며, code2는 받는값 또는 전체 빌림 구간을 그대로 둔다. 평범한 모양의 칸 소유자는 배열 칸 보기에 대해 검사한다. 경계 싸개는 값이 아닌 문법이고 원래 식 번호와 구간은 그대로 남는다. 한도 graph4096, work1MiB, locals128, depth/round64는 그대로다. 형과 출처만 다룬다: 수명, 충돌, 이동, 자식 세대는 아직 미완이고, 이름 보기 인터페이스 형 표현도 아직 남아 있으며 완료를 주장하지 않는다. 관측하지 않은 코퍼스 수치, 배포/리눅스/푸시, HIR/MIR/C/CLI나 stage1/stage2는 없다.

| Observation | Result |
| --- | --- |
| Development check | 193 source files; 2366 MIR functions; passed |
| Selected development native cases | 9/9 passed |
| Reference acceptance / code / span | 17/17 matched |
| Aliases / bounds | Whole-owner root/scope retained; omitted bounds keep zero IDs; explicit usize bounds typed in order |
| Call shortening | Mutable scalar views can shorten to shared call arguments; local declarations remain exact |
| Bounds | types 4096; work 1048576; active locals 128; depth/rounds 64 |

The implementation receipt covers scalar view families and contextual consumers.
Nominal view interfaces and nominal array-field borrowing remain pending, as do
record/control/variant/builtin and standalone expressions, earlier declaration
rejection, loan generations/lifetime/conflict/move, unsafe/pointer provenance,
move-aware cleanup, cross-unit links/layouts and HIR/MIR/C/CLI. The separate full
fixed-corpus observation is documented below. Full current dispatcher native and
stage1/stage2, release, Linux and push were not run. Public language/SDK ABI,
SDK source, DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-views-20261009.json`.

## Scalar-view corpus observation (2026-10-09)

The fixed dispatcher snapshot completed admission and all detailed assertions in
one development execution. The 160-module, 1048562-byte input was admitted as 26
bounded units with continuous coverage over 0..160, and the accepted source
manifest hashes were re-verified against the receipt
(`dispatcher-admission-cooked-intern-20261009.txt`, sha256
8281ef6c4761a3d6627f81a0050b9f967e3d9ee3851e16a1c76aee73ab5dcd05). The accepted
view families report 74 array expressions (32 checked), 473 index expressions (135
checked), 171 slice expressions (166 checked) and 10 recorded view origins, with
original expression IDs, spans and whole-owner roots retained. The existing 9/9
development native selection and 17/17 reference acceptance remain the
implementation evidence.

| Observation | Result |
| --- | --- |
| Input / units | 160 modules; 1048562 bytes; 26 units |
| Array expressions / checked | 74 / 32 |
| Index expressions / checked | 473 / 135 |
| Slice expressions / checked | 171 / 166 |
| View origins | 10 |
| Maximum value work / graph | 825705 / 1048576; 427 / 4096 |
| Retained bounds | work 1048576; types 4096; entries 1024; active locals 128; depth/rounds 64 |
| Detailed coverage | All ranges continuously cover 0..160; execution passed without timeout |

Nominal view interfaces and nominal array-field borrowing remain pending, as do
lifetime/conflict/move validation and the remaining expression families, unsafe
and pointer provenance, cross-unit links/layouts and unit HIR/MIR/C/CLI. Full
current dispatcher native, stage1/stage2, release, Linux and push were not run.
Public language/SDK ABI, SDK source, DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-views-corpus-20261009.json`.

## Nominal view interfaces and array-field borrows (2026-10-09)

module_types는 first 자리에 기존 명목 원소 타입을 두는 정규 공유(kind 17)와 가변(kind 18) 명목 뷰 인터페이스를 들고 있고, 정규 그래프로 검증한다. module_unit_interfaces는 각 서술자의 원소를 pending shape 큐로 따라간다. 명목 배열은 문맥 기대 타입이 맞는 공유/가변 뷰이거나 배열 범위를 명시적으로 대여할 때만 그런 뷰가 된다. 문맥 없는 그냥 `ref p.data`는 명목 필드에서 실험 code2로 거부하고, 문맥 없는 이름 있는 고정 배열은 뷰가 아니라 배열 참조를 그대로 둔다. 평범한 이름 있는 전체 소유자는 뿌리와 범위를 지키고, 원소는 제공자 모듈/선언 정체성을 유지한다.

필드 소유자는 field_owner로 다시 검사하고 원소 선택은 element_borrow 검사를 거친다. 불변 저장소에 가변 대여를 하면 대상 범위에서 E1008이다. 평범하지 않은 소유자의 원소/하위 필드의 가변 대여는 선택 범위에서 E1008이고, 공유 대여는 전체 식에서 실험 code2다. 평범하지 않은 필드 범위는 배열 필드 소유자 범위에서 E1008이다. 인덱스와 내장 필드는 원소 해석과 len/is_empty를 공유/가변 서술자로 확장한다. 한도는 types 4096, work 1 MiB, entries 1024, active locals 128, depth/rounds 64 그대로다.

이 항목은 타입과 출처 투영만 한다. 수명, 충돌, 이동, 세대 검사는 아직 미완이라 대여 충돌이나 소유권 완료를 주장하지 않는다. 선정 개발 native 사례는 12/12 통과, reference는 code/span fixture 19/19 일치, check는 소스 193개와 MIR 함수 2370개다. 처음 실패한 fixture 2개는 테스트 작성 오류로, 기대 선언 번호가 module 선언 1번을 반영하지 못한 것이며 receipt에 기록되어 있다. 새 전체 코퍼스 관찰은 다음 consumer와 묶는다. control/variant/builtin/standalone과 discard·독립 표현식 검사, 이전 선언 거부, unsafe/포인터 출처, cross-unit links/layouts, unit HIR/MIR/C/CLI가 남는다. 전체 현재 dispatcher native와 stage1/stage2, release/Linux/push는 실행하지 않았다. 공개 언어/SDK ABI, SDK 소스, DB, CI, 러너는 그대로다.
근거: `docs/ops/evidence/candidate-module-unit-nominal-views-20261009.json`.

## Canonical record construction (2026-10-09)

module_unit_value_record가 이제 레코드 집합 생성자 식을 받는다. 이름 있는 생성자는 정규 모듈/선언 정체성으로 대상을 찾으므로 `S{...}`는 새 타입이 아니라 정규 구조체나 자원을 재사용한다. 원시 이름이 지역 변수나 함수로 해석되는 생성자는 집합이 아니라서 식 전체에서 실험 code2로 남고, 모듈이 다른 자원이나 소유 집합 생성자는 선언한 이름 구간에서 E0103이다. 모양이 없으면 내부 invariant code12이고, 문맥 열거 레코드는 아직 code0으로 대기하며 지원 완료가 아니다.

필드 행은 레코드의 오른쪽 연결 목록을 한 번 훑어 모은 뒤 선언 순서와 무관한 소스 순서로 검사한다. 알 수 없거나 잘못된 행과 중복 행은 행 구간에서 code2, 빠진 필드는 생성자 전체 구간에서 code2, 값 불일치는 E1006, 스칼라 범위 초과는 E1013으로 값 구간에서 난다. 필드 wrapper는 state2/type0을 유지하고 필드 출처(제공자 모듈/선언, 소스 번호)와 기대 타입을 저장한다. 완성된 레코드는 정규 대상을 가진 state1이 되고 자리는 비며, 필드 없는 집합도 받는다.

알려진 레코드는 배열, 필드 접근, 인덱스, 호출, 반환, cleanup, 제네릭 소비자로 이어지고, 제공자 필드 좌표는 가져온 모듈과 제네릭 제공자 단위까지 남는다. 소유 SDK `Vec` 집합은 종류13 소유 집합 seed로 특수화를 기록하고, 배열 리터럴의 평범 모양 gate는 그대로다.

앞선 실행에서 주석 없는 ordinary 생성자 `B{...}`와 `Empty{}`가 frozen 인터페이스 그래프에서 빠져, 선정 native 10개 중 8개가 정상 통과하고 2개가 오류였다. 이제 module_unit_body_types::ordinary_records가 그래프 freeze 전에 그 생성자를 등록한다. 비제네릭 일반 함수를 훑어 생성자를 찾으며 함수의 public/private 종류로 거르지 않는다(도우미 ordinary_scan 자체만 private). transaction rollback, 3 context 생성자 demand mark를 쓰고, 템플릿은 만들지 않으며 원래 선언 ID와 구간을 보존한다. 오류 뒤 수정한 실행은 receipt에 남긴다.

| 검증 | 결과 |
| --- | --- |
| check | 소스 194개; MIR 함수 2376개; 통과 |
| native | 선정 개발 사례 11/11 (이전 10개 + body-types 회귀 1개) |
| reference | code/span fixture 15/15 (오류 9, 정상 3, 다중 파일 3) |

고정 160모듈 코퍼스는 이전 manifest 해시가 모두 맞게 복원되어 새 상대경로 manifest로 다시 측정했고, 그 관찰은 통과했으며 아래에 기록한다. 한도 types 4096, work 1 MiB, entries 1024, active locals 128, depth/rounds 64는 그대로이고 컴파일러 로직은 전부 Fairy다. 제어/variant/builtin/standalone과 discard·독립 표현식 검사, 선언·형태 오류의 조기 거부, loan 세대/수명/충돌/이동, unsafe/포인터 출처와 이동 인식 cleanup, cross-unit links/layouts, unit HIR/MIR/C/CLI가 남고, 전체 현재 dispatcher native와 stage1/stage2도 남는다. release/Linux/push는 실행하지 않았다. 공개 언어/SDK ABI, SDK 소스, DB, CI, 러너는 그대로다.
근거: `docs/ops/evidence/candidate-module-unit-records-20261009.json`.

## Record construction corpus observation (2026-10-09)

고정 dispatcher 스냅샷이 한 번의 개발 실행에서 admission과 모든 상세 단언을 끝냈다. 160모듈, 1048562바이트 입력이 39회 admission과 13회 split을 거쳐 26개 단위로 들어갔고 범위는 0..160을 연속으로 덮는다. 받아들인 레코드는 검사 450/전체 570, 제공자 필드 2420, ordinary 생성자 전용 demand 457, 전체 타입 demand 7370(함수 3289, cleanup 3624, ordinary 457)이다. 최대 value work는 1048576 중 845545, graph는 4096 중 427이고, 최대 interface는 295576(이전 283813)이다.

원본 manifest는 가변 module_types가 454바이트 늘어 1049016이 되어 재현되지 않았다. 고정 스냅샷은 현재 나머지 159개 파일과 예전 bf28c16b의 module_types 바이트로 복원했고, 이전 receipt의 160파일 해시가 모두 일치했다. 입력 바이트를 지우거나 한도를 늘린 것이 아니다. 기존 고정 스냅샷일 뿐 전체 현재 dispatcher 통과는 아니다.

| 관찰 | 결과 |
| --- | --- |
| 입력 / 단위 | 160모듈; 1048562바이트; 26단위 |
| admission / split | 39회 admission; 13회 split |
| 레코드 검사 / 전체 | 450 / 570 |
| 제공자 필드 | 2420 |
| ordinary 생성자 전용 demand | 457 |
| 전체 타입 demand | 7370 (함수 3289; cleanup 3624; ordinary 457) |
| 최대 value work / graph | 845545 / 1048576; 427 / 4096 |
| 최대 interface | 295576 (이전 283813) |
| 상세 범위 | 모든 범위가 0..160을 연속으로 덮음 |
Evidence: `docs/ops/evidence/candidate-module-unit-records-corpus-20261009.json`.

이 시점의 다음 작업은 control/variant/builtin/standalone과 discard·독립 표현식 검사, 선언·형태 오류의 조기 거부였다.

## Conditional blocks and statement consumers (2026-10-09)

`module_unit_value_control` projects ordinary and unsafe block tails and `if`
results into canonical value types. Both branches receive the outer expected
context; unequal results report E1006 at the alternative expression. A missing
`else` yields void after its condition and consequent are known. Return and
literal-final assert statements do not change ordinary if-branch joins. Derived
values carry no storage place; borrow-origin merging remains pending.

Owned statement consumers cover standalone expressions, discard, assert, while
conditions and scalar for conditions in ordinary, generic and cleanup contexts.
They preserve original expression IDs and spans. Bool contexts report E1007,
and the structural ADR0213 for-header restriction is charged against the same
1 MiB work budget. Assignment/post clauses, defer/burn, missing-return checks,
function-body tails, switch/variant/builtin consumers and declaration/shape
rejection remain pending. The existing inferred-void-local rejection also still
differs from reference semantic admission.

Verification: development check (195 sources, 2390 MIR functions); 19 reference
acceptance/code/span fixtures; development native selection 7/7, followed by
2/2 affected tests after the no-else deferred-tail gate correction, with the five
unaffected tests reused. The initial 6/7 native attempt assumed an assignment
post-clause was already typed; the fixture now exercises the admitted condition
consumer. These runs and their exact source identities are retained in
`docs/ops/evidence/candidate-module-unit-control-20261009.json`.

Types 4096, work 1048576, entries 1024, active locals 128, depth/rounds 64 remain
unchanged. No new full corpus, full current dispatcher, stage1/stage2, release,
Linux or push was run. Corpus observations are bundled with the next tagged
consumer unit. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged. Next: switch/variant payload typing and assignment consumers.

## Contextual variant construction (2026-10-09)

`module_unit_value_variant` now checks contextual `.none`, `.some(value)`,
`.ok(value)` and `.err(value)` against canonical Option/Result payload types.
Nested tags and nominal/resource payload identities are retained. Enum values
and named payload records retain canonical module/declaration identity and the
original provider variant/field coordinates. Payload rows are checked in source
order; wrappers and contextual callees remain nonvalues. Unsupported context,
arity, member/field, duplicate/missing fields and payload mismatches retain their
original spans; scalar overflow remains E1013 and argument labels remain E1014.

Development check passed (196 sources, 2399 MIR functions), selected native
cases passed 7/7 including SDK Vec and conditional regressions, and reference
acceptance/code/span fixtures matched 19/19. Types 4096, work 1048576, entries
1024, active locals 128 and depth/rounds 64 are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-variants-20261009.json`.

Switch/payload binding, propagation/builtin, assignment/post, defer/burn,
function-tail/missing-return validation, earlier declaration/shape rejection,
inferred-void-local admission, ownership/unsafe provenance and unit HIR/MIR/C/CLI
remain. No new corpus, full current dispatcher, stage1/stage2, release, Linux or
push was run; corpus work is bundled with switch consumers. Public language/SDK
ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Switch values and scoped payload bindings (2026-10-09)

`module_unit_value_match` now projects exhaustive canonical Result/Option/enum
switches in source order, with duplicate/missing variant and payload checks.
Pattern nodes remain nonvalues; their payload type and existing scoped symbol
root annotate every matching path and its local metadata. Unrelated same-name
symbols remain separate. Enum pattern fields retain original provider module,
declaration and field coordinates. Typed payloads feed inferred locals, fields,
call arguments and returns in ordinary, generic and cleanup contexts.

Result excludes no-tail final direct-return and literal-final assert(false)
arms from value joins; Option excludes only literal-final assert(false), and
user enums exclude neither. All-excluded built-in switches have void type.
Other arm mismatches report E1006 at the arm block, overflow remains E1013,
and unsupported scrutinee/arity/pattern/exhaustiveness retains original spans.
A standalone non-void function with only switch-contained returns is still
rejected by the reference's checked-return gate; the all-terminal native fixture
uses a void function and does not claim that missing-return analysis is done.

Development check passed (197 sources, 2404 MIR functions), selected native
cases passed 7/7 including SDK Vec, and reference acceptance/code/span fixtures
matched 19/19. Types 4096, work 1048576, entries 1024, active locals 128 and
depth/rounds 64 remain unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-matches-20261009.json`.

The fixed 160-module corpus observation completed and passed with added
switch/variant/payload metadata assertions: the 1048562-byte input was admitted
as 26 bounded units covering 0..160 continuously through 39 admissions and 13
splits, with verified source, compiler, test, lock and input hashes. Match totals
are 109/109 checked, 811 variant values, 222 bound payloads and 38 pattern
fields; corpus value totals are 186137 value expressions, 117985 known values,
11442 value locals, 4783 value returns and 8500 unresolved values, with maximum
value work 937515 of 1048576 and graph 427 of 4096. This is the existing fixed
input, not a full current dispatcher pass.
Evidence: `docs/ops/evidence/candidate-module-unit-matches-corpus-20261009.json`.
Propagation/builtin, assignment/post, defer/burn, function tails/missing returns,
earlier declaration/shape rejection, inferred-void-local admission, ownership,
unsafe provenance, cross-unit HIR/MIR/C/CLI, full current dispatcher and
stage1/stage2 remain. Release, Linux and push were not run. Public language/SDK
ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Recursive value dispatcher stack repair (2026-10-09)

The fixed 160-module corpus aborted on Windows with STATUS_STACK_OVERFLOW
(3221225725). A diagnostic generated-C trace placed the failure in module17
type_cli declaration11, near the depth-31 tail of the long else-if ladder.

`module_unit_value_scalar`'s evaluate function now routes its group, field,
index, safe-borrow and cast branches through separate Fairy functions while
preserving dispatch results and depth increments. In the same debug workspace
the MSVC `/Od` diagnostic scalar-evaluate frame measured 19384 bytes before and
4808 bytes after. Types 4096, work 1048576, entries 1024, active locals 128 and
depth/rounds 64 stay unchanged, as does the stack reserve.

The new 56-branch else-if regression fixture covers normal i64 values plus the
final bool type error E1006 with its original span. Development native cases
passed 13/13 (6 conditional, 6 switch, 1 SDK Vec); the new reference
acceptance/span fixtures matched 2/2, and stage0 error/normal exit codes stayed
3 and 0. Evidence:
`docs/ops/evidence/candidate-module-unit-value-stack-20261009.json`.

The full 160-module corpus re-run completed and passed, recorded in the fixed
corpus observation above. The full dispatcher, stage1/stage2, release, Linux and
push paths were not run. Public language/SDK ABI, SDK sources, DB, CI and runner
contracts are unchanged.

## Result propagation value consumer (2026-10-09)

A new `module_unit_value_result` module projects the canonical Result operand's
success type. The operand is evaluated with zero context, so the enclosing
success type never manufactures a Result for it; the operand and the enclosing
kinds must both be 6 with equal second (error) canonical IDs, while an enclosing
success type may differ. A non-Result operand reports E4102, a non-Result or
error-mismatched enclosing type reports E4103, and both point at a 1-byte span
at the operand's original end. An unsupported construction whose fallback is
code2 is promoted to E4102 to match the reference diagnostic priority, and other
diagnostics and budget errors pass through unchanged.

Ordinary and generic providers, nested Results and nominal or resource success
payloads feed inferred locals, fields, call arguments and returns, and no new
place root is created.

Development check passed (198 sources, 2410 MIR functions), selected native cases
passed 6/6 (4 new propagation cases, deep else-if, SDK Vec), and reference
acceptance/code/span fixtures matched 13/13 (10 errors, 2 single-file positives,
1 provider generic). Limits types 4096, work 1048576, entries 1024, active locals
128 and depth/rounds 64 remain unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-propagation-20261009.json`.

Result must-use, flow, move and loan/cleanup ownership remain separate follow-up
work. Builtin, assignment/post, defer/burn, function tails and missing returns,
earlier declarations, provenance, unit HIR/MIR/C/CLI, full current dispatcher and
stage1/stage2 remain. The full fixed corpus re-run is bundled with the next
builtin unit, so no new corpus pass is claimed here. Release, Linux and push were
not run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged.

## Raw-view and source-location builtin value consumers (2026-10-09)

`module_unit_value_builtin` preserves the closed builtin kind from name binding
and the exact lexical unsafe flag in the Values Expression builtin metadata.

Raw view builtins 9..12 and CStr kind 14 project their result types from three
arguments in source order: the owner must be a safe shared or exclusive borrow
whose mutability exactly matches the builtin, the data carries a byte or opaque
raw pointer with matching mutability, and the length is usize. Arity reports
E1010, a non-usize length reports E1006, a missing unsafe context reports E5101,
and an unsupported owner or data keeps code2 at the original span.

The source_location builtin 200 takes exactly zero arguments, yields refstr and
leaves the root at 0; E4401 covers reserved declarations, parameters, locals and
non-call use, and E1014 covers labels. Only the borrowed tail origin inside a
single normal or unsafe block is preserved; scalar and owned tails create no
place, and multi-branch origin joins and lifetimes remain pending. An inferred
raw text alias root updates to the original parameter root by fixed point, and
generic providers and cleanup keep source_location's original coordinates.

Development check passed (199 sources, 2420 MIR functions), and selected native
cases passed 7/7 (5 new builtin cases, deep else-if, SDK Vec) after two initial
failures: one fixture bug that expected raw builtin 12 where the canonical CStr
kind is 14, and one real bug where an unsafe block dropped the borrow origin.
Reference acceptance/code/span fixtures matched 19/19 (15 errors, 3 single-file
positives, 1 provider-generic-cleanup). Limits types 4096, work 1048576, entries
1024, active locals 128 and depth/rounds 64 are intact. Evidence:
`docs/ops/evidence/candidate-module-unit-builtins-20261009.json`.

The SDK storage consumer is a separate follow-up implementation and is not
claimed complete, and the full corpus re-run is bundled after the next SDK
builtin. Source_location string and MIR emission stay under unit values and
follow the full unit HIR/MIR/C/CLI, so only its type projection is complete and
no whole source_location CLI completion is claimed. Unsafe pointer allocation,
generation, provenance and loan/move remain. Release, Linux, push, full current
dispatcher and stage1/stage2 were not run. Public language/SDK ABI, SDK sources,
DB, CI and runner contracts are unchanged.

## SDK storage value consumer (2026-10-09)

`module_unit_value_storage` projects builtins 101..107: storage_size and
storage_alignment yield usize with no arguments, storage_take yields the exact
canonical element from one argument, storage_put yields void,
storage_element_address yields ptr mut void, and
storage_borrow/storage_borrow_mut yield ref kinds 7/8 or the matching scalar
and nominal slice descriptors for T[]. Arguments are checked in source order;
arity and storage operand type failures keep code2 at the whole original call
span, and a missing unsafe context reports E5101. Child expression diagnostics
retain their own source spans.

Only authenticated std::vec storage borrow slice arguments peel the element
for annotation demands; unsized by-value types stay rejected. Canonical
element IDs, slice flags and lexical unsafe facts survive generic dependencies
and owned bodies. The cleanup inline of take_slot<T> retains inline_unsafe.
Borrowed owner roots propagate through block tails and aliases; take, put and
element addresses introduce no safe owner root. Nominal slice elements use the
existing scalar-leaf struct graph including bounded inline arrays.

Development check passed (200 sources, 2433 MIR functions), selected native
tests passed 7/7 (5 storage tests plus raw alias and deep else-if
regressions), and reference fixtures matched 5/5. Native storage tests cover
scalar operations, nominal array-field views, moved resources and cleanup,
five malformed-metadata/unsafe mutations, and user/SDK-spoof rejection.
Candidate lookup code5 maps to reference E0101; candidate E6801 at the spoofed
module name corresponds to the stable reference invalid-standard-library
namespace failure. Evidence:
docs/ops/evidence/candidate-module-unit-storage-20261009.json.

The fixed 160-module observation planned 26 units and completed 0..60 across
12 units before the 600-second observer timeout; this is not a complete corpus
pass. The probe now accepts from=<module-index> at a planned unit boundary,
and the 60..160 suffix is being observed with the prior 900-second allowance.
Imported ordinary SDK view_u8/view_mut_u8 still fail closed because
module_signatures::build excludes those SDK function signatures; generic
view<T> storage value projection is covered.

Limits 4096 types, 1048576 work, 1024 entries, 128 active locals and 64
depth/rounds are unchanged. Assignment/post, defer/burn, function tails and
missing returns, earlier declaration and shape rejection,
loans/lifetimes/conflicts/moves, unsafe pointer provenance, cross-unit
HIR/MIR/C/CLI, the full current dispatcher and stage1/stage2 remain. Release,
Linux and push were not run. Public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged.

## Fixed corpus resumed observation (2026-10-09)

The earlier 600-second observation completed 0..60 across 12 units before
timing out. Source, input and compiler identities were re-verified, and
from=60 resumed the 60..160 suffix with a 900-second allowance. The two
observations cover the unchanged 160-module, 1048562-byte input continuously
across 26 bounded units; the plan admitted 39 ranges with 13 splits. Evidence:
docs/ops/evidence/candidate-module-unit-storage-corpus-20261009.json
(implementation commit 3080ead6).

All 1221 storage value calls were checked. Total builtin calls were 1225, with
1223 checked; the two remaining raw-view values remain unresolved rather than
accepted. Maximum value work was 974364/1048576 and maximum graph size was
427/4096. The fixed input contains no storage slice calls; selected native
tests separately cover scalar and nominal T[] storage borrows. This is an
observation of the fixed input archive, not a full current dispatcher or
stage1/stage2 promotion. Release, Linux and push were not run.

## Cached SDK annotation authority (2026-10-09)

`module_signatures::Program` carries a per-module vector_sources bool cache,
populated from the existing sdk::authentic result at metadata construction.
After source_matches or validated source stamps, body type, owned-body and
interface annotation consumers reuse that fact by O(1) lookup. Per-body and
per-cleanup SHA rehashes in those annotation paths are removed, and each
authority read is charged as one unit of analysis work. This does not change
the existing generic-expansion or name-binding authentication paths.

Development check passed (200 sources, 2435 MIR functions) and the five
storage native tests passed 5/5 again. The fixture also checks ordinary
caller=false, authentic SDK=true and out-of-range module=false;
namespace/source spoof rejection is retained. Evidence:
docs/ops/evidence/candidate-sdk-annotation-authority-cache-20261009.json. The
complete fixed-corpus observation remains pinned to implementation commit
3080ead6; it was not re-measured for this cache-only change. Type projection
behavior is unchanged while some analysis work counters increase. Release,
Linux and push were not run; public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged.

## Ordinary SDK view signatures and unit storage bodies (2026-10-09)

`module_signatures::build` now collects authenticated std::vec non-generic
functions through the same canonical signature path as other ordinary
functions. The ten shared/mutable byte, i64, i32, u32 and usize view functions
retain their declaration coordinates, public visibility, parameter and result
types, effects, must-use, unsafe and foreign flags. Five scalar Vec
instantiations are retained in the metadata graph; ref/ref-mut parameters and
matching slice results stay canonical.

Development check passed (200 sources, 2435 MIR functions), selected native
tests passed 4/4 and reference fixtures matched 4/4. Caller tests cover exact
parameter/result IDs and E1006/E1010 caller spans, SDK unit tests check all
ten ordinary storage-slice bodies and owner roots against original SDK
coordinates, and the existing nominal array-field storage view remains
covered. Evidence:
docs/ops/evidence/candidate-module-unit-sdk-calls-20261009.json. Ordinary-call
origin summaries, loans/lifetimes/moves, assignment/post, defer/burn, function
tails/missing returns and cross-unit HIR/MIR/C/CLI remain. No new fixed-corpus
observation, full dispatcher, stage1/stage2, release, Linux or push was run
for this slice; the next corpus observation is bundled after the statement
consumers. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged.

## Assignment and post-clause value consumer (2026-10-09)

`module_unit_value_assignment` consumes assignment statements and post clauses
from the unit value walk. It resolves the canonical target expression with
zero context, then evaluates the right-hand side with the target's canonical
type as its expected context. A non-value or type-mismatched right-hand side
reports E1006 at the value span. The target must project a writable mutable
local, field, checked index or named admitted reference dereference; immutable
and non-place targets keep E1008 at the original span. Indexed struct
array-field replacement and empty-struct dereference are rejected with E1008.
Plain = accepts matching types, while the ten compound operators require one
of the five integer scalar types. Other compound targets retain code2 over the
statement.

Header init/post markers 1/2 require a direct scalar local target and
structurally scalar right-hand sides. Declaration initializers have the same
scalar header restriction. Violations retain code2 over the original clause
without its separator. Explicit allowed casts are admitted by both the unit
header consumer and the existing type_control predicate. Checks record
expected/actual canonical IDs, target and operator token coordinates, header
flags and the checked flag.

Development check passed (201 sources, 2443 MIR functions), selected native
tests passed 6/6 and reference acceptance/code/span fixtures matched 18/18.
Native cases check 31 assignments, ten compound operators across five integer
scalar types, plain bool assignment, mutable field/array/view/named-reference
paths, original provider generic and cleanup coordinates, and error and header
restrictions. Evidence:
docs/ops/evidence/candidate-module-unit-assignments-20261009.json.

Defer/burn, function completion and missing returns, inferred-void-local
admission, earlier declaration/shape rejection,
loans/lifetimes/conflicts/moves, unsafe pointer provenance and cross-unit
HIR/MIR/C/CLI remain. All consumer logic is Fairy. No new fixed corpus, full
dispatcher, stage1/stage2, release, Linux or push was run; the corpus
observation remains bundled after statement consumers. Public language/SDK
ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Function and cleanup completion gate (2026-10-09)

`module_unit_value_completion` applies the reference type checker's syntactic
completion gate to each function and cleanup body. It resolves the original
declaration body, requires a statement block without a top-level value tail,
and admits void results. Non-void bodies must contain a direct return, literal
assert(false), a completing unsafe/region body, or an if/else chain whose
branches complete. The parser's reverse statement links are walked directly
with charged work rather than repeatedly scanning indexed statements. Missing
completion retains code2 at the original whole-body span; Values publishes
checked_completion.

This is the reference's syntactic gate, not a CFG proof. A loop, plain nested
block or switch returning from every arm does not complete a non-void body. An
if without an else also remains insufficient. A branch with a value tail
follows the reference's semantic admission; actual return and termination
validation still belongs to the subsequent HIR/MIR path. Function and cleanup
bodies remain statement blocks, and top-level function value tails remain
parser errors. No syntax was added.

The walk now admits inferred canonical void locals and their aliases, removing
the earlier unconditional void-local rejection. Declared-type mismatches
retain E1006. Development check passed (202 sources, 2447 MIR functions),
selected native tests passed 6/6 (four new completion tests plus header and
provider/cleanup assignment regressions), and reference acceptance/code/span
fixtures matched 12/12. The stable Go check also admitted the void-inference
fixture. Native tests cover the syntactic boundaries, original error spans,
canonical alias identity, parser rejection and fail-closed exhaustion of the
unchanged 1048576 work budget. Evidence:
docs/ops/evidence/candidate-module-unit-completion-20261009.json.

Defer/burn, earlier declaration/shape rejection,
loans/lifetimes/conflicts/moves, unsafe pointer provenance and cross-unit
HIR/MIR/C/CLI remain. Type, work, entry, active-local and depth/round bounds
are unchanged. The completion logic is entirely Fairy. No new fixed corpus,
full current dispatcher, stage1/stage2, release, Linux or push was run. Public
language/SDK ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Defer, burn and validated resource abilities (2026-10-09)

`module_unit_value_terminal` consumes defer and burn statements from the unit
value walk. A defer is a direct checked invocation with a void result and
deferrable scalar arguments (bool and the five integers). Ordinary callables
must be non-foreign/non-template; a generic specialization must resolve a
checked concrete instance with matching owner, result and arity. The consumer
reuses invocation/argument tables and stays pending while a call or argument
is unchecked. Invalid targets keep code2 at the original statement, and
argument mismatches keep E1006 at the value span.

A burn requires a bare named local/parameter place with a canonical resource
declared by the current module, no drop ability and no resource-bearing
fields. Mutability is not required. Droppable resources, projected/non-place
operands, integers and foreign-module resources keep E1109 at the original
operand. Non-value expressions are rejected; unknown types remain pending.
Resource ability metadata is now validated declaration-locally by
resource_abilities::read before shape/cleanup publication, retaining E1107
spans for empty/repeated/malformed attributes, unknown/duplicate abilities and
key without store. Attribute names and argument texts are charged to the
existing work budget; other resource attributes remain with declaration
admission.

Development check passed (203 sources, 2453 MIR functions), reference fixtures
matched 18/18 (16 single-file and two module inputs), and native coverage
passed 5/5 (four terminal tests and one storage regression). The initial
native selection passed four tests; the imported-burn fixture used an illegal
direct provider constructor and was corrected to p::make, then passed alone.
Exact implementation/SDK/compiler/lock hashes and unchanged
test-file/preamble/function bytes were verified before reusing the four
earlier individual passes. Evidence:
docs/ops/evidence/candidate-module-unit-terminal-20261009.json.

This slice checks types and admission, not deferred-call MIR scheduling or
lifetime/loan/conflict/move validation. Earlier declaration/shape rejection,
ownership, unsafe pointer provenance and cross-unit HIR/MIR/C/CLI remain.
Existing type, work, entry, active-local and depth/round bounds are unchanged.
No new fixed corpus, full dispatcher, stage1/stage2, release, Linux or push
was run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged.

## Unit nominal field contracts and bounded defer lookup (2026-10-10)

module_unit_shape_contracts checks duplicate nominal fields, enum variants and
payload names (E0102), safe-borrow fields (E1003), resource-bearing struct
fields (E1002), invalid inline array field shapes (code2) and resource
abilities (E1107). Selected unused nongeneric nominal declarations are
retained and checked before body typing. The authenticated SDK Vec generic
resource keeps its existing exception. Canonical field validation preserves
original module coordinates and type spans. Other declaration attributes and
general generic nominal admission remain separate.

Defer invocation lookup uses binary search over the expression-ordered
invocation table, and checked defer/burn consumers are not re-evaluated. A
regression with 900 ordinary zero-argument calls and 900 defers stays within
the original work budget. Development check passed (204 sources, 2459 MIR
functions), reference diagnostic/token cases matched 14/14 and selected native
development tests passed 6/6: three shape/budget tests plus SDK nominal array
views, missing returns and imported generic/cleanup defer/burn regressions.
The initial budget fixture exceeded the unchanged 4160 parser-expression
bound. After correcting only that test body, it passed alone; byte-identical
preamble, other test functions, implementation, compiler, SDK, inputs and lock
justified reusing the five earlier passes. Evidence:
docs/ops/evidence/candidate-module-unit-shape-contracts-20261010.json.

Type 4096, work 1048576, entries 1024, active locals 128 and depth/rounds 64
remain unchanged. Loans/lifetimes/conflicts/moves, ordinary-call origins,
unsafe pointer provenance and cross-unit HIR/MIR/C/CLI remain. This slice does
not claim a new fixed-corpus observation, full dispatcher, stage1/stage2,
release or Linux validation. Public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged; nothing was pushed.

## Fixed corpus after statement and field consumers (2026-10-10)

The unchanged fixed 160-module corpus was freshly observed after SDK ordinary
view signatures, assignment/post consumers, syntactic completion, defer/burn
and nominal field contracts. The archived manifest, all source inputs and
compiler hashes were verified. Development probe check/build and every unit
observation passed across 26 continuous ranges covering 0..160, with 1048562
joined bytes. The current native build gate is the six individually verified
shape/budget, SDK view, completion and terminal regressions from commit
93920dd4.

The observation reports 2566 value contexts, 186158 expressions, 123386 known
values and 3113 unresolved values; 4781/4783 returns are checked. Maximum
value work is 987087 and maximum value graph is 427, within unchanged limits.
Unresolved values remain unresolved; these counts do not establish full source
compilation or ownership safety. Evidence:
docs/ops/evidence/candidate-module-unit-statements-corpus-20261010.json. Other
declaration attributes, ordinary-call origins,
loans/lifetimes/conflicts/moves, unsafe pointer provenance and cross-unit
HIR/MIR/C/CLI remain. Full current dispatcher, stage1/stage2, release and
Linux validation remain separate. Public language/SDK ABI, SDK sources, DB, CI
and runner contracts are unchanged; nothing was pushed.

## Restricted enum payload field rejection (2026-10-10)

Follow-up review exposed four enum payload admission mismatches against the
reference: safe borrow, resource, fixed array and an inert struct containing a
fixed array. The preserved 93920dd4 development probe accepted all four; the
reference rejected them. Nominal field contracts now reject direct enum
payload arrays with code2 and recursively require inert-copyable payloads,
including Option/Result and nested structs/enums. Resource, borrow, void,
non-inert aggregate and cyclic payloads retain E1004 at the original field
type span; scalars, C scalars and raw pointers follow the reference admission.
Recursive traversal retains the shared work and depth bounds.

Current development check passed (204 sources, 2460 MIR functions), reference
acceptance/code/span cases matched 21/21, and two native development tests
passed: the enum payload regression and the earlier 14 malformed-field cases.
The positive payload case checks plain struct, Option<struct> and
Result<struct,i32> together to exercise sibling traversal state. Evidence:
docs/ops/evidence/candidate-module-unit-enum-payload-shapes-20261010.json. The
preceding 160-module corpus observation remains evidence for commit 93920dd4;
it was not rerun after this focused rejection fix. Ownership, unsafe
provenance and unit execution integration remain. No language/SDK ABI, SDK
source, DB, CI or runner contract changed; no release, Linux or stage1/stage2
validation or push was performed.

## Checked provider bodies for borrow-origin analysis (2026-10-10)

module_unit_origin_inputs collects the checked value bodies required by
borrowed-result calls. The primary plan stays caller-owned; an additional
provider module is checked and owned once, including its private ordinary
forwarding calls. It uses the existing bind, call-closure and value phases
instead of guessing a return origin from the signature. Analysis frame/owner
IDs preserve original declaration coordinates and local lexical scopes.
Generic instance IDs are scoped to their owning unit, and canonical type IDs
remain in each unit graph.

Development check passed (205 sources, 2499 MIR functions), native tests
passed 4/4 and reference cases matched 4/4. Coverage includes repeated
provider-call caching and private forwarding, distinct generic parameter
owners, E1009 at the provider return-expression span, and matching generic
instance numbers that refer to different concrete types in two units. The
fourth test was appended after three passes; byte-identical preamble/functions
and compiler, SDK, implementation, input and lock hashes justified reusing
those passes. Existing work, type, context, expression, owner, invocation and
active-parameter bounds are retained. Evidence:
docs/ops/evidence/candidate-module-unit-origin-inputs-20261010.json.

This is the input prerequisite for return-origin inference; it does not
publish guessed origins or validate lifetime/conflict/move safety.
Static/unknown origins, branch joins and call substitution remain next,
followed by ownership, unsafe pointer provenance and cross-unit execution. No
new fixed corpus, full dispatcher, stage1/stage2, release or Linux validation
was run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged; nothing was pushed.

## Bounded syntactic paths for origin consumers (2026-10-10)

module_unit_origin_paths now provides bounded syntactic outcomes (fallthrough,
return/trap, break and continue) for a checked provider frame. It removes
definite dead statement/expression suffixes, joins if/switch arms, consumes
nested loop jumps and excludes a for post when its body can only break or
return. Loops retain a possible zero-iteration exit; no predicate correlation,
value-based branch elimination or at-least-once loop proof is inferred. Cast
and generic type-arena IDs are not traversed as expressions.

Development check passed (206 sources, 2517 MIR functions). Three native
development tests passed, covering return/trap and branch suffixes, loop/post
reachability and work-exhaustion failures. Reference grammar acceptance passed
for all 11 distinct marker fixtures; this is a syntax check, not semantic or
return-origin equivalence. The first native run passed two tests and failed
the loop test because its byte-view for fixture lacked let. Only that failed
fixture was corrected and rerun; byte-identical passing functions, preamble,
other tests, compiler, SDK, sources, inputs and lock hashes justify reusing
the other two passes. Evidence:
docs/ops/evidence/candidate-module-unit-origin-paths-20261010.json.

Dead intervals are inserted in ascending start order. Each maximum stores the
largest end in its sorted prefix, so binary search tests containment even when
intervals nest or share a start. All traversal, insertion, prefix construction
and lookup use the shared 1048576-work budget; depth, list and storage bounds
fail with original source coordinates rather than declaring a path dead.
Return-origin inference and ownership integration remain subsequent consumers.
No new corpus, full dispatcher, stage1/stage2, release, Linux or push was
performed; public language/SDK ABI, SDK sources, DB, CI and runner contracts
are unchanged.

## Lexical binding identity for origin consumers (2026-10-10)

module_unit_value_model.Expression now retains the original local binding ID
separately from Place.root. The value walk may forward a reference alias to
its initial owner, but that forwarded root cannot identify the alias being
reassigned. module_unit_value_collect preserves the resolver-owned binding
before projection, and module_unit_origin_inputs.binding_owner converts it to
a frame-scoped owner with bounds and kind checks. Nonlocal syntax returns
owner zero; invalid frame/binding IDs fail with code 12.

Development check passed (206 sources, 2518 MIR functions). Three native
development tests passed: reference reassignment retains the alias binding
while its value-place root is forwarded; shadowed aliases retain distinct
bindings and lexical scopes; and the earlier provider-unit generic-instance
collision regression still passes. Reference observations matched 2/2: the
reassigned return is rejected with E2103 at the original return r span, while
the shadowed nonescaping aliases are accepted. The candidate value-only plan
still retains the reassigned example for the subsequent ownership consumer;
this is not semantic-admission equivalence. Evidence:
docs/ops/evidence/candidate-module-unit-origin-bindings-20261010.json.

This change supplies lexical identity for return-origin inference without
guessing origin sets from current value-place roots. Static/unknown
distinctions, branch and assignment joins, call substitution,
loan/lifetime/conflict/move checks and unsafe provenance remain. Original
module/declaration coordinates, type IDs and bounds are retained. No new
corpus, full dispatcher, stage1/stage2, release, Linux or push was performed.
The internal candidate value model/accessor changed; public language/SDK ABI,
SDK sources, DB, CI and runner contracts are unchanged.

## Distinct bounded origin sets (2026-10-10)

module_unit_origin_sets provides canonical analysis-only origin sets.
Unreachable/no-result, Unknown and Static have separate handles; Unknown
absorbs a join, while Static remains a member when joined with a lexical
owner. Owner leaves preserve the global frame/local identity supplied by
origin inputs. Sorted merges deduplicate members and canonical interning makes
equivalent sets share a handle. These handles are not published HIR/MIR type
or origin IDs.

Development check passed (207 sources, 2534 MIR functions), and three native
development tests passed. They verify static/unknown/unreachable behavior,
order-independent canonical unions, duplicates, membership lookup, boundary
owner IDs, malformed handles and exhaustion even on a no-op join. Existing
1048576-work, 16384-owner, 4096-set and 32768-member bounds fail closed.
Evidence: docs/ops/evidence/candidate-module-unit-origin-sets-20261010.json.
This abstract set algebra has no Go semantic-reference equivalence claim.

Return-flow environments, branch/loop joins and argument substitution are the
next consumers. Loan/lifetime/conflict/move, unsafe provenance, cross-unit
execution and self-hosting remain. No new corpus, full dispatcher,
stage1/stage2, release, Linux or push was performed; public language/SDK ABI,
SDK sources, DB, CI and runner contracts are unchanged.

## Bounded return-origin flow and call substitution (2026-10-10)

module_unit_origin_returns now infers direct safe-reference/view return
summaries from checked original and provider bodies. Its flow interpreter uses
stable lexical bindings, evaluates assignments in source order, excludes
terminated branch environments, preserves Static in mixed sets and substitutes
unit-scoped callee parameter origins through argument values captured once in
evaluation order. Loop header environments converge through
fallthrough/continue backedges; exit environments include the zero-iteration
path and break edges. Probe iterations suppress return facts, and nested loops
retain independent edge depths. Scalar-for post execution follows live
backedges; byte-view iteration evaluates its input once.

Development check passed (211 sources, 2575 MIR functions), native development
tests passed 3/3, and reference acceptance/diagnostic/span observations
matched 17/17. Coverage includes static versus unknown, dead return suffixes,
overwrite versus branch joins, terminated branches, argument side effects,
recursive unknown summaries, retained local origins, five loop cases and
colliding generic instance numbers across provider units. Exhausted prior work
fails with code 10 and stale source metadata with code 12. Binary value lookup
charges actual probes and source/node/binding coordinates are rechecked before
analysis. Existing work/depth/round/storage bounds remain fail-closed.
Evidence:
docs/ops/evidence/candidate-module-unit-origin-returns-20261010.json.

A function-wide summary is not a diagnostic for each return: distinct
individually valid returns may join multiple parameter sources for a
subsequent call. Individual return-expression records and diagnostic/lifetime
consumers remain next. Invalid local origins remain identifiable in the
summary; forwarding a nonparameter/local origin yields Unknown rather than
inventing a caller owner. Tagged borrowed return payloads, region origins,
loan/subview facts, lifetime/conflict/move and unsafe provenance are still
pending. No new fixed corpus, full dispatcher, stage1/stage2, release, Linux
or push was performed. Public language/SDK ABI, SDK sources, DB, CI and runner
contracts are unchanged.

## Original return sites and stable origin records (2026-10-10)

Return-origin plans now own individual return records in addition to
function-wide summaries. Each record retains its analysis frame, inferred set
and original return-expression byte span. Per-frame ranges provide bounded
access without scanning unrelated returns. Summary-round records are discarded
before the next round; loop probes never publish records. The final plan
therefore contains only the stable round, live return sites and the original
expression coordinates required by the next diagnostic consumer.

Development check passed (211 sources, 2595 MIR functions), and four native
development tests passed, including all three preceding inference tests rerun
against the changed implementation. The new test verifies two individually
valid alternative returns, a mixed returned call at its complete expression
span, loop-return records without probe duplicates, and exclusion of a dead
return suffix. Reference observations matched 20/20: the three new cases prove
that alternative returns and loop returns are accepted while forwarding the
mixed summary through a single returned call reports E2103. The preceding 17
reference cases were reused only after compiler identity, original fixture
hashes and byte-identical original test functions were verified. Evidence:
docs/ops/evidence/candidate-module-unit-origin-return-sites-20261010.json.

This distinction prevents a future validator from rejecting a whole function
merely because its valid return paths use different parameters. Actual
E2102/E2103/unsupported diagnostics, related owner locations, tagged/region
origins, loan/subview/lifetime/conflict/move checks and unsafe provenance
remain next. The 8192-site cap, 1048576 work and existing frame/depth/round
bounds fail closed. No new fixed corpus, full dispatcher, stage1/stage2,
release, Linux or push was performed; public language/SDK ABI, SDK sources,
DB, CI and runner contracts are unchanged.

## Located static sources through return summaries (2026-10-10)

Reference probes showed that two distinct static literals must retain separate
origin locations: returning their conditional join reports E2103, and
forwarding a function summary preserves the callee literal locations.
Collapsing both literals to the abstract Static handle would erase that
distinction. Origin inputs now append kind-11 static owners with original
expression spans and a bounded mapping for each frame expression. A charged
open-addressed cache shares the same module/source-expression identity across
generic instances without scanning every owner. Static owners have no lexical
local/scope; frame-local ranges and parameter owner IDs remain unchanged.

Return flow now uses these located static owners for borrowed literals and
source_location calls. Call substitution preserves a static source unchanged
instead of replacing it with the caller parameter or call location.
Development check passed (211 sources, 2604 MIR functions), and all five
native development tests passed, including the earlier four tests rerun
against the implementation. The new test covers distinct literal coordinates,
forwarding and shared static identity across two generic instances. Reference
observations matched 24/24, including exact candidate-origin literal spans in
the four new cases. The previous 20 cases were reused after compiler and
embedded source-fixture identity checks. One initial native invocation refused
a stale lock after a source change; the current check refreshed the lock
before the successful run. Evidence:
docs/ops/evidence/candidate-module-unit-origin-static-locations-20261010.json.

The abstract set-algebra Static handle remains distinct from Unknown, but
actual inferred static leaves now retain input owner coordinates. Combined
lexical/static owners, expression mappings, hash slots, work, return sites and
depth/round caps remain bounded. This is still origin inference and input
preparation; E2102/E2103/unsupported diagnostic publication, tagged/region
origins, loan/subview/lifetime/conflict/move, unsafe provenance and cross-unit
execution remain. No new fixed corpus, full dispatcher, stage1/stage2,
release, Linux or push was performed; public language/SDK ABI, SDK sources,
DB, CI and runner contracts are unchanged.

## Direct return-origin diagnostic reports (2026-10-10)

An internal Fairy report collector now consumes the stable individual return
records. Returning a local/by-value parameter owner reports E2102, multiple
origins report E2103, and an unknown origin records unsupported analysis with
exit status 2. Valid alternative returns are checked separately instead of
rejecting their function-wide summary. Diagnostic records retain original
module indices and return-expression spans; related records retain origin or
candidate-origin roles and original owner spans, including provider-module
static literals.

Related locations are sorted and identical source diagnostics are deduplicated
across generic instances using a bounded open-addressed cache. The collector
carries forward the inference work count even when a caller passes a smaller
prior-work value. Exhausted work, allocation failures and inconsistent records
return a Problem rather than a successful report. Bounds cover 8192 return
sites/diagnostics, 16384 related rows per diagnostic, 32768 total related rows
and 16384 cache slots within the existing 1048576-work budget.

Development check passed (212 sources, 2631 MIR functions). Four selected
native development tests passed, covering local escapes, by-value parameters,
branch unions, distinct static literals, recursion, live/dead alternative
returns, provider coordinates, generic duplicate diagnostics and work
exhaustion/no-reset. Eight fresh reference comparisons matched exit status,
diagnostic code, module index, primary byte span and every related role/span
against the Go Stage0 compiler. Compiler, SDK, source, test, lock and embedded
fixture hashes were checked before accepting the result. Evidence:
docs/ops/evidence/candidate-module-unit-origin-diagnostics-20261010.json.

This completes direct return-origin diagnostic collection, not CLI publication
or the full dispatcher. Tagged/region origins,
loan/subview/lifetime/conflict/move, unsafe provenance and cross-unit
HIR/MIR/C/CLI integration remain. No new fixed corpus, stage1/stage2, release,
Linux or push was performed; public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged.

## Manifest return-origin checking entry point (2026-10-10)

The private candidate CLI now exposes --check-return-origins-file <manifest>.
It reuses the existing bounded manifest loader and parses every file once,
then runs signatures, binding, call closure, value typing, return-origin
inference and report collection against those exact source owners. Errors and
related origins print the original manifest filename and byte coordinates.
Direct semantic origin errors return 3; unknown origins, unsupported forms,
phase failures and loader/analysis limits return 2; a successful direct
return-origin report prints return origins ok.

An initial CLI probe reproduced a false success for a local borrow returned
inside Option. The coordinator now scans nested result types and imported
aggregate/payload shapes with a depth/work bound, refusing borrowed payload
returns until their origin flow is implemented. Valid borrowed Option and
nested Result payloads are also deliberately unsupported. Inert Option results
remain admitted. This boundary is recorded separately from semantic
equivalence; it does not count as matching the reference compiler for tagged
borrowed returns.

Development check passed (213 sources, 2635 MIR functions) and a fresh native
candidate build passed. Eleven CLI/reference comparisons matched exit status,
diagnostic codes, original files, primary spans and related role/spans,
including CRLF manifests, provider-first source order and a local aggregate
borrow. Two existing checker modes, four malformed-input refusals, four tagged
boundary cases and command help passed. Current compiler, candidate, SDK,
source, test, lock and fixture identities were verified. Evidence:
docs/ops/evidence/candidate-module-unit-origin-cli-20261010.json.

The command currently uses one bounded primary unit and checks direct return
origins. Tagged/region origin inference, loans, lifetimes, conflicts, moves,
unsafe provenance, split-unit execution and the complete project dispatcher
remain. Release, Linux and stage1/stage2 were deferred as planned. No push,
public Stage0 CLI/language/SDK ABI change, DB change or CI/runner change was
made.

## Single-leaf tagged return origins and wrapper storage (2026-10-10)

The Fairy origin pipeline now recognizes a single safe-borrow leaf through
nested Option/Result types, matching the current Stage0 admission rule.
Constructors preserve the selected payload origin, including success and
failure payloads; .none and inert selected payloads use a known-empty handle
distinct from Unknown and an out-of-scope binding. Empty payloads remain live
through variables, assignments, branch/loop joins and call substitution,
without inventing an owner or emitting an unknown-origin diagnostic. Multiple
borrowed leaves remain explicitly unsupported.

Origin inputs now distinguish a tagged wrapper storage owner from the owner of
its contained borrow. Shared forwarding of an Option parameter retains its
payload origin, while taking a reference to a local or by-value wrapper
records kind-12 storage provenance and reports E2102 at the original
expression, with the wrapper declaration as the related origin. All storage
aliases share the existing owner/work bounds. The unit type resolver and
borrow-place eligibility also admit refs to Option/Result so those storage
cases reach origin analysis. Generic and provider calls retain original
declaration and module coordinates.

The manifest command now admits these single-leaf wrapped returns. Development
check passed (214 sources, 2640 MIR functions); five selected native tests
passed, including known-empty algebra, invalid handles/work exhaustion,
direct-return regression and explicit refusal of multiple borrowed leaves and
pending ? flow. Twenty-four API/reference comparisons matched exit status,
diagnostic codes, module indices, primary spans and related role/spans. A
fresh native candidate build then matched 35 CLI/reference cases and passed
existing checker modes, malformed input, unsupported boundaries, inert tagged
returns and help. Current compiler, candidate, SDK, source, test, lock and
fixture hashes were checked. A source check refreshed the lock after the
native observer snapshot; the native report itself passed 5/5 with lock
verified. Only that snapshot differed, so unchanged compiler/SDK/source/test
identities were verified and the successful native output was re-observed
against the final lock without rebuilding those tests. The recovery and both
lock digests are retained in the receipt. Evidence:
docs/ops/evidence/candidate-module-unit-origin-tagged-20261010.json.

Initial native failures are retained as failures: one test helper missed its
statement terminator, a loop fixture included an unsupported empty statement,
and container-ref cases initially reached the old unit type admission refusal.
The corrected fixtures and the implemented type path were rerun; these earlier
attempts are not counted as passes. Match payload bindings, implicit Result ?
return origins, region origins, loans, lifetimes, conflicts, moves and unsafe
provenance remain. Split-unit execution, the full project dispatcher,
stage1/stage2, release and Linux remain deferred. No push, public Stage0
language/SDK ABI change, DB change or CI/runner change was made.

## Pattern payload origins and complete unsupported reports (2026-10-10)

Option/Result switch arms now seed borrowed payload bindings by lexical symbol
ID from the subject origin, and clear the binding at arm exit. Parameter,
static, local, nested and forwarded origins retain their original owner
coordinates. Taking the address of a by-value scalar pattern payload reports
E2102 with the pattern declaration as its related origin. Inert and
known-empty payloads remain distinct from Unknown.

The origin flow now retains every bounded unsupported return, binding,
assignment and match fact, ordered by frame and preceding return-site ordinal.
Reports preserve their feature and source span, deduplicate equivalent source
facts across instances, and publish semantic E2102/E2103 diagnostics when
present; otherwise the CLI prints all unsupported facts. Facts remain limited
to 8192 and the existing work budget. The Go reference was corrected
separately so assert(false) ends a borrow-flow path and scalar match bindings
have Local provenance (70dfc3c5); a stale bool-view premise was aligned with
ADR 0465 (d3a1eaae). The semantic package passed.

Development source checking passed (215 files, 2654 MIR functions), three
selected native tests passed, and 21 exact API/reference comparisons matched
codes, module indices, primary and related spans. A fresh locked candidate
build matched all 56 CLI/reference cases, plus two existing checker modes,
four malformed inputs, three tagged boundary checks and help. Compiler, SDK,
sources, tests, fixtures, candidate and lock hashes were verified. The earlier
incomplete unsupported projection is retained as a failed observation.
Evidence:
docs/ops/evidence/candidate-module-unit-origin-patterns-20261010.json.

Result ? success and implicit error-return origins remain next.
Region/loan/subview/lifetime/conflict/move and unsafe provenance, split-unit
execution, full project command integration, stage1/stage2, release and Linux
remain pending. No push, public CLI/SDK ABI, DB or CI/runner change was made
in this candidate slice.

## Result propagation return origins (2026-10-10)

The Fairy origin pipeline now admits a single safe-borrow leaf returned through
Result `?`: the success payload keeps its owner provenance, and the implicit
error return is recorded at the full `?` expression only when the failure
carries a safe borrow. Empty payloads, Unknown, parameter/local/static owners,
nested tagged wrappers, generic and provider calls, and loop bodies keep the
existing bounded rules. Multiple borrowed leaves remain explicitly
unsupported.

Development source checking passed (215 files, 2654 MIR functions) and three
selected native tests passed. Twenty exact API/reference comparisons matched
diagnostic codes, module indices, primary spans and related role/spans; a
fresh locked candidate build matched all 76 CLI/reference cases with the same
code/module/primary/related spans, and the legacy pair, four malformed inputs,
two tagged boundary checks and help also passed. Evidence:
docs/ops/evidence/candidate-module-unit-origin-propagation-20261010.json.

Region/loan/subview/lifetime/conflict/move and unsafe provenance remain
pending, together with split-unit execution, the full project dispatcher,
stage1/stage2, release and Linux. No push, public CLI/SDK ABI, DB or CI/runner
change was made in this candidate slice.

## Region capability type groundwork (2026-10-10)

`Region` is now a canonical compiler-owned capability type: `module_types.fy`
resolves the bare name to a single kind-19 node shared across modules with no
scalar, first or second metadata. The graph validator rejects forged
kind-19 records, and `module_bind.fy` recognizes the builtin type name.

Admission covers ordinary, direct, provider and scalar-generic fixed `Region`
input parameters, whose capability identity forwards across modules and calls.
Struct, resource and enum payload Region fields, Region results in ordinary
and specialized value contexts, and `let`/`var` copies including an inferred
`let` are rejected with E3103. A lexical region construct is explicitly refused
too: the unit value walker rejects a lexical region symbol (kind 6) with
unsupported code 2 pending Arena/source/allocation/reset validation, so no
incomplete local capability is published. Wrapper, reference and array Region
shapes stay outside candidate admission; the Go reference admits them, which
is an admission boundary rather than a proven contract bug.

Reference discrepancy: the Go reference still accepts a direct Region return
and `let`/`var` Region copies contrary to the Region language contract and the
E3103 entry. No Go change was made in this unit; the discrepancy is kept for
later alignment without a broad semantic-parity claim.

Final validation passed on the lexical Region refusal gate: root source
checking covered 215 files and 2654 MIR functions, four selected native tests
passed (three new tests plus the tagged return-origin regression), and a
fresh locked candidate build passed all 14 CLI observations. Current compiler,
product source, test, candidate and lock hashes were verified. The CLI keeps
phase failures at status 2; the Region refusals are not yet a complete
structured semantic diagnostic interface.
Earlier failed attempts and the fixed builtin Region binding are recorded in
the receipt. Evidence:
docs/ops/evidence/candidate-module-region-types-20261010.json.

Region.place and lexical region enter/reset/origins remain pending, as do
loans, lifetimes, conflicts, moves, unsafe provenance, cross-unit runtime, the
dispatcher, stage1/stage2, release and Linux. No push, public CLI/SDK ABI, DB
or CI/runner change was made in this candidate slice.

## Region.place and Region input origins (2026-10-10)

`Region.place` resolves against an exact authenticated `std::alloc` public
`AllocError` and a source-set match; `module_unit_interfaces.fy` owns the
implicit error shape. Kind 300 typechecks one operand on an exact Region receiver
and returns `Result<ref T, AllocError>`, rejecting `NeedsDrop`/resource
transitively with code 3102; borrow-containing placements remain unsupported
with code 2.

Fully inert scalar, raw pointer, struct, enum, tagged and array placement is
supported, and enum reference admission joined the owned type and borrow-place
path; a regression placing the same nominal struct through direct and Option
field traversal is fixed. `sdk_scopes` admits only the exact authenticated Arena
initialize/drop declarations (rules 70/71, `u32(ptr mut void)`), with SDK
bytes/ABI unchanged and unauthenticated or forged `fy_` symbols refused.

Region parameters carry a kind-13 origin owner distinct from ordinary borrowed
parameters and wrapper storage, and allocation derives Region provenance rather
than the placed value's origin. Direct, generic and provider forwarding,
argument reordering, `?`, match, nested wrappers and mixed-input E2103 keep their
original coordinates.

Validation: source check passed (217 files, 2658
MIR functions), native 3/3 passed (tagged regression and exhaustion), 17/17
API/reference comparisons matched codes, module indices, spans and related roles
on the authentic Alloc SDK, and a fresh locked candidate build passed CLI/reference 17/17 with refusals 8/8;
native and API hashes match the current sources and tests. A forged
`fy_std_arena_drop_v0` alias is rejected with code 5106/exit 2 where the reference
exits 0, holding the reserved SDK symbol authentication boundary; the retained
E5106 initial failure was a missing SDK Arena foreign-symbol admission. Evidence:
docs/ops/evidence/candidate-module-region-place-20261010.json.

Go discrepancy: the reference rejects a direct resource placement (negative 3,
exit 3) but accepts nested Option-resource (negative 4) and borrowed Option
(negative 5) placement at exit 0 that the candidate rejects as normative
non-inert inputs, so no broad Region parity is claimed. Runtime Region execution
and the lexical Region construct with E3101 region escape remain pending, as do
release, Linux and stage1/stage2; no push or public Stage 0 CLI/SDK ABI, DB or CI
change was made.

## Lexical Region source typing (2026-10-10)

The resolved lexical Region binding receives the canonical kind-19 capability
type. Its source must be an exact `ref mut std::alloc::Arena` expression, with
the Arena declaration owned by the authenticated SDK. The source expression is evaluated before
entering the nested block, and invalid sources report code 2 with the primary span
on the source expression rather than the binding name. The gate also refuses
unchecked or unresolved source consumers after a bounded fixed point.

Development source checking passed (218 files, 2664 MIR functions). Native 5/5
selected tests and 14/14 exact API source comparisons passed, matching Go codes,
module indices and source spans. A fresh locked CLI build passed all 14 exact comparisons on status and
span, and a local region-escape boundary case reports candidate code 2/status 2
for the unsupported borrow return against the reference E3101/status 3 with an
allocation related span. This unit covers typing and block analysis only; E3101
region escape, lifetime, loan and reset handling and runtime Region execution are
excluded, and a local Region borrow-return stays explicitly unsupported pending
the next unit. Current compiler, SDK, source, test, lock and candidate hashes were verified.
Public Stage 0 CLI/SDK ABI, DB and CI are unchanged; no push was made. Linux,
release and stage1/stage2 remain deferred. Evidence:
docs/ops/evidence/candidate-module-region-sources-20261010.json.

## Region escape and allocation provenance (2026-10-10)

Region local origins are now kind-14 facts seeded only inside the statement-17
block, with the capability identity kept distinct from a per-origin allocation
map. `g::Value` transports the origin and allocation map through variables,
assignments, match, `?`, arguments, branches and loops, and remaps generic and
provider summaries; a mixed summary keeps each leaf's allocation because caller
Region arguments may alias. A `RegionParam` uses its callee's allocation, while a
borrowed parameter forwards the actual argument allocation, and local origin
declarations for E2103 use the entire lexical Region statement. Provider
cross-module origin analysis is implemented.

E3101 reports the primary span at the return expression or `?` with the
allocation as related role 3 (module and span), without an E2102 cascade, and the
existing private `--check-return-origins-file` prints E3101 and the allocation.
Allocation sites are capped at 16384 immutable entries, and the provenance ledger
holds 4096 maps and 36864 rows inside the existing 1 MiB work budget.

Development source checking passed (219 files, 2700 MIR functions), five selected
native tests and all 21 exact API cases passed against the Go reference on code,
module and primary and related roles and spans, including alias Region arguments
(callee E2103 with the caller E3101 at the first callee site), direct and assigned
second sites, a provider-index-2 related allocation, implicit borrow-failure `?`
full-span reporting, a loop return and accepted known-empty Result/Option values,
with recursion's unknown-2 summary preserved. A fresh locked dev build passed all
128 CLI comparisons on status, code, file and primary and related roles and spans,
and the current native and API inputs match their hashes. Split-unit HIR/MIR and
runtime execution and the full dispatcher remain pending, together with runtime
region enter/reset, loans, lifetimes, conflicts, moves, unsafe provenance,
stage1/stage2, release and Linux. The public Stage 0 CLI and SDK ABI are
unchanged. DB and CI are unchanged; no push was made. Evidence:
docs/ops/evidence/candidate-module-region-escape-20261010.json.

## Owned Region lifecycle facts (2026-10-10)

The Region lifecycle is immutable per Values frame, and the Plan owns its Scope,
Placement, and Exit records. Each scope stores an authenticated Arena and source
type with the source expression, local owner, and source mode (1 = direct `&mut`
Arena local, 2 = named reference, 3 = evaluated expression). Type, source, and
statement IDs retain their context; the records survive disposal of parsed
sources and specializations. Placements retain owner, shared-borrow, Result, and
AllocError type IDs, and allow a RegionParam region ID of 0. Exit kinds are
fallthrough, return, propagate, break, and continue. Nested exits are grouped
inner first; loop exits record resets only for regions entered inside the loop,
and a trap records no reset. Cleanup boundaries are spans only; move-aware drops
and executable MIR or C enter/reset remain pending.

Source checking passed (223 files, 2754 MIR functions), and three selected native
tests passed, covering 16 lifecycle cases, 14 Region source cases and 21 escape
cases. Exact Go MIR region, placement, and reset kind/span comparisons passed
13/13; source and escape regression output also matched the retained reference
observations. Empty `return;` uses the statement span. Metadata is inspected
after parsed source owners are dropped. Three semantic-only forms remain outside
Go MIR execution: named sources, call sources, and RegionParam placement.
Evidence: `docs/ops/evidence/candidate-module-region-lifecycle-20261010.json`.
Remaining work includes loans, lifetimes, conflicts, moves, unsafe provenance,
executable Region runtime, split-unit/full dispatcher, stage1/stage2, release,
and Linux. No public CLI, SDK ABI, DB, CI or remote changes were made.

## Region borrow lifetime across lexical resets (2026-10-10)

Immutable allocation provenance now carries `Row.expired` per leaf. Joins keep
expiration from either path and select the expired branch's allocation site.
Region exit expires the environment and captured break/continue edge copies,
preserving origin identity. Re-entry into the same lexical Region does not revive
an old alias. A fresh value or `.none` replaces it, and a direct assignment target
does not read the old borrow. Origin Flow also evaluates void/scalar contexts
with Regions or placements. Expired or off-scope LocalRegion reads report E3101
with allocation role 3; identical return-expression diagnostics are deduplicated.
The private CLI label now describes borrow escape rather than only return escape,
and origin Flow handles empty `return;`.

Source checking passed (223 files, 2760 MIR functions), native tests passed 5/5,
and a fresh locked CLI build passed 17 lifetime cases (10 rejected, seven
accepted). The additional native API guard accepts a known abstract Static leaf
mixed with a live Region and still reports the expired Region. Unlocated Static
return candidates now report unsupported code 2 rather than an internal error or
a fabricated related span. API/CLI codes, module indices, primary spans and allocation roles/spans
matched 17/17 and independently specified source anchors. The prior 17 Region
input, 21 return and 16 lifecycle traces remain byte-identical. A failing positive
loop fixture exposed its retained old alias on allocation failure: the valid case
now clears the failure branch, and the original is retained as negative case 16.

The Go semantic reference accepts all 17 cases, including ten post-reset uses
forbidden by `docs/language/0003-regions-and-arenas.md`. These are intentional
normative differences, not Go parity. Coverage includes inside-scope use, input
Regions, overwrite before/after reset, nested Regions, break/continue, repeated
entry, allocation failure, branch joins, dead trap paths, aliases, and provider
and generic allocation coordinates. Evidence:
`docs/ops/evidence/candidate-module-region-lifetime-20261010.json`.
Ordinary owner loans/lifetimes, conflicts, moves, unsafe provenance, move-aware
cleanup, executable MIR/C Region runtime, full dispatcher, stage1/stage2, Linux
and release remain pending. Public CLI, SDK ABI, DB and CI are unchanged; no push.

## Owned borrowed binding liveness (2026-10-10)

Origin Inputs now owns a liveness plan for each Values frame. It retains borrowed
binding declaration spans, statement/expression source points, canonical sparse
maps, and the furthest future use for each binding. These records remain readable
after parsed sources and the primary typed plan are released. Frame indices keep
provider and generic specialization contexts separate; Arena/Region capabilities
are excluded from ordinary borrow liveness.

The reverse walk kills a previous reference at binding or direct overwrite,
clears outgoing liveness at return, replaces branch state with the union of its
arms, and removes switch payload bindings at the pattern boundary. Loops solve a
bounded fixed point with the nearest break/continue targets. Classic-for post
clauses stay outside borrow flow, following the existing scalar-only header rule
in `internal/semantic/for_loop.go`. Depth, work and sparse-map storage limits fail
closed. The checked flag distinguishes an analyzed empty map from an untouched
point.

Source checking and a locked development build passed. Six selected native tests
passed, including 16 liveness cases and the existing Region lifetime/escape/
lifecycle checks. All 171 shared source points match the Go liveness reference,
including binding and last-use byte spans; these exact expectations are retained
in the Fairy native test. The five prior native traces remain byte-identical, and
17 Region lifetime CLI cases match the prior exit/stdout/stderr observations.
The first regression invocation supplied a relative SDK path to isolated test
processes; the affected five tests were rerun with an absolute path. Evidence:
`docs/ops/evidence/candidate-module-loan-liveness-20261010.json`.

This unit retains liveness facts only. Ordinary owner loans, borrow conflicts and
owner-move diagnostics are the next step. Unsafe provenance, move-aware cleanup,
executable MIR/C Region runtime, full dispatcher, stage1/stage2, Linux and release
remain pending. Public CLI, SDK ABI, DB and CI are unchanged; no push.

## Whole-root owner loan conflicts (2026-10-10)

Origin Flow now carries frame-local loan sets alongside origins and allocation
provenance. Aliases, Option/Result payloads, branch joins and loop edges preserve
these sets; direct rebinding replaces them. Calls retain earlier arguments until
the call finishes, and borrowed return summaries select the caller's loan sets
by parameter. Callee-local loan IDs are excluded from published summaries.
Written view/array loops hold the iterator owner through break/continue/return
and release it at the written exit. Fixed-point probes emit no diagnostics.

Owner writes and incompatible explicit shared/mutable borrow creation report
E2101 with loan role 4 and last-use role 5. Field and element loans protect the
whole root. Owner writes publish one diagnostic per source site. Call argument
overlap also retains origin, mutability and source spans; the existing borrowed
type predicates cover scalar and struct views as well as ordinary references.
Conflict records remain owned after parsed sources and typed inputs are dropped.
The private return-origin CLI renders these new diagnostics. The stable Stage 0
CLI, SDK ABI, DB and CI are unchanged.

Source checking passed (230 files, 2875 MIR functions), and a fresh locked
development build passed. The Go reference, candidate API and candidate CLI
agree on 38/38 cases, including exact primary and related byte spans: 15 accepted,
23 rejected, 25 diagnostics. These source anchors are frozen in the native test.
Seven selected native tests passed. The five previous Region traces and the
16-case/171-point liveness trace remain byte-identical; the previous 17 Region
lifetime CLI observations also match exactly. Evidence:
`docs/ops/evidence/candidate-module-owner-loan-conflicts-20261010.json`.

Liveness now kills all named payload bindings and checks the root statement ID
before indexing. The guard accepts Values frame kinds 1 (concrete), 2 (cleanup)
and 3 (ordinary). Initial validation caught an omitted outer argument-block
conflict, an accidentally rewritten enum fixture literal, and a guard that
excluded ordinary frames; all were corrected before the final checks.

E2105 owner moves are next. Parent/subview generations, complete reference
access/store conflicts, full owned loan-fact publication, borrow MIR/C/runtime,
unsafe provenance, move-aware cleanup before Region reset, full dispatcher,
stage1/stage2, Linux and release remain pending. No push was made.

## Owner moves with retained loans (2026-10-10)

Origin Flow rejects direct resource binding, assignment, return, call arguments
and Option/Result constructor payload moves while their whole-root loans remain
live. Resource-containing tagged types and SDK Vec types are classified in their
own Values type graph. E2105 retains owner role 6, loan role 4 and last-use role
5 after parsed and typed owners are released. Grouping preserves the full moved
expression span. Aliases sharing one loan produce one diagnostic with the
furthest future use. Nested argument statements restore the outer statement
liveness context; fixed-point probes publish no diagnostics.

Source checking and a fresh locked development build passed. All 39 exact API
and CLI goldens passed: 16 accepted, 23 rejected, 24 diagnostics. Thirty-six
cases match Go exactly. Go accepts two later-argument moves while an earlier
shared argument is held; the candidate rejects them under the normative local
loan rule. One alias case selects the furthest use instead of Go's arbitrary
map iteration order. These three expectations use independent source anchors.
Eight selected native tests passed, and the previous seven native traces remain
byte-identical. E2101 CLI cases match 38/38, and Region lifetime CLI cases match
17/17. An initial Region test invocation omitted its alloc.fy input argument;
the affected five tests were rerun with the absolute source path. Evidence:
`docs/ops/evidence/candidate-module-owner-moves-20261010.json`.

Later borrowed arguments after an earlier move in the same expression are the
next bounded step. Parent/subview generations, complete reference access/store
conflicts, full owned loan-fact publication, borrow MIR/C/runtime, unsafe
provenance, move-aware cleanup, full dispatcher, stage1/stage2, Linux and release
remain pending. Stable CLI, SDK ABI, DB and CI are unchanged; no push.

## Moves before later borrowed arguments (2026-10-10)

E2105 now also consults the borrowed binding map after the moved expression,
alongside the enclosing statement map. A stored reference used by a later
argument stays live even when the call is its last use. This covers aliases,
resource consumption inside nested calls and Option borrowed payloads. A later
argument block that replaces the reference before using it releases the old
root; borrowing another owner and consuming after an earlier call are accepted.

Source checking and a fresh locked development build passed. Seven independent
API and CLI source goldens passed: four rejected and three accepted. Go and the
previous candidate accept all seven; the four rejections follow the normative
local-loan rule. Three selected native tests passed, and the existing owner-move
and owner-conflict native traces remain byte-identical. The previous 39 owner
move, 38 owner conflict and 17 Region lifetime CLI results remain exact. Evidence:
`docs/ops/evidence/candidate-module-owner-arguments-20261010.json`.

This does not implement general moved-owner availability or move-aware cleanup.
Parent/subview generations, complete reference access/store conflicts, full
owned loan-fact publication, borrow MIR/C/runtime, unsafe provenance, full
dispatcher, stage1/stage2, Linux and release remain pending. Release and Linux
validation were outside these local borrow-analysis units. Stable CLI, SDK ABI,
DB and CI are unchanged; no push.

## Retained shared-reference access conflicts (2026-10-10)

Mutable call arguments and stores through named reference dereferences, fields
or indices now reject E2101 when their origins overlap a live shared reference.
The single borrowed leaf of Option/Result retains its mutability. This includes
parameter origins without a local root loan, aliases, branch joins and
provider/generic return substitution. Last-use, rebinding and known-empty tags
release the restriction. Call-held owner-write diagnostics precede lexical
last-use diagnostics, matching Go's attempted-store location; explanation
suppression is restricted to the same frame and E2101 code.

Source checking and a fresh locked development build passed. All 27 Go/API/CLI
cases match exact primary and related spans; two selected native tests passed.
The 38 owner-conflict, 39 owner-move, seven later-argument and 17 Region lifetime
CLI observations remain exact, and the prior owner-conflict native trace remains
byte-identical. Evidence:
`docs/ops/evidence/candidate-module-reference-access-20261010.json`.

Parent/subview generations and receiver holds through effectful bounds and
store evaluation are next. General moved-owner state, move-aware cleanup, full
owned loan facts, borrow MIR/C/runtime, unsafe provenance, full dispatcher,
stage1/stage2, release and Linux remain pending. Stable CLI, SDK ABI, DB and CI
are unchanged; no push.

## Mutable descendant generations and evaluation holds (2026-10-10)

Mutable view slices and projected element/field borrows now retain frame-local
generation records with their selected origins and lexical parent. Descendants
inherit ancestor generations; copied parent descriptors do not acquire child
generations. Paths reject access through an overlapping ancestor or sibling
while that child is live. Alias, branch, loop and tagged payload flow preserves
generation sets; known-empty tags and last-use release the restriction.

Selected receivers remain borrowed through index and slice-bound evaluation.
Explicit fixed-array range/element borrows establish the root loan before
effectful selectors run. Named reference stores retain their receiver through
target index and RHS evaluation, including the direct-owner-read check while
allowing a read through the same reference. Descriptor replacement is distinct
from writing its referent. Pure length bounds remain accepted.

An owned descendant return-summary flag participates in the call fixed point.
The caller materializes a fresh generation at its call site; callee-local loan
IDs never cross frames. Provider and scalar-generic return paths retain this
flag. The owned API test reads direct/descendant/empty flags after parsed source
and primary typed owners drop, at function declaration IDs 2, 3 and 4 (the
module declaration occupies ID 1).

Source checking and a fresh locked development build passed. All 38 Go/API/CLI
cases match exact source diagnostics, and six selected native tests passed.
Previous CLI cases remain exact: shared-reference 27, owner-conflict 38,
owner-move 39, later-argument seven and Region lifetime 17. Four selected prior
native traces remain byte-identical. The initial owned-summary test assumed
function declaration IDs excluded the module header; its expectation was
corrected and that test alone was rerun. Evidence:
`docs/ops/evidence/candidate-module-subview-generations-20261010.json`.

SDK mutable storage-view generations, complete ordinary owner-read/access/store
coverage, full immutable loan facts, general moved-owner state, move-aware
cleanup, borrow MIR/C/runtime, unsafe provenance, full dispatcher, stage1/stage2,
release and Linux remain pending. Release/Linux validation was outside these
local analysis units. Stable CLI, SDK ABI, DB and CI are unchanged; no push.

## SDK mutable storage-view generations (2026-10-10)

Authenticated SDK mutable indexed storage borrows now create descendant
generations at their builtin source site. Their owned return summaries carry
the descendant flag to callers. Vec byte, i64 and plain nominal views retain
the parent restriction through aliases and Option payloads. Shared views remain
accepted; last use and known-empty tag replacement release the restriction.

Source checking and the locked development build passed. Nine SDK Go/API/CLI
cases match exact primary and related diagnostic spans; the selected native
test passed all nine cases. Existing subview 38 and shared-reference 27 CLI
cases remain exact against the same candidate image. Evidence:
`docs/ops/evidence/candidate-sdk-storage-generations-20261010.json`.

Complete ordinary owner-read/access/store coverage, full immutable loan facts,
general moved-owner state, move-aware cleanup, borrow MIR/C/runtime, unsafe
provenance, full dispatcher and stage1/stage2 remain pending. Release and Linux
checks were outside this local analysis unit. Public CLI, SDK ABI, DB and CI
are unchanged; no push.

## Owned historical loan records (2026-10-10)

The origin Plan now owns the converged round's root and descendant generation
records, including empty ranges for skipped frames. Private immutable accessors
publish each frame's records, mutability, lexical parent, borrow span and
selected origin leaves after parsed source and primary type owners drop.
Generic/provider frames retain their own ranges; frame-local loan IDs never
become cross-function identities. Root origins resolve through their owned
lexical owner; joined descendant origins use the owned canonical set table.

Each fixed-point round replaces the previous snapshot. Copies and clears are
charged to the existing work budget; total historical records are capped at
16,384. These historical analysis records include temporary/probe sites.
Retained-binding use/release histories and live environments remain required
for complete loan-fact publication.

Source checking and a fresh locked development build passed. Four native API
tests passed: source-owner drop, converged-round deduplication, branch-joined
origins and provider/generic frame separation. Their source spans match
independently selected source tokens. The initial generic test assumed caller
order; it now finds frames by module index and its selected rerun passed.
All 158 CLI cases match their existing references: SDK 9, subview 38, shared
reference 27, owner conflict 38, owner move 39 and later argument 7. The owner
suites use frozen contract goldens with the previously documented Go-reference
differences; they are not all direct Go equivalence claims. Evidence:
`docs/ops/evidence/candidate-owned-loan-records-20261010.json`.

Complete loan-use/release facts, ordinary owner-read/access/store coverage,
general moved-owner state, move-aware cleanup, borrow MIR/C/runtime, unsafe
provenance, full dispatcher and stage1/stage2 remain pending. Release and Linux
checks were outside this local analysis unit. Only the private candidate API
changed; public CLI, SDK ABI, DB and CI are unchanged. No push.

## Owned retained-borrow last-use and escape facts (2026-10-10)

The origin Plan now publishes a separate immutable table of retained safe
borrow facts. Each fact carries its frame, first retaining lexical binding,
source borrow coordinates, one origin leaf, shared/mutable kind, nullable last
use and valid return escape flag. Historical evaluation-only records remain
distinct: discarded call-bounded borrows never appear as retained facts.
Aliases extend the original record's use. Descriptor replacement and empty
tags stop extending the replaced loan; the last-use anchor describes its end.
Final loop passes contribute uses while probes do not activate retained facts.

Static literals/source locations and shared parent reborrows use fact-only
records without freezing an owner or adding descendant generations. Return-only
parameter facts are synthesized when no explicit loan site exists. Local or
ambiguous returns do not gain valid-escape flags. Published origin leaves and
function ranges remain owned after syntax and primary type owners drop. The
converged round replaces earlier facts; copies, merges and publication use the
existing work budget and 16,384 record/fact limits.

Source checking and a fresh locked development build passed. One selected
native test verifies frozen Go coordinates/kinds/last uses/escape flags for
27 fixtures; its 34 complete fact rows, including origin names and original
function/source coordinates, exactly match the current Go semantic API.
Fixtures cover unused/temporary references, aliases, replacement, tagged
payloads, mutable descendants, static and parameter returns, rejected local
escape, branch/loop uses, shared call conversion and generic providers.
All prior 158 CLI cases remain exact against their existing references;
owner move/argument suites retain their documented Go differences. Evidence:
`docs/ops/evidence/candidate-owned-borrow-facts-20261010.json`.

Hidden iteration loans and Region/raw constructor fact coverage, complete
ordinary owner-read/access/store checks, general moved-owner state, move-aware
cleanup, borrow MIR/C/runtime, unsafe provenance, full dispatcher and stage1/
stage2 remain pending. Release/Linux checks were outside this local analysis
unit. The private candidate API changed; public CLI, SDK ABI, DB and CI are
unchanged. No push.

## Iteration source typing before borrow analysis (2026-10-10)

For-in source expressions now participate in the value fixed point even though
the item has no written initializer. The origin pass receives a canonical
source type for direct slices. Exact element types and mutable item eligibility
are validated for fixed arrays and indexed views. Previously, mutable iteration
over `p[..1]` terminated the candidate with status 134; it now checks normally.

Source checking and a fresh locked development build passed. Twelve Go-accepted
iteration CLI cases pass. The selected native test passed five cases, including
shared/mutable slices, i64 items, wrong item width and immutable array rejection.
The 38 owner-conflict and 38 subview CLI cases remain exact. Evidence:
`docs/ops/evidence/candidate-iteration-source-types-20261010.json`.

Hidden iteration provenance/facts and Region/raw facts remain next. Release and
Linux checks were outside this local frontend fix. Public CLI, SDK ABI, DB and
CI are unchanged; no push.

## Hidden iteration loans and retained item facts (2026-10-10)

Fixed-array iteration now retains a hidden descriptor loan, and reference
items carry shared/mutable child provenance. Borrowed aggregate array fields
preserve their parameter origin; direct slices retain their own descriptor.
An unused reference item has no last-use anchor. Valid returned parameter
items escape; local items do not. Per-item liveness ends at each backedge,
while the descriptor remains held until loop exit. Shared items freeze a
mutable parent. Iteration conflict diagnostics use the held header span.

Locked source checking and a fresh locked development build passed. One
selected native test checks 18 iteration fixtures; all 23 complete fact rows
match the Go semantic API, including origins, borrow/use spans and escapes.
Six CLI cases match Go diagnostics and related locations, covering owner and
parent conflicts, release after exit and local/parameter return behavior.
The previous 27 borrow-fact fixtures still match all 34 complete rows. All
158 earlier CLI cases match their existing references; the owner suites
retain their previously documented Go differences. Evidence:
`docs/ops/evidence/candidate-iteration-borrow-facts-20261010.json`.

Region/raw constructor facts, complete owner access coverage, general move
state, cleanup, borrow MIR/C/runtime, unsafe provenance, full dispatcher and
stage1/stage2 remain pending. Release/Linux checks were outside this local
analysis unit. Public CLI, SDK ABI, DB and CI are unchanged; no push.

## Region and raw constructor retained-fact coverage (2026-10-10)

The existing constructor flow already matches the Go semantic borrow facts.
Raw str, byte-slice, mutable byte-slice and CStr constructors forward their
owner's records without creating a constructor-site loan. Explicit owner
borrows retain their original sites and aliases extend their uses. Region
placement records allocation provenance, but allocation itself is not a
retained loan. Valid returned RegionParam values gain return-only facts;
local Region and local raw-owner escapes remain rejected.

A selected native test now freezes 19 constructor fixtures and origin
rejection codes. All 14 complete fact rows match the Go semantic API,
including source/function identity, origins, kind, borrow/last-use spans and
escape flags. The source and primary owners drop before the immutable Plan
is queried. The test loads the real alloc SDK source as its first argument;
SDK-internal records are excluded from this root/provider comparison.

The reference uses AnalyzeExplicitFiles for the semantic analysis under test.
CheckExplicitFiles additionally fails its existing MIR verification for the
raw mutable descendant fixture with V1001, immutable byte-view slice shape
is invalid. That downstream behavior is recorded as a limitation; MIR/native
equivalence is not claimed. Evidence:
`docs/ops/evidence/candidate-constructor-borrow-facts-20261010.json`.

Implementation and lock are unchanged, so the preceding source check/build
and 158-case CLI regression remain valid without rerunning them. Release and
Linux validation remain deferred. Complete ordinary owner access, general
move state, cleanup, borrow MIR/C/runtime, unsafe provenance, full dispatcher
and stage1/stage2 remain pending. Public CLI, private API, SDK ABI, DB and CI
are unchanged; no push.

## Go reference ordinary owner-read correction (2026-10-10)

The borrow language contract rejects competing access while an exclusive
loan is live. The Go reference was missing ordinary root reads: seven new
baseline cases accepted scalar, aliased/tagged, call-held, array and aggregate
field reads. The reference now checks retained and pending mutable owner
loans in its path-access analysis. Storage-place roots are selected without
being treated as value reads, while selector callbacks still check reads.
Fixed-array length is type metadata. Existing descendant checks and E2105
resource-move diagnostics retain their behavior without duplicate errors.

All 25 selected boundary cases and the complete semantic package passed.
Exact source spans are asserted for scalar read, borrow and last use. A fresh
Go CLI build checks all 25 fixtures with their expected exit and sole error
code. Selected compiler checks pass for reference-selection execution, copied
mutable struct MIR and memory/disk partial-cache fallback.
Initial integration failures exposed direct owner metadata projections during
closed storage construction and raw pointer casts carrying safe call loans.
Metadata selection is now limited to direct fields of the same owner; blocks
and effectful arguments still check reads. Address casts check borrow creation
but raw results do not carry safe-loan state into later arguments. The reference
selection test now reads the live second owner through its selected reference.
The broader compiler run has 1,625 passed test events and 31 failed roots.
Thirty failed roots also reproduce with unchanged HEAD semantic sources:
the obsolete nonplain-owner rejection and SDK module-graph/native fixtures.
The remaining root was the reference-selection test and its corrected rerun
passes. These existing failures remain pending; the full suite is not green.
Evidence:
`docs/ops/evidence/candidate-reference-owner-read-20261010.json`.

This corrects Go first. The unchanged Fairy candidate still accepts nine
competing-read cases, including held selectors and effectful raw metadata.
Candidate guard wiring and exact diagnostic parity are next. Public CLI
acceptance changes for invalid competing reads; public API, SDK ABI, DB and
CI are unchanged. Release/Linux remain deferred; no push.

## Candidate ordinary owner-read guard parity (2026-10-10)

The Fairy origin flow now checks direct owner reads against live exclusive
root loans and call-held borrows. Alias/tagged retention, loop backedges,
branches and generic provider returns preserve the lexical owner identity.
Shared and expired loans permit reads, and the selected reference retains
access. Resource consumption keeps E2105 and its owner anchor.

A scoped source-node marker separates place selection from value reads.
Fixed-array length and direct metadata fields of the first raw/storage
constructor argument preserve their existing admission; callbacks and
effectful argument blocks remain checked. The marker is restored after each
expression and store-target evaluation. New traversals use the existing work
budget and historical loan bounds. The cleanup receiver Symbol is prepared
before borrowing its local Context; the corrected Go compiler exposed the
previous later-argument read of that Context.

Source checking and a fresh locked development build passed. All 31 selected
CLI cases and the selected native diagnostic test match complete Go error,
borrow, last-use and move-owner coordinates. The original nine competing-read
acceptance gaps are closed. Six added cases cover branch release/retention,
loop break/continue and a generic provider. Prior 158 CLI cases and six
iteration conflict cases match their references; the owner suites preserve
their documented Go differences. Three selected native API regressions match
64 fixtures and all 71 owned fact rows. Evidence:
`docs/ops/evidence/candidate-owner-read-access-20261010.json`.

Remaining referent/call access edges, general move state, cleanup, borrow
MIR/C/runtime, unsafe provenance, full dispatcher and stage1/stage2 remain
pending. The known SDK fixture module graph and obsolete nonplain-owner test
expectation also require repair. The full compiler suite was not rerun after
this candidate source change; release/Linux remain deferred. The private
Context shape changed; stable CLI, SDK ABI, DB and CI are unchanged. No push.

## Compiler scanner fixture closure (2026-10-10)

The 30 previously recorded baseline failing compiler roots are repaired.
Five scanner test workspaces now share the complete local import list,
including candidate_limits. Field replacement beside an inline array is
verified by execution instead of retaining an obsolete rejection expectation.
The C candidate storage cap follows the joined scanner's 1 MiB budget;
configurable byte-work exhaustion remains a separate diagnostic contract.
PDB folded procedure records count as aliases of one owned code range.
Exact byte partition, nonoverlap and release/debug RVA checks still pass.

The 31 affected roots (including the new sibling-storage case) pass across
focused runs: 29 passed initially and the two exposed boundary/PDB failures
passed after repair. Three additional C lexical/byte/combined-limit roots
also pass. Source-cap tests cover exact and one-byte-over input in both
native profiles, indexed/vector Fairy, standalone C and C-library paths.
The full compiler suite was not repeated. Evidence:
`docs/ops/evidence/candidate-compiler-fixture-repair-20261010.json`.

These fixture failures no longer block the next unit. Remaining access edges,
general move state, cleanup, borrow MIR/C/runtime, unsafe provenance, full
dispatcher and stage1/stage2 still require implementation. Full candidate
release and Linux remain deferred. No normative language documents, public
CLI/API, SDK ABI, DB or CI runner configuration changed; no push.

## Go held referent-read correction (2026-10-10)

Earlier exclusive call arguments now forbid later scalar dereference, record
field and borrowed-index reads through the same referent or its aliases.
Previously Go and Fairy both accepted nine invalid cases. Descriptor copies,
view length metadata, reads evaluated before the exclusive argument and ordinary
selected-store RHS reads remain admitted. Pure place selection stays separate.

The complete Go semantic package passes, including 21 selected read boundaries
and exact primary/loan/last-use coordinates. Five selected MIR/interpreter
integration roots and 24 fresh reference CLI fixtures pass. Evidence:
`docs/ops/evidence/candidate-reference-held-referent-read-20261010.json`.
The corrected compiler exposes 39 existing candidate-source argument reads in
five files. Preparing values before passing mutable descriptors is the next
source correction; candidate guard wiring follows. The candidate was not
rebuilt with these invalid calls. General moved-owner state, cleanup, borrow
MIR/C/runtime and dispatcher self-hosting remain pending. No push.

## Candidate held referent-read parity (2026-10-10)

Fairy now rejects competing field, dereference and borrowed-index reads during
earlier exclusive argument and mutable view-selector holds. Branches, loop
bodies, aliases, returned references and generic providers retain the selected
origin. Descriptor copying and view length metadata remain separate from reads;
shared holds, selected-store RHS reads and access after release remain admitted.

The corrected reference exposed 39 invalid argument reads in five candidate
files. The private parser append receives its pure Node value before mutable
descriptor arguments; 33 callers follow that order. Block depth, bound Symbols,
join kinds and returned origin state are prepared before exclusive acquisition.
Full source checking and a fresh locked development build pass.

All 32 new CLI fixtures and the selected native diagnostic test match complete
Go primary, loan and last-use coordinates. Existing 195 CLI comparisons pass;
owner suites retain their previously documented Go differences. Three selected
native APIs match 64 fixtures and all 71 owned borrow-fact rows. Evidence:
`docs/ops/evidence/candidate-held-referent-read-20261010.json`.

Remaining access edges, general move state, cleanup, borrow MIR/C/runtime,
unsafe provenance, the full dispatcher and stage1/stage2 remain pending.
Full candidate release and Linux stay deferred. The private parser append
signature changed; stable CLI, SDK ABI, DB and CI are unchanged. No push.

## Candidate module move-state checking (2026-10-10)

The Fairy candidate now tracks per-lexical-local move state instead of relying on
the Go owner flow. New Fairy-only modules `module_unit_move_model`,
`module_unit_move_walk`, `module_unit_move_control` and `module_unit_moves` mark
each local Live, Dead or MaybeLive and walk consuming arguments, returns, record
constructors, Option/Result `switch` arms and `?`, short-circuit operands,
reachable branch joins, classic scalar `for`/`while` and iteration backedges with
`break`/`continue`, reassignment restoration and scope transitions.

`--check-return-origins-file` merges these checks into its existing diagnostics as
E1101 use after move, E1102 second cleanup obligation, E1103 field move out of a
resource, E1104 inconsistent states at a branch join and E1105 inconsistent states
at a loop edge. E1103 keeps its field and owner anchors across provider modules;
E1102 keeps the cleanup receiver declaration anchor. `discard` counts as a read.
A classic `for` header fixes the loop-header coordinates its E1105 anchor reports,
and cast type syntax is not traversed as a value.

Non-indexed field assignment now reads its right-hand side before the store
target, matching Go, so a move in the right-hand side rejects the target read
with E1101; indexed targets read the receiver and index before the right-hand
side. The private fact API returns an immutable owned plan validated after the
source and values are released, under work 1048576, depth 64, facts 16384,
diagnostics 8192 and 1024 loop edges.

Source checking passed 240 files and 2987 MIR functions and a fresh locked
Windows development build succeeded. The new 58 CLI fixtures match complete Go
primary and related spans, two native tests pass (one 58-fixture diagnostics
check plus one six-owned-fact and exhausted-budget check), and the previous 227
CLI comparisons pass (195 prior plus 32 held reads); the owner suites keep their
documented Go differences. Evidence:
`docs/ops/evidence/candidate-module-move-state-20261010.json`.

Cleanup drop obligations and their effects with MIR/C/runtime wiring, outstanding
borrow and access edges, unsafe provenance, the full dispatcher and stage1/stage2
self-hosting, and general move semantics beyond this unit remain pending. Release,
Linux and the full compiler suite stay deferred; no Go source changed. The stable
CLI, SDK ABI, DB and CI runner are unchanged; a private candidate API was added.
No push.

## Candidate cleanup requests and drop-effect policy (2026-10-10)

The candidate now derives its cleanup requests from its own move state rather than
the Go owner flow. Each live local that reaches a scope exit, `return`, `break`,
`continue`, reassignment of a live owner, a `?` error edge, a consumed `is_ok`, a
discarded temporary or a match arm produces one request; moved and MaybeLive locals
produce none, and the cleanup receiver `this` is exempt. `module_unit_moves` exposes
these requests through a separate API that leaves the earlier move-fact API
unchanged. All 18 request fixtures match the Go error primary span, and the corrected
Go reference now handles the previously omitted match payload cleanup on
`break`/`continue`, retaining the payload when only a nested loop is left (commit
c32e1775).

A new Fairy-only `module_unit_cleanup_policy` walks canonical resource, Option and
Result types. It rejects an outer resource whose ability set omits `drop` before
inspecting its ordered fields, so an outer rejection supersedes inner effects; it
deduplicates the recursive non-drop blockers it visits; and it checks every nominal
owner's cleanup effect mask against the caller, including generic clones and provider
sources. `--check-return-origins-file` merges the result into its existing report as
E1108 (resource does not permit implicit cleanup) and E7102 (cleanup effect exceeds
caller authority), each carrying an `ability-declaration` or `effect-declaration`
related span under roles 11 and 12.

Go also gained the matching fixes for discarded temporaries and cleanup-body linear
temporaries (commit 0e171d70) and for cleanup-body fallthrough live locals (commit
6e3d269c), and the complete Go semantic package passes in 0.477s. The current-source
check passed 241 files and 3007 MIR functions under `--locked`, and the final
current-bootstrap locked development build passed.

The cleanup-policy CLI holds 58 fixtures that match complete corrected-Go primary and
related spans after adding two generic caller-authority fixtures (a missing-authority
rejection and an admitted case). All 285 previous CLI comparisons (58 move plus 227
prior) pass on the final candidate. Four native tests pass: 134 fixtures total (18
cleanup-request, 58 cleanup-policy and 58 move) plus a separate six-owned-fact and
exhausted-budget check, with the policy test confirmed at 58 fixtures and the three
unchanged tests reused. Evidence:
`docs/ops/evidence/candidate-cleanup-exit-requests-20261010.json` and
`docs/ops/evidence/candidate-cleanup-policy-20261010.json`.

Cleanup requests and the root/tagged cleanup declaration checks are connected.
Propagating cleanup effects through nested resource fields remains pending: a probe of
`resource Box { R inner; cleanup {} }` whose `R.cleanup` declares `@effects(console)`
is still admitted at the caller's implicit `Box` drop, the same omission the Go
reference shows, so the 58-fixture policy covers the root nominal effect mask rather
than a full recursive effect policy. Recursive non-drop blockers are done; implicit
field cleanup effect propagation remains before MIR/C/runtime elaboration. Normal and
abrupt cleanup event MIR, C and runtime elaboration, outstanding borrow and access
edges, unsafe provenance, and the full dispatcher and stage1/stage2 self-hosting
remain pending; this is not full compiler semantics. Release, Linux and the full
compiler suite stay deferred. The stable CLI, SDK ABI, DB and CI runner are unchanged;
a private candidate API was added. No push.

## Candidate recursive cleanup field effects (2026-10-10)

The candidate cleanup policy now propagates cleanup effects through owned resource
fields. `module_unit_cleanup_policy` `required` walks each nominal owner's ordered
fields in declaration order, following `Option` and `Result` payloads and
deduplicating canonical types through the shared seen set, so an outer resource whose
field cleanup declares an effect the caller does not hold is rejected with E7102
anchored at that field's `effect-declaration`. The earlier 58-fixture policy receipt
`docs/ops/evidence/candidate-cleanup-policy-20261010.json` covered the root-only mask
and is retained as the old receipt for that behavior.

Go gained the matching reference work: commit ad7c5d35 propagates cleanup effects
through owned resource fields, and commit 697e5da7 independently verifies transitive
cleanup authority at MIR `OperationDrop` sites. The complete Go semantic package
passes in 0.508s, the full MIR package in 0.051s, and the selected compiler tests
(`TestPrivateVectorNestedResourceDrops`, `TestPrivateVectorNestedResourceDropTrace`
and `TestBlockResultPrecedesLexicalCleanup`) in 1.588s.

The new recursive cleanup CLI holds 23 fixtures that match complete corrected-Go
primary and related spans, and the new native test matches 1/1. The eleven earlier CLI
suites were rerun on this candidate: 343 fixtures across cleanup policy, move state,
held reads, owner accesses, storage generations, subviews, reference access, loan
conflicts, owner moves, owner arguments and iteration conflicts, all matching, for 366
total. Five selected native tests pass: 157 fixture comparisons (18 cleanup request, 58
cleanup policy, 58 move and 23 recursive) plus a separate six-owned-fact and
exhausted-budget check. The current-source check passed 241 files and 3007 MIR
functions under `--locked`, and the locked development build passed. Evidence:
`docs/ops/evidence/candidate-cleanup-recursive-effects-20261010.json`.

Partial construction on the propagation error edge remains the open nested case: a
probe of `.ok(Box { .first = make(), .n = number()? })` whose `R.cleanup` declares
`@effects(console)` is accepted by both the corrected Go reference and the candidate.
When `number()?` returns an error before the `Box` is complete, the already evaluated
`R` temporary is not charged its cleanup effects; charging those effects on that error
edge is the next field-effect fix to consider. Normal and abrupt cleanup event MIR, C
and runtime elaboration, outstanding borrow and access edges, unsafe provenance, and
the full dispatcher and stage1/stage2 self-hosting remain pending; this is not full
compiler semantics. Release, Linux and the full compiler suite stay deferred. The
stable CLI, SDK ABI, DB and CI runner are unchanged; the private candidate API is
unchanged. No push.

## Candidate partial-owner cleanup effects (2026-10-10)

The candidate now retains cleanup effects for partially evaluated owners. The move
context carries a bounded 1024-entry pending-owner stack plus loop markers: a completed
record construction or call invocation truncates its pending frame, `?` and explicit `return` charge
every pending owner, and `break`/`continue` capture only the current loop's pending
owners, so an inner-loop `break` retains the outer pending owners. Go gained the matching
reference work in commit 52a424b4, retaining pending field and argument owners and
checking the actual abrupt exits; the complete Go semantic package passes in 0.542s and
the selected compiler tests (`TestByteForCleanupOnResultPropagation`,
`TestPrivateVectorSDKCandidateLoopPropagationCleanup` and
`TestBlockResultPrecedesLexicalCleanup`) in 2.766s.

The new partial-owner CLI holds 24 fixtures that match complete corrected-Go primary and
related spans: a no-drop `@abilities(store)` pending owner rejected with E1108, an
authority-correct case admitted, an earlier evaluated field versus a later owner not yet
constructed, repeated-owner dedup, and nested, tagged and generic cases. The old 366 CLI
fixtures were rerun on this candidate and all match, for 390 total. Four selected native
tests hold 123 fixtures (18 cleanup request, 58 cleanup policy, 23 recursive and 24
partial); the three earlier tests are valid reuse, and only the partial test was rerun
after its fixture was corrected from an invalid `@abilities(move)` (rejected with E1107,
which aborted the initial run) to `@abilities(store)`, passing all 24. The current-source
check passed 241 files and 3010 MIR functions with the lock unchanged, and the locked
development build passed. Evidence:
`docs/ops/evidence/candidate-cleanup-partial-owners-20261010.json`.

The partial field and call early-exit cleanup effect fix is complete. Split-unit cleanup
event MIR, C and runtime elaboration, outstanding borrow and access edges, unsafe
provenance, and the full dispatcher and stage1/stage2 self-hosting remain pending; this
is not full compiler semantics. Release, Linux and the full compiler suite stay deferred.
No new public or private API was added; the internal move context and Fact requests keep
the existing private API unchanged. No push.

## Candidate cleanup operand identity (2026-10-10)

Cleanup requests now carry a source-qualified operand identity. An operand is an
expression qualified by the evaluation frame it belongs to: an owned lexical root keeps
its nonzero binding with a zero expression, and a temporary keeps its nonzero original
source expression ID with a zero binding. The two identifiers are mutually exclusive and
a request that sets both is rejected before publication. The pending context stack
stores expression IDs, and the consumed `is_ok` and discarded-temporary paths retain the
operand's original source index. Generic clones keep the same source expression IDs
distinct through different concrete frame IDs.

The new `tests/module_unit_cleanup_operands.fy` native test passes 3/3: distinct repeated
partial operands, unambiguous roots and a consumed temporary, and generic clones
qualified by different frames. The earlier cleanup-request and partial-owner tests are
valid reuse on unchanged core code, so the five-test native run holds 18 cleanup-request,
24 partial-owner and 3 operand tests. The initial run failed 1/5 because an untyped
`.ok(make()).is_ok` fixture was malformed; replacing it with an explicitly typed boxed
Result factory made all three operand tests pass.

The 105 existing CLI fixtures (58 cleanup policy, 23 recursive effects, 24 partial
owners) were rerun on the current candidate and match the corrected Go primary and
related spans exactly. The current-source check passed 241 files and 3012 MIR functions
with the lock unchanged, and the locked development build passed. No Go source changed
this unit, so the earlier valid semantic and selected compiler checks are reused.
Evidence: `docs/ops/evidence/candidate-cleanup-operand-identity-20261010.json`.

Source-independent cleanup MIR publication and the C handoff remain pending, as do
executable split-unit functions and control flow; this is not full compiler semantics.
Release, Linux and the full compiler suite stay deferred. The stable CLI, SDK ABI, DB and
CI runner are unchanged; the private candidate Fact API was expanded. No push.

## Candidate source-independent cleanup MIR fragment (2026-10-10)

A private, source-independent cleanup MIR fragment now lowers the checked values into
owned canonical tables. `module_unit_cleanup_mir_model.fy` defines the `Program`
resource with its `Frame`, `Glue`, `Field`, `Operand`, `Drop` and `Problem` records;
`module_unit_cleanup_mir_lower.fy` builds them from the move and cleanup-policy output;
and `module_unit_cleanup_mir_verify.fy` revalidates them independently. The fragment owns
the complete canonical type table, frames with module/declaration/context, authority
bits and cleanup receiver, per-type recursive glues, ordered fields, unique
frame-qualified operands with their original spans, and drop sites with bounded exit
reasons. After lowering it needs no original `Parsed`, `Values` or provider
`Interfaces`. A hash index rejects duplicate canonical type descriptors and duplicate
operands.

`lower` rejects outstanding move diagnostics, runs the cleanup-policy gate, then
resolves each pending expression ID or binding ID against its frame before the typed
context is released. `check` confirms canonical descriptor uniqueness, void cleanup
signatures, a nonempty resource bound to exactly one matching cleanup receiver, the
owned nested and tagged effect union with recursive drop-ability checks and cycle detection,
and operands and drop sites that stay owned, droppable, inside the frame authority and
off the cleanup receiver. The contract is
`docs/architecture/candidate-module-cleanup-mir-v0.md`.

The new native test holds 4/4: provider type identity with ordered fields and distinct
frames, distinct partial operand values, a cleanup receiver exempt while a same-type
local is not, and thirteen negative mutations: out-of-range drop operand, out-of-range
operand type, removed caller authority, removed nonempty cleanup frame, changed provider
module, reordered field ordinal, removed nested drop ability, duplicate qualified
operand, recursive field cycle, invalid Result alternative, exceeded work budget,
non-void cleanup result and duplicate canonical descriptor. Twelve are rejected with
1001, the exceeded work budget with 1002, and the accepted copy returns 0. The
current-source check passed 245 files and 3076 MIR functions with the lock updated, and
the locked development build passed. Receipt
`docs/ops/evidence/candidate-module-cleanup-mir-20261010.json` records the four native
tests and thirteen reject guards; native report
`.cache/candidate-tagged/cleanup-operands-unit/mir-native-final.json`.

The fragment keeps semantic request order and does not attach drop sites to CFG blocks
or fix runtime instruction order; CFG attachment and the reverse-lexical, defer and
Region ordering are required before the C handoff. The model holds no ordinary function
bodies, CFG or local-execution definitions, so it does not yet carry all source
provenance, and this is not executable whole MIR or a new C pipeline. The existing
`module_program` C path still joins and is unchanged. The existing 105 CLI fixtures are
valid from the stage1 sources and code, unchanged, and no CLI route exercises the new
fragment. No Go, source SDK, DB or CI change; the private IR API is new. No push.

## Candidate owned module body IR (2026-10-10)

A private, source-independent body IR now owns the checked function and cleanup bodies as canonical tables.
`module_unit_body_ir_model.fy` splits them into a `Body` (typed expression nodes with their syntax links, statements with their assignment operators, locals, calls, arguments and Region scopes/exits, and the raw and cooked text pool) and a `Program` (the owned cleanup-MIR fragment, the bodies, and the layout/field/callable tables with one bounded name pool).
`module_unit_body_ir_lower.fy` builds them over `cleaning::lower`; `module_unit_body_ir_lookup.fy` recovers a canonical node or statement from its original source index by binary search; and `module_unit_body_ir_verify.fy` revalidates them. After lowering it needs no original `Parsed`, `Values` or imported `Interfaces` owner; this is a private API with no CLI route.
`lower` copies each typed expression node with its linking syntax node and appends its raw source slice; a literal keeps its cooked bytes, and scanner string cooking is decoded to real bytes through `literals::decode` instead of a hex transport. It retains the assignment operator bytes and copies locals, calls, arguments, Region scopes and exits. A cleanup body's `cleanup_start` is the `cleanup` keyword, so it differs from the root block span.
`check` re-runs the cleanup-MIR verifier, matches the body count to the cleanup frame count, bounds the node/statement/text/work/name/table totals, requires one kind-1 root statement per body, and confirms each node's source order, kind, span and left/right against its syntax node, each block list length against its reverse links, and each call, Region and operand shape. Layouts keep their declaration identity, and callable parameters stay inside the parameter pool.
The fragment keeps semantic request order and does not attach drop sites to CFG blocks or connect the reverse-lexical, defer and Region ordering, so CFG structure, drop order, the C handoff and any executable split-unit body remain pending.
The existing `module_program` C path still joins and is unchanged, and no CLI route exercises the fragment. The new `tests/module_unit_body_ir.fy` holds 4/4 native tests: three bodies plus a fourth that accepts a valid clone and rejects eleven malformed table modes with ten 1001 and one 1002. Evidence: `docs/ops/evidence/candidate-module-body-ir-20261010.json`. Release, Linux and the full compiler suite stay deferred, and the stable CLI, SDK ABI, DB and CI runner are unchanged. No push.

## Candidate source-independent CFG (2026-10-10)

A private source-independent control-flow graph now lowers from the owned typed body. `module_unit_cfg_model.fy` defines one-operation `Block` records in a `Graph` resource (1 eval, 2 store, 3/4 branch, 5 drop, 6 return, 7 jump, 8 assert, 9 propagate error, 10 short-circuit copy, 11 branch/block copy, 12 unreachable completion) and `module_unit_cfg_lower.fy` continuously builds them in reverse from one terminal block through successor edges.
`check` bounds the block count and work, requires a valid frame and entry, and checks each block's shape and bounds: the successor, source kind, constructor type and node/statement link, bool branch and assert sources, the store target matching its expression node, the Result `?` propagation against the frame result, and short-circuit and block-copy operands. Equivalent repeated cleanup requests share one runtime drop operation and every in-frame cleanup request is covered; this is a shape and coverage check, not a dominator or dominance-path proof.
The drop chain orders a temporary's materialized operands in construction order so execution reverses them before lexical owners on an abrupt exit, keeps a replacement's right-hand side before the old owner is dropped with no target-root load, and normalizes a revisited loop condition's identical cleanup requests to one runtime drop site. if, the `&&`/`||` short-circuit skip branch, `while`, and the classic `for` backedge and its post clause are formed, and break/continue and scope exits attach their reason-matched drops before jumping. Normal records are supported; only the named-construct syntax (node kinds 32/33) is rejected with 1004, along with switch, foreach, slices, defer and Region; the source admission origin and unsafe preadmission checks remain separate.
The new `tests/module_unit_cfg.fy` holds the graph path tests (graph topology and block order, not generated C execution or a self-hosting graph interpreter) and passes 7/7 native tests; its ninth mutation mode is accepted as a copy while modes 1-7 and 9 reject with 1001 and mode 8 with 1002. Evidence: `docs/ops/evidence/candidate-module-cfg-20261010.json`.
Generated C, the whole language, defer/Region, the remaining source admission origins and unsafe preadmission, and the drop dominance-path proof remain pending. The existing `module_program` C path still joins and is unchanged, the stable CLI, SDK ABI, DB and CI runner are unchanged, Release, Linux and the full compiler suite stay deferred, and no push.

## Candidate bounded lexical defer stage (2026-10-10)

`defer call;` now lowers into the source-independent CFG. Explicit arguments evaluate
immediately left to right and freeze into capture slots over the closed scalar set
(kinds 1..6, now including `i64`), and the target must be a resolved internal call
returning `void` whose callable or instance frame passes the authority, result,
argument-count and effect-subset checks; foreign, template and value-returning targets
are rejected. Registered calls invoke in reverse registration order (LIFO) on
fallthrough, `return`, Result `?` propagation, `break` and `continue`, with the return
value frozen before the deferred calls and both before implicit drops, and
`break`/`continue` invoke only the defer suffix registered inside the loop while the
outer registrations wait for the enclosing exit. Defer site IDs are assigned in
per-block preparation order and are not a registration-order contract, so the nested
case asserts the captured literal values in execution order `3,2,4,1` rather than site
IDs. The stage is validated by five `tests/module_unit_cfg_defer.fy` tests (four
behavior paths plus one mutation case that accepts a valid clone and rejects nine
malformed defer modes with 1001) and the unchanged seven `tests/module_unit_cfg.fy`
graph tests, twelve unique native passes in all; the source check passed 252 files and
3205 MIR functions and the locked development build passed, with the receipt
`docs/ops/evidence/candidate-module-cfg-defer-20261010.json` recording the twelve passes
and the nine new defer rejections. `Region` is pending for the next commit, and
`switch`, `foreach`, slices, the named-construct syntax, the C-graph evaluator and
executable C remain unimplemented; this section supersedes the earlier defer-pending
statements above.

## Candidate Region placement and reset staging (2026-10-10)

The owned body IR now also retains Region structure: `regions::Scope` (parent, capability,
scope, statement, body, source mode, arena owner/type and spans) and
`regions::Placement` (region, capability, the `place` call and argument, owner/borrow/
result/error types and spans) are copied at lowering and revalidated independently,
pairing every builtin `place` call one-to-one, requiring the kind-19 capability local and
the canonical `ref mut std::alloc::Arena` source, and checking that a nested Region stays
inside its parent body. `module_unit_cfg_region.fy` lowers a Region checkpoint (block kind
15) and Region resets (kind 16): on an exiting edge it emits all active defers first, then
for each crossed Region its drops followed by its reset innermost-first, then the remaining
outer drops, with reset group/order chaining to the parent and the reset and drop spans
checked against the source; the verification stays at these ledger and span facts and does
not claim dominance, liveness or runtime ordering.

Eight `tests/module_unit_cfg_region.fy` outcomes now pass - five new Region cases
(fallthrough, return with `?` interleaving inner-first resets, break/continue resetting
only the crossed Region, eight invalid graph modes rejected with 1001, and an eleven-mode
invalid body-ledger mutation rejected with 1001) plus three existing regression tests, split
across `region-native-final.json` and `region-mutations-regression.json` - giving nineteen
new rejections in all (eight graph and eleven body modes). The case reads the real
`internal/sdkstdlib/sources/std/alloc.fy` by byte identity, recorded in the receipt
`docs/ops/evidence/candidate-module-cfg-region-20261010.json`, with no SDK ABI change and
no production source change after the first four-pass run. The locked source check passed
253 files and 3215 MIR functions and the locked development build passed, changing the
private IR API while the stable CLI, SDK ABI, DB and CI stay unchanged with no push.
Generated C, the whole language, the drop dominance-path proof, the whole source admission
and pipeline and any Arena runtime checkpoint execution remain pending, the tests
exercising the `.fy` compiler graph lowering and paths only, with no C emitted from or executed for this new CFG.

## Candidate owned-body scalar C emitter (2026-10-10)

A private scalar C render API now emits a standalone numerical C translation unit from the owned body IR without the original `Parsed`, its joining modules or a source join. `module_unit_c_emit.fy` exposes `render(ref ir::Program p, usize entry, ref mut bytes::Buffer out)`: it revalidates the body IR through `module_unit_body_ir_verify`, requires the caller-selected `entry` frame to be a zero-parameter `i32` function, freshly lowers every body's `Graph` with `module_unit_cfg_lower`, and emits numerical function, local, block and value identifiers instead of source names. `module_unit_c_scalar.fy` renders the closed scalar set of kinds 1..6 (i32, i64, u8, u32, usize, bool) as fixed-width `int`/`uint` locals and treats kind 7 as void, with the checked helpers `fy_unit_signed`, `fy_unit_unsigned` and `fy_unit_bits`, so unsigned widening and high-bit shifts stay out of signed C arithmetic. Output is transactional: render validates and builds one bounded `bytes::Buffer` (2 MiB) and appends to the caller's buffer only on success, so a rejected program leaves the destination untouched. The header carries `fairyc.candidate-module-scalar-c/v1`, 64-bit and two's-complement `_Static_assert`s, and `_Noreturn void fy_runtime_trap(uint32_t category)`.

The unit emits boolean branches, `while` and classic `for` loops with `break`/`continue`, compound assignments, block/if value copies, frozen scalar `defer` capture slots and internal ordinary and generic calls; resources, `Result`/`Option`, views, references, arrays, casts, extern calls and Region execution are rejected with 1004. `tests/module_unit_c_emit.fy` holds six native cases: five emitted C programs (calls, loops and lazy branch values; defer-argument freezing; closed integer widths and normalized literals; an overflow trap; a divide-by-zero trap) plus a rejection that proves a `sentinel` buffer survives both a 1004 unsupported-type program and a 1001 bad entry with no partial output. The five programs pass under both MSVC `/Od` and `/O2`, ten executions in all, with the expected exits, including generic identity `h::id<i32>(answer)==11` and recursive `factorial(5)==120` and the latest literal signed-bit conversion; the expected exits are 0 for success and, through the dedicated standalone trap shim, 101 for trap category 1 (overflow) and 102 for category 2 (divide by zero). They are compiled by `tools/verify-candidate-module-c.py` from a `fairyc test --native --capture-output` report whose compiler-source provenance is caller-supplied and not authenticated by the tool. The render test runs as `/development`, so the generated C is not the candidate release build.

The complete dominance and path proofs, whole source admission, and the full dispatcher and stage1/stage2 remain pending, so this is neither a whole-language nor a full-path claim. The generic and recursive call assertions passed on the final source. Public CLI, stable SDK ABI, Go, DB and CI are unchanged and there is no push. The scoped evidence receipt `docs/ops/evidence/candidate-module-c-execution-20261010.json` records the six native cases, the ten generated-C executions and the source hashes. Release, Linux and the full compiler suite stay deferred. This supersedes the earlier scalar-C handoff pending notes above.

## Candidate owned aggregate C unit (2026-10-10)

`module_unit_c_aggregate.fy` and the updated `module_unit_c_emit.fy` emit used canonical structs, resources, Option and Result in topological definition order under `fairyc.candidate-module-owned-c/v1`. Field access checks retained field ID, type, module, declaration and source provenance; tagged values use a tag and active-payload union. A cleanup receiver is a pointer to live owner storage, so its body can mutate the fields subsequently dropped. Per-type drop helpers run the cleanup body first, then nested fields in reverse declaration order, and visit only the active Option/Result payload. A graph drop operand resolves to its frame-local binding or materialized original expression. Aggregate construction, internal calls, moves and return copies follow the checked CFG ownership ledger; Result `?` projects success or returns the error after the existing exit defers and drops.

Ordinary scalar extern C calls now resolve retained callable provenance, signature and effect subsets and `ffi_symbols::valid` linker names. Exact compatible linker prototypes are deduplicated across providers; pointer, aggregate and C-scalar extern ABI remain unsupported. Registered defers still accept only internal targets with scalar arguments. Admission revalidates the owned body IR and CFG; checked arithmetic uses the bounded scalar helpers. The existing non-unwinding trap contract remains: traps run no defers, drops or Region resets and have no cleanup edge.

The combined native run of `tests/module_unit_c_emit.fy` and `tests/module_unit_c_owned.fy` with `--capture-output` passes 12/12 (six new owned cases and six scalar regressions). `tools/verify-candidate-module-c.py --suite all` compiles ten programs under both MSVC `/Od` and `/O2`, with 20/20 executions passing (ten owned and ten scalar) and exact expected exits and stdout. The owned C-shim traces are: replacement receiver mutation `102,103`; tagged nested cleanup `5,4,3,100,2,1`; consumed owner and partial propagation `20,1,2,3,9,4`; loop LIFO defers `20,10,0,21,11,1,100`; provider cleanup `30,10`. Native generation tests use the development profile; generated-C `/O2` execution does not validate a candidate Release build. The scoped receipt `docs/ops/evidence/candidate-module-owned-c-execution-20261010.json` records `passed=true`, all 14 source hashes verified against current files, and SHA-256 references to the full cached reports. The locked source check passed 256 files and 3252 MIR functions, and the locked development build passed.

Arrays, references, pointers, views, casts, enums and Region execution remain unsupported. Full path, dominance and initialization proofs, complete source admission and unsafe provenance, the whole dispatcher and stage1/stage2 remain pending. The added Fairy module reaches the existing 256-source cap without changing it. The private C API expands; public CLI, stable SDK ABI, Go, DB and CI are unchanged. No push; candidate Release, Linux and the full suite remain deferred. This section supersedes earlier aggregate, resource, Option/Result and scalar-extern C pending notes only within this bounded unit.

## Candidate reference C unit (2026-10-10)

The updated `src/module_unit_c_aggregate.fy` and `src/module_unit_c_emit.fy` emit canonical `ref T` and `ref mut T` as C pointers, with a const pointee for shared `ref`. Dereference, field projection and address formation preserve the owner's storage; borrowed resources are not dropped, and generic reference returns are supported. `src/module_unit_cfg_lower.fy` and `src/module_unit_cfg_model.fy` add CFG kind 17 to snapshot a scalar compound-assignment target's old value before evaluating the RHS; the generated C store consumes that snapshot. Five mutation modes reject malformed, missing or duplicate snapshots.

The combined three-suite native run passes 18/18 (six new reference cases and twelve regressions). Fifteen generated C programs pass under both MSVC `/Od` and `/O2` (30/30 executions: ten reference and twenty regression), with exact reference traces: root/field `3,6`, borrowed resource `10,3`, indirect `5` and sequential `8`. The latest `tools/verify-candidate-module-c.py --suite all` includes references; earlier sections retain their historical receipt counts. The locked check passed 256 sources and 3253 MIR functions, and the locked development build passed. The receipt `docs/ops/evidence/candidate-module-reference-c-execution-20261010.json` records the results and fifteen source identity hashes verified against current files (fourteen candidate source/test/lock files and one verifier). Generated-C `/O2` execution is separate from candidate Release validation.

Raw pointer, array, view, cast and enum transport, foreign pointer ABI and SDK calls, Region execution, complete source admission and unsafe provenance, dominance/path/initialization proofs, the whole pipeline and stage1/stage2 remain pending. The deliberate non-unwinding trap contract remains unchanged. Public CLI, stable SDK ABI, Go, DB and CI are unchanged; no push. Candidate Release, Linux and the full suite remain deferred. This bounded unit supersedes earlier references-pending notes.

## Candidate bounded fixed-array C unit (2026-10-10)

`src/module_unit_c_aggregate.fy` and `src/module_unit_c_emit.fy` now emit canonical kind-16, rank-one fixed arrays of scalar kinds 1..6 or inert structs, with lengths 1..4096. Dependency-ordered numerical wrapper structs contain `fy_items[N]`, allowing by-value arguments, returns and copies. List and repeated literals copy frozen child values; the repeat operand evaluates once, and `arrays::count` checks the retained repeat-count text against the canonical extent. Array `.len` is supported. Index bounds trap with category 6 before a read and before an assignment RHS. Projected-place writes and element-reference formation target actual owner storage, retaining kind-17 compound-assignment snapshots. Shared/mutable array views and whole-array borrow parameters remain unsupported. `array_c_copies_values_across_calls_and_returns` exercises ordinary array copies with a separate scalar identity call; generic admission remains scalar-only under ADR 0384. Fixtures use a plain Container array field and a separate scalar-field resource, preserving the resource-field restrictions of ADRs 0463/0468.

The final locked source check passed 256 files and 3253 MIR functions, and the final locked development build passed; no implementation source file or source-cap change was needed. One full seven-test native report on the final production code and fixtures records 7/7 passed, zero failed: five array programs and the updated `module_c_rejects_unsupported_types_and_bad_entry_without_partial_output` and `owned_c_rejects_pending_view_transport_without_partial_output` text-transport rejection tests. `tools/verify-candidate-module-c.py --suite arrays` compiled five programs at MSVC `/Od` and `/O2`, passing 10/10 executions with exact stdout matched. The receipt `docs/ops/evidence/candidate-module-array-c-execution-20261010.json` records twenty source identity hashes verified against current files (nineteen candidate source/test/lock files and one verifier).

The original loop-filled 4096-element `i64`-maximum repeat passed at `/Od` but copied -1 instead of the maximum into owner storage at `/O2`, failing assertion category 5 (exit 105). Repeat emission now uses an aggregate initializer with the once-frozen operand repeated, following `c_arrays`; the corrected program passes `/O2` with stdout `4,1,3,4,9,1`. Optimization remains enabled.

The expanded `--suite all` (23 native tests, 40 C executions) was not run for this unit. Historical receipt counts remain unchanged; the unchanged baseline suites were not rerun on current sources. Generated-C `/O2` execution remains separate from candidate Release validation.

Direct nested-array elements, raw pointer/view/cast/enum transport, authenticated SDK calls and foreign pointer ABI, Region execution, complete source admission and unsafe provenance, full path/dominance/initialization proofs, the whole pipeline and stage1/stage2 remain pending. Traps retain the deliberate non-unwinding contract: no defers, drops or Region resets. The private C API expands; public CLI, stable SDK ABI, Go, DB, CI and hygiene rules are unchanged. No push; candidate Release, Linux and the full suite remain deferred. This bounded status supersedes earlier fixed-array C pending notes.

## Candidate array-backed view C unit (2026-10-10)

`src/module_unit_c_aggregate.fy` now emits canonical byte views (kinds 10/11), extended scalar views (15) and plain-struct views (17/18) as numerical descriptors containing `T const *fy_data` or `T *fy_data` and `uintptr_t fy_length`. Array borrows and subranges alias original owner or field storage; left-value slots may still materialize array copies. Descriptors support `.len`, `.is_empty`, reads and mutable writes through `fy_data`, category-6 bounds checks, empty end ranges, nonempty nested subranges and omitted bounds. This unit covers array-backed borrowed views only.

`src/module_unit_cfg_lower.fy` evaluates the slice owner, lower bound and upper bound in order without emitting the kind-27 wrapper. `src/module_unit_c_emit.fy` freshly revalidates checked expected/actual argument types and shared-conversion flags; mutable-to-shared view arguments use a descriptor compound literal, and reference pointers convert to const pointees.

The first view-mutation C program passed MSVC `/Od` but failed assertion category 5 (exit 105) at `/O2`: owner elements were 4/5 and the materialized copy held 5, while a scalar read retained 3. Production kind-2 aggregate owner reads, excluding scalar/reference types, now use `fy_unit_copy`. Under `_MSC_VER`, `#pragma function(memcpy)` and `__declspec(noinline)` force CRT `memcpy` only in that helper; `#pragma intrinsic(memcpy)` then restores scalar-bit helper intrinsics. Final generated C passes with `/O2` retained. Descriptors still alias original storage; materialized array copies remain.

Final validation on the final core and fixtures passes 11/11 native tests: seven view programs, the existing 4096-repeat array and owned nested-tag-drop programs, and two existing text-rejection guards. Nine C programs pass under both MSVC `/Od` and `/O2` (18/18 executions: fourteen view and four aggregate), matching every stdout and exit exactly. View stdout includes owner alias `7,4,5,`, struct-field data `6,7,` and bound order `11,22,2,33,`; reversed and upper-bound slices exit 106 with stdout `1,` and `2,`, respectively, and an empty-end index exits 106 with empty stdout. The locked check passed 256 sources and 3264 MIR functions, and the locked development build passed. The receipt `docs/ops/evidence/candidate-module-view-c-execution-20261010.json` records the final results, failed C attempt and targeted correction, with all twenty-one current source hashes verified (twenty candidate source/test/lock files and one verifier). Expanded `--suite all` (30 native tests, 54 C executions) was not run; full scalar/reference/owned/array suites were not rerun, while the two selected existing aggregate programs passed. Prior receipt counts remain historical. Linux was not run; generated-C optimization remains separate from candidate Release validation.

Byte, text and CStr literals remain unsupported: literal node kind 1 still uses the scalar-only emitter. Raw pointer/cast/enum/direct nested-array transport, authenticated SDK calls and foreign pointer ABI, Region execution, complete source admission and unsafe provenance, dominance/path/initialization proofs, the whole pipeline and stage1/stage2 remain pending. Traps retain the deliberate non-unwinding contract. The private C API expands; normative contracts, public CLI, stable SDK ABI, Go, DB, CI and hygiene rules are unchanged. No push; candidate Release, Linux and the full suite remain deferred. This section supersedes earlier array-view pending notes only within the stated scope.

## Candidate owned scalar-cast C unit (2026-10-10)

The private owned-body scalar C emitter now admits explicit scalar casts. `module_unit_c_scalar.fy` checks a cast node (kind 23) as an exact target/result/source pair -- the retained `as` text, `target == result`, and the operand scalar against the admission table in `scalar_casts.fy` -- rejecting a missing or inconsistent target, a missing operand or a wrong `as` token as a malformed cast (1001) and an admitted pair mismatch, such as a forged identity, as an unsupported pair (1004).

`scalar_casts.fy` admits the seven value-preserving integer widening pairs (u8 to i32, i64, u32 and usize; i32 to i64; u32 to i64 and usize) plus the exact two-way i32/c::int conversion from `c_abi_types::cast`. c::int stays nominally distinct: it is spelled `int` while i32 stays `int32_t`, so locals, parameters, returns and ordinary scalar extern prototypes keep the two apart, and `module_unit_c_aggregate.fy` emits `_Static_assert(sizeof(int)==4 && _Alignof(int)==4 && INT_MIN==INT32_MIN && INT_MAX==INT32_MAX,"requires signed-32 c.int ABI");` only when a used c::int (scalar 132) is actually admitted; every other C scalar and all pointer casts stay closed (1004).

A cast freezes its operand by evaluating it once before the widening, and arithmetic after widening remains checked at the target type range by reusing the existing helpers. `tests/module_unit_c_casts.fy` adds focused cases for the admitted widening ranges, signed-32 round trips through separate C functions, single evaluation of projected and call operands, widened checked overflow, and transactional rejection: a full render of a c::uint program returns 1004 with the sentinel buffer intact, and four cast-guard modes -- three malformed (missing or inconsistent target, missing operand, wrong `as` token) and one unsupported forged identity pair -- are turned away by the direct scalar emitter before any write, leaving the sentinel intact. The earlier sections here are historical and this follow-up status supersedes them only within this cast scope; Linux, the candidate Release build, the full suite, whole source and path proofs, Region, pointers, enums, SDK authentication and stage1/stage2 remain pending, and public CLI, stable SDK ABI, Go, DB, CI and normative contracts are unchanged with no push.

Final validation passes 7/7 native C-emitter cases (five new cast cases and two regressions) plus all 4/4 existing cast-typing cases, 11/11 native tests in total. Six generated C programs pass at MSVC /Od and /O2 (12/12 executions), matching every exit and stdout. Signed-32 ABI round-trip stdout is -2147483648,2147483647,42, and single-evaluation/defer stdout is 1,7,9,; widened overflow exits 101. The locked check passed 256 sources and 3264 MIR functions, and the locked development build passed. The receipt docs/ops/evidence/candidate-module-cast-c-execution-20261010.json records the results with 21 current source hashes verified. The full suite, Linux and candidate Release were not run; generated-C /O2 is separate from candidate Release validation.

## Candidate owned pointer-cast typing unit (2026-10-10)

This unit adds owned-body source typing and unsafe provenance for explicit pointer casts and adds no C transport. `module_unit_name_model.fy` gives a cast (expression kind 23) its own `Reference` kind 4 and `module_unit_name_walk.fy` runs that operation before the operand, so the cast's own lexical unsafe boundary is recorded separately; an unsafe nested in the operand or a closed sibling unsafe cannot confer cast permission. `module_unit_value_collect.fy` matches the reference by exact expression, source index and span and preserves it as `builtin.unsafe_context`, which `module_unit_value_scalar.fy` requires for an admitted pointer pair and otherwise rejects with 5101. `module_unit_value_place.fy` admits an exact scalar or nominal reference pointee, narrowing from mutable to shared while refusing shared to mutable, shared and mutable byte views, `str` and `CStr` to byte pointer, and the exact integer cast rules are retained; void erasure is limited to a direct named reference or a direct borrow of a named local, array-backed view to void is its descriptor data as a typing state only, and raw pointer to pointer and integer to pointer reinterpretation are refused. Admitted pointer casts retain `type == target` with state 1 fully checked under ADR 0123. Pointer C transport, the `void**` output slot and raw dereference and arithmetic remain unsupported and pending, with raw dereference and arithmetic excluded by the Stage contract and unopened; owned C transport is the next unit, so this records source typing and unsafe provenance only. Public CLI, stable SDK ABI, Go, DB and CI are unchanged, the historical records above are untouched and this adds a new scope, and there is no push with Linux, candidate Release and the full suite deferred.

Focused validation passes 9/9 native tests: four new pointer tests, one updated field/pointer test, three scalar regressions and the existing complete-coordinate owner-read regression. The locked check passed 256 sources and 3268 MIR functions and the locked development build passed. All 249 root-production-source and cast-fixture input hashes stayed unchanged during the run. Evidence: docs/ops/evidence/candidate-module-pointer-typing-20261010.json. This unit emitted no pointer C; candidate Release, Linux and the full suite remain deferred.

## Candidate owned pointer C unit (2026-10-10)

The owned-body C emitter now transports canonical kind-12 raw pointers using their C spellings: encoded scalar, `void` and `void**` pointers plus nominal pointees spelled `T *` or `T const *`. It renders typed `null`, inert pointer copies, and internal pointer arguments, returns and `==`/`!=` comparisons. Reference casts forward the referent pointer, including the original fixed-array wrapper address for an opaque `void` cast, while borrowed view casts extract `fy_data`; an opaque local resource or array/view address crosses as `void` without a carrier address. Foreign ABI stays limited to encoded scalar, `void` and `void**` pointers, so a nominal pointer foreign ABI is still rejected 1004 pending `repr(C)` layout assertions, while nominal pointers still pass and compare through internal calls.

A direct cast of a fixed-array borrow (`ref mut u8[2] as ptr mut u8`) is refused by the contract, matching the Go baseline, so the fixture binds it explicitly (`let ref mut u8[] data = ref mut bytes;`) and then casts `data as ptr mut u8`; the array opaque `void` cast verifies the original owner address unchanged.

The central `raw_cast` rule in `module_unit_value_place.fy` now additionally admits only the single mutable opaque pointer output slot per ADR 0059 (`ref mut ptr mut void` to `ptr mut ptr mut void`); pointer-integer and pointee reinterpretation are refused and raw dereference and arithmetic stay excluded by the Stage contract. The source-independent body IR verifier rechecks each pointer cast in `check_casts` against the canonical source, target, state and `as` token with its unsafe flag, requiring containment in a retained unsafe expression (kind 31) or a matching non-foreign unsafe callable found by module and declaration signature, and an opaque conversion additionally requires retained direct local syntax. This is a retained-IR gate over the owned body, not authentication of the original source or a full path proof.

Private C emission stays transactional, and the foreign nominal pointer rejection leaves the sentinel buffer intact. The focused C programs cover `null` with a foreign returned pointer and a `void**` output slot, mutable and shared scalar references with a generic provider unsafe call, opaque resource and array/view data crossings whose cleanup runs once, and nominal pointer identity, with the existing C `c::int` and view regressions selected. byte, text and CStr literals remain unsupported for C and text/CStr parameters stay unsupported for C despite typing; foreign nominal ABI, enums, Region, authenticated SDK calls, full source-path proofs, the whole pipeline and stage1/stage2 remain pending; Linux, candidate Release and the full suite stay deferred; normative contracts and public CLI, stable SDK ABI, Go, DB, CI and hygiene rules are unchanged with no push.

Focused native validation passes all 8 selected identities: four pointer C programs, one pointer guard, one output-slot typing case and two C regressions. The first run passed 7/8; after correcting only the unsupported direct-array-to-byte-pointer fixture, its 1/1 rerun passed. Exact byte reconstruction verifies all other helper/test bytes and production/verifier inputs unchanged, so the seven earlier passes were reused. Six generated C programs pass MSVC /Od and /O2 (12/12 executions: eight pointer and four regression), matching every exit and stdout. Pointer traces are 42,77, for null/result/output-slot; 9,6, for scalar storage; and 8,5,9,9, for byte-view/array/resource storage and one cleanup. Locked check: 256 sources and 3271 MIR functions; locked development build passed. The receipt docs/ops/evidence/candidate-module-pointer-c-execution-20261010.json records the initial failure, correction, final results and 22 verified current source identities. Linux, candidate Release and the full suite were not run; generated-C /O2 is separate from candidate Release validation.

## Candidate owned literal C unit (2026-10-10)

Receipt: `docs/ops/evidence/candidate-module-literal-c-execution-20261010.json`. This unit supersedes earlier pending notes for owned literal storage and internal text/CStr transport.

- Implementation: Numerical immutable translation-unit arrays; full SHA-256 symbol order; exact-byte deduplication/collision check; one-byte non-null empty sentinel with zero logical length; CStr decoded storage includes exactly one trailing NUL.
- C transport: Canonical distinct kind9 text/kind10 byte/kind14 CStr descriptors; internal arguments/returns/copies; immutable data-pointer casts; text and byte length/is_empty; byte checked indexing; no direct foreign descriptor ABI.
- Owned literal proof: `9acfc353`; 3/3 native identities, including five payload/type mutations.
- Current native outcomes: 9/9 passed; first run 6/9, then 3/3 corrected signature guards. Exact prior-byte reconstruction proves the six reused fixture/helper and production inputs unchanged.
- Generated C: 12/12 MSVC executions passed (`/Od` and `/O2`); four literal programs plus pointer and view regressions. Byte index at literal extent exits 106.
- Candidate: check ok: 2 instance(s), 257 source file(s), 3284 MIR function(s), lock verified; built: build/lexer-candidate.exe (x86_64-windows-msvc, development, lock verified).
- Go reference boundary: two standalone checks pass; direct-call result address casts exceed its verified-MIR subset (`raw-address cast source is not a direct borrow`). Provider graph parity remains unverified.
- Changes: private API and local C verifier expanded; normative contracts, stable CLI, SDK ABI, Go, DB, CI and hygiene unchanged; no push.
- Deferred: Linux, candidate Release and the full suite. Generated C `/O2` does not establish candidate Release.
- Remaining: Enums/Region/authenticated SDK and full dispatcher C execution; Raw views from authenticated SDK constructors; Foreign nominal pointer/aggregate ABI layout assertions; Original source/unsafe authentication, path/dominance/initialization proof and stage1/stage2; Resolve/test differential against baseline direct-call address subset.

## Candidate owned enum metadata proof (2026-10-10)

Receipt: `docs/ops/evidence/candidate-owned-enum-proof-20261010.json`. This is the prerequisite metadata proof for enum C transport; enum/switch execution remains pending.

- Layout: Unique canonical layout; module/declaration/source-field coordinates; retained variant source order; contiguous payload-before-marker ranges; unique variant/payload names; no enum cleanup/abilities/reprC metadata.
- Payload: Acyclic inert scalars/raw pointers/structs/user enums and recursively inert Option/Result; safe borrows/resources/Vec/Region/void excluded; Option/Result cleanup classification is not confused with ownership.
- Construction: Retained bare/aggregate constructor matches canonical enum/variant/field provenance and names; every payload field supplied exactly once at its canonical value type.
- Validation: final 3/3 native identities passed (provider/empty/nested inert and copy-tagged shapes, ten owned-IR mutations, existing UTF-8/CStr proof regression); locked check passes 258 sources and 3294 MIR functions.
- Corrected attempts: private fixture mutation was refused at compilation; payload-before-marker arena order then required a verifier correction. All three final tests reran on unchanged final inputs.
- Private API: read-only retained-name byte view and canonical enum metadata helpers. No changes to normative contracts, stable CLI, SDK ABI, Go, DB, CI or hygiene; no push.
- Deferred: candidate build, Release, Linux, full suite and generated enum C execution.
- Remaining: Enum private C tag/union construction and transport; Enum/Option/Result switch CFG and payload binding execution; Owned exhaustive-pattern tables and active-payload proof; Complete enum-helper analysis-budget/allocation diagnostic propagation; Source/unsafe authentication and full path/dominance/initialization proofs; Authenticated SDK, whole dispatcher and stage1/stage2.

## Candidate owned enum diagnostic propagation (2026-10-10)

Receipt: `docs/ops/evidence/candidate-owned-enum-budget-20261010.json`. Analysis-budget failure now retains 1002 even when a multi-unit charge fails below the maximum; structural failures retain 1001 and helper allocation failures retain 1003. The private layout helper carries a diagnostic slot. This completes the diagnostic propagation follow-up in the preceding enum unit.

Final 2/2 focused native identities pass: every incomplete measured-work prefix reports 1002, ten structural mutations report 1001, and the provider/empty/nested inert and copy-tagged cases pass. Locked source check passes 258 sources and 3295 MIR functions. Allocation injection and a synthetic depth fixture were not run; those mappings were inspected. Candidate build, Release, Linux and the full suite remain deferred. Stable CLI, SDK ABI, normative contracts, Go, DB, CI and hygiene are unchanged; no push. Enum C transport, switch execution, SDK, the full dispatcher and stage1/stage2 remain pending.

## Candidate owned enum C value transport (2026-10-10)

Receipt: `docs/ops/evidence/candidate-module-enum-c-transport-20261010.json`. This completes owned enum C representation, construction and by-value transport; source switch execution remains pending.

- Layout: Canonical kind4 enum carriers; private signed-32 declaration-order tag, named union variants and declaration-order typed fields; empty variants and uninhabited enum declarations retained.
- Dependency order: Complete by-value scalar/struct/nested enum/copy-tagged payloads before their containing enum; raw nominal pointees only forward declared during traversal and completed afterwards, allowing an acyclic E -> ptr S -> by-value E relation.
- Values: Fresh carrier initialization then field writes from frozen typed expression values; payloadless and payload-bearing constructors retain canonical field IDs; Inert by-value copies, internal parameters/returns, shared borrowed carrier addresses and Option/Result wrappers over enums.
- Validation: final 5/5 native identities pass; four generated-C programs pass MSVC `/Od` and `/O2` (8/8 executions). Two enum programs, a nested-resource/tag drop regression and the exact-byte literal regression all match exit/stdout. Ordered enum initialization and the final copied-value trace are `1,2,2,`.
- Oracle: private same-translation-unit test companions inspect actual tags and scalar/struct/raw-pointer/nested enum/copy-tagged payloads. They are test readers, not source-level switch lowering or a public enum ABI.
- Reference: both new source programs pass the pinned Go compiler experimental verified-MIR check (the provider source is supplied explicitly). Locked candidate check passes 259 sources and 3298 MIR functions; locked development build passes.
- Guards: foreign enum arguments reject 5102 during signature admission; source switch C rendering rejects 1004 transactionally.
- Changes: private C API and local C verifier expanded; normative contracts, stable CLI, SDK ABI, Go, DB, CI and hygiene unchanged; no push.
- Deferred: Linux, candidate Release, full suite, source switch execution and public enum ABI. Generated C `/O2` is not candidate Release evidence.
- Remaining: Enum/Option/Result switch CFG and active payload binding execution; Owned exhaustive-pattern table and active-payload proof; Authenticated SDK/Region and full dispatcher execution; Foreign aggregate ABI layout proof and complete source/unsafe/path proof; Stage1/stage2 and promotion.

## Candidate owned switch pattern proof (2026-10-10)

Receipt: `docs/ops/evidence/candidate-owned-switch-proof-20261010.json`. Owned Enum/Option/Result switch tables and lexical payload bindings now receive a source-independent structural proof before CFG lowering. CFG and generated-C switch execution remain pending.

- Tables: concrete scrutinee/result types; every arm and reverse-list record has one owning switch; every canonical variant appears exactly once, with no missing or duplicate arm.
- Payloads: every named enum payload field appears once at its canonical owner/field/type coordinates; reordered fields and arms retain their identities; builtin named and wildcard payloads retain their expected types.
- Symbols: one owning arm per pattern, field wrapper and lexical binding; original symbol, type and scope agree; retained identifier uses remain inside that arm body, including nested switches and shadowed names.
- Results: continuing tail types match the switch result; the reference's narrow direct-final-return Result and assert(false) Option/Result no-tail exceptions remain intact. This is not full path/dominance proof.
- Validation: final 4/4 focused native identities pass, including provider/nested/empty enum regressions, nineteen switch/body/local mutations and the existing ten layout/construction mutations. Every incomplete measured-work prefix of four representative switch frames reports 1002. Locked source check passes 260 sources and 3307 MIR functions.
- Changes: private verifier API and source lock updated. Normative contracts, stable CLI, SDK ABI, Go, DB, CI and hygiene unchanged; no push.
- Deferred: allocation injection, standalone candidate rebuild, candidate Release, Linux, full suite and generated switch C execution.
- Remaining: switch CFG and active-variant payload binding C execution; authenticated SDK/Region and whole dispatcher; source/unsafe authentication, full path/dominance/initialization proof; stage1/stage2 and promotion.

## Candidate switch CFG and payload gates (2026-10-10)

Receipt: `docs/ops/evidence/candidate-switch-cfg-20261010.json`. Enum/Option/Result switches now lower to frozen-subject case tests and guarded active payload seeds. C rendering still rejects the new block kinds until their emitter is implemented.

- Each case retains its source switch and arm IDs, has one successful payload-seed edge, and follows the retained source-order failure chain. Unknown tags end at a trap. Missing, duplicate, relabelled or bypassed case/seed edges and graph entry into a payload seed or middle case reject 1001.
- Arm result copies, terminal no-tail arms, returns, lexical arm defer suffixes and named owned payload match-arm drops now lower through the existing cleanup graph. This is not complete CFG dominance/initialization proof.
- Final 5/5 focused native identities pass: provider and nested Option/Result arms, one scrutinee call, named owned cleanup and return transfer, arm defers, twelve CFG mutations, and loop/short-circuit regressions. Locked check passes 261 sources and 3310 MIR functions.
- Inspection found the candidate resolver omits wildcard payload symbols while reference MIR retains anonymous bindings. Owned wildcard cleanup needs verification and repair before C switch admission; no runtime leak is claimed by this receipt.
- Private CFG API and lock updated. Normative contracts, stable CLI, SDK ABI, Go, DB, CI and hygiene unchanged; no push. Generated switch C, standalone build, Release, Linux, allocator injection and the full suite were not run in this CFG unit.
- Remaining: anonymous owned wildcard cleanup; C variant tests and active payload extraction; authenticated SDK/Region and whole dispatcher; full source/unsafe/path/dominance/initialization proof; stage1/stage2 and promotion.
