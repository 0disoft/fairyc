# Fairy 컴파일러 전환: 후속 작업 목록

2026-10-07 현재 소스와 진행표를 기준으로 정리했다. 14번의 구조체 필드와
배열 실행, 15번의 resource 타입·필드 소유권과 16번의 factory 생성, 17번의 이동 HIR과
18번의 이동 후 사용 검사, 19번의 분기 합류와 20번의 반복·재진입 검사까지 완료했다.
승인된 1~20번은 완료했고, 번호 21~127은 107개 후속 작업이며,
항목마다 별도 커밋이나 전체 테스트를 요구하는 목록은 아니다.
의존성이 맞는 작은 구현 단위로 묶어 순차 커밋한다.
Linux 검증과 빌드 비용 측정은 요청대로 뒤에 묶어서 수행한다.

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
24. 완료: 반복 본문 fallthrough와 nearest-loop break/continue에서 실제로 빠져나가는 중첩 본문만 정리. 바깥 root 보존, for post 이전 정리, 반복 재생성과 return을 dev/release native 실행. 근거는 `evidence/candidate-resource-loop-exits-20261007.json`.
25. 완료: RHS를 먼저 고정한 뒤 live 이전 세대만 정리하고 새 donor를 설치. 자기 이동·소비 후 복원·factory 호출과 scalar Result ? 실패/성공을 구분하며 MaybeLive는 기존 Flow 검사로 거부. resource-bearing Result의 직접 make()? 실행은 27번과 함께 검증한다. 근거: `evidence/candidate-resource-overwrite-20261007.json`.
26. 부분 초기화 실패와 중첩 resource 필드 정리 순서 처리.
27. 기존 계약이 허용하는 resource 필드·집합 값 이동 지원.
28. 실제 소스에 쓰이는 resource ability와 명시적 burn 규칙 연결.
29. 기존 lexical defer와 resource cleanup 실행 순서 통합.
30. 일반 trap이 cleanup을 풀지 않는 기존 계약을 생성 코드에도 유지.

## C. 자기 소스에 필요한 borrow·뷰 확장

31. 스칼라·구조체·resource place 참조와 역참조 구현.
32. 필드·인덱스 공유 대여에서 실제 소유자 추적.
33. 필드·인덱스 가변 대여와 참조를 통한 수정 지원.
34. 공유 재대여, 소유자 수정 제한, 마지막 사용 이후 대여 종료 연결.
35. 가변 재대여 중 부모 사용 제한과 종료 후 부모 사용 재개.
36. 뒤쪽 인자 평가 중 앞쪽 인자의 재대여 유지.
37. 참조·뷰 반환 origin 요약과 호출자 origin 치환.
38. 실제 소스에 필요한 혼합·재귀 반환 origin 처리. 지원 전에는 추측하지 않고 거부.
39. 공개 언어 계약이 허용하는 참조·뷰 보유 집합 타입 지원.
40. 분기·반복·조기 종료에서 borrow·loan 상태 합류.
41. 필요한 ref str·바이트 메타데이터를 SDK와 연결하고 텍스트 bracket 제한 유지.
42. 실제 사용과 기존 계약이 뒷받침하는 경우에만 임시·필드 배열 뷰 확장.
43. ADR0415 제한 유지: 가변 부분 뷰를 여러 가변 sibling 동시 보유로 확대하지 않기.

## D. 제네릭·SDK·컴파일러가 사용하는 언어 기능

44. 스칼라 제네릭 함수의 타입 인자·치환·시그니처 정체성 구현.
45. 특수화 계획과 동일 특수화 중복 제거.
46. 이미 허용된 SDK Vec<T> 타입·필드 레이아웃 연결.
47. Vec 할당·초기화된 prefix·push/pop·대여 접근·cleanup 연결.
48. Vec의 resource 원소, 할당 실패 시 donor 반환, 토큰·진단의 중첩 자원 정리.
49. Buffer/String 할당·크기 변경·바이트 접근·소유권 전달 연결.
50. CLI의 UTF-8 검사·프로세스 인자·콘솔 출력 연결.
51. File open/read/write/sync/close/replace 연결.
52. SDK enum·오류·태그 타입을 일반 import 경로에서 해석.
53. effects 선언과 호출 효과 전파 검사.
54. unsafe 범위와 기존 extern-C 선언·호출 검사.
55. SDK/runtime의 C ABI 스칼라·포인터·집합 값 전달 연결.
56. 이미 파싱하는 @test, @must_use, @effects, @const_eval 등의 의미 연결.
57. 모듈 상수·제한된 스칼라 컴파일 타임 계산·static_assert 연결.
58. 정확한 source_location 전개와 선언 문서 주석 연결.
59. 도달 가능한 소스에서 쓰는 경우 바이트 읽기 반복·region 할당 의미 연결.
60. 자기 소스의 거부 문법을 계약·담당별로 분류. 현재 후보가 이름 검사에서 거부하는 `as` 캐스트도 포함. 스크립트 대체 컴파일로 우회하지 않기.

현재 30비트 tagged 타입 표현은 nominal/resource leaf 하나에 14비트를 쓴다.
따라서 예시 조합 `Result<Option<Token>,Problem>`에서 양쪽 leaf가 nominal/resource라면
32비트가 필요해 표현하지 못한다. 이 예시가 현재 자기 소스에 존재한다고 주장하지 않는다.
실제 `Result<Graph,state::Problem>` 등 도달 가능한 조합부터 inventory에 넣고,
표현할 수 없는 필수 조합이 확인되면 선언 정체성을 유지하는 소유 타입 테이블 등으로 바꾼다.
중첩 깊이·레이아웃·분석량 상한은 타입 ID의 비트 상한과 별도로 유지한다.

## E. 모듈·패키지·실제로 쓸 수 있는 컴파일러 명령

61. 기존 스칼라 다중 파일 검사에 nominal/tagged/resource 타입 연결.
62. 모듈 간 선언 정체성과 private/public 범위 연결.
63. 가져온 함수 시그니처·제네릭 인스턴스·반환 origin 요약 연결.
64. 모듈 간 HIR/MIR·호출·레이아웃 순서·C 심벌 이름 통합.
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

즉시 다음 작업은 **21번 비어 있지 않은 cleanup 본문과 receiver 검사 → 자원 MIR/C 실행 연결**이다.
선택 기능은 별도 도입 판단을 거친다. 이 목록은 push·PR·설치·release를 승인하지 않는다.
