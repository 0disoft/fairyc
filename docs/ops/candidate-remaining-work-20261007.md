# Fairy 컴파일러 전환: 남은 작업 117개

2026-10-07 현재 소스와 진행표를 기준으로 정리했다. 번호 11~127의 117개
항목이며, 항목마다 별도 커밋이나 전체 테스트를 요구하는 목록은 아니다.
의존성이 맞는 작은 구현 단위로 묶어 순차 커밋한다. 12~20번은 기존 승인 순서를
유지한다. Linux 검증과 빌드 비용 측정은 요청대로 뒤에 묶어서 수행한다.

근거는 candidate-tagged-ownership-progress.md, examples/lexer_candidate/README.md와
실제 Fairy 후보 소스, 기존 Go semantic/MIR/package/CLI 구현,
docs/architecture/self-hosting-readiness-v0.md다.

Go에 이미 있는 기능은 Fairy 후보에 이식·통합할 작업이라는 뜻이다.
후보에 이미 있는 스칼라 배열, 정적 문자열·바이트 리터럴, 바이트 뷰,
스칼라 구조체, 정수·bool enum payload, 스칼라 Option/Result 실행을
처음부터 다시 만들지는 않는다. 파싱과 타입 검사·실행 지원은 구분한다.
마지막 선택 기능은 셀프호스팅 필수 조건에 포함하지 않는다.

## A. 현재 비교 문제와 승인된 12~20번

11. **Result 처리 의무의 Go/후보 차이 해소.** ? 오류 반환 전에 다른 미처리
    Result가 남는 경우 Go는 허용하고 후보는 거부한다. 정상 경로의 discard와
    오류 경로의 반환을 문서·구현·재현으로 대조해 결론을 반영한다.
    후보 자체 검증은 통과했지만 기준 구현과의 일치 승인은 보류 상태다.
12. 자기 소스에 필요한 중첩 Option/Result의 타입·생성·패턴·실행 지원.
13. Option/Result의 구조체·진단 payload와 레이아웃 의존 순서·패턴 캡처 지원.
14. 구조체 필드·배열에 enum/Option/Result 저장, 복사·인덱싱·대입·초기화 지원.
15. 선언별 resource 타입 정체성과 필드 소유권·표현 정보 연결.
16. 명시적 resource factory와 생성 HIR 연결.
17. 바인딩·대입·인자·반환에서 resource 전체 값 이동 구현.
18. 이동 후 사용 오류와 이동·사용 위치 진단 구현.
19. 분기 합류에서 사용 가능·이동됨·일부 경로에서 이동됨 상태 구분.
20. 반복 진입·재진입·break·continue에서 이동 상태 분석.

11번 소스·원시 실행·도구 해시는 evidence/candidate-result-handling-20261007.json에
있다. 이 단위에서 공개 Go 소스는 수정하지 않았다. 비교 차이가 해소되기 전에는
후보를 기준 컴파일러로 승격하지 않는다.

## B. 이동 검사 이후 resource 수명 관리

21. cleanup 본문과 receiver 소유권, 허용 연산 검사.
22. 정상 스코프 종료에서 자원을 계약 순서대로 한 번씩 정리.
23. 직접 return과 Result ?의 성공·오류 경로에 cleanup 연결.
24. 반복 본문 종료·break·continue에 cleanup 연결.
25. 자원 덮어쓰기에서 이전 세대 정리와 donor 이동 구분.
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
60. 자기 소스의 거부 문법을 계약·담당별로 분류. 스크립트 대체 컴파일로 우회하지 않기.

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
75. 컴파일러 프로세스 timeout과 자식 종료 정책 정리.
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
85. 노드·할당·깊이·분석량 상한과 초과 시 실패 처리. 부분 성공 금지.
86. 실제 누락에 직접 대응하는 회귀 추가. 전체 검사 반복과 구현을 베낀 테스트 줄이기.
87. 진행표·readiness 문서를 실제로 유지되는 구현과 실행 결과에 맞춰 갱신.

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
98. Clang 전처리 line-marker/runtime-cache 문제 해소. 이전 GCC 한정 수정 범위 확인.
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

120. 현재 바이트 뷰를 넘어 일반 타입 비소유 뷰 확장.
121. 현재 SDK 예외를 넘어 일반 사용자 제네릭 집합 타입 확장.
122. 실제 여러 타입 알고리즘에 필요한 작은 정적 제약.
123. 실제 요구와 빌드 비용이 뒷받침하는 컴파일 타임 계산 확장.
124. 일반 구조체로 해결하기 어려운 도메인 혼동이 입증된 뒤 nominal scalar 축약 판단.
125. 더 넓은 public tagged-aggregate FFI·추가 target·편집기 LSP 기능.
126. C 출력 경로가 측정된 목표를 막을 때 직접 native backend·추가 최적화 판단.
127. 별도 제품 목표가 선택된 뒤 contract/on-chain profile 확장.

즉시 다음 작업은 **11번 비교 차이 해소 → 12번 → 13~20번**이다.
선택 기능은 별도 도입 판단을 거친다. 이 목록은 push·PR·설치·release를 승인하지 않는다.
