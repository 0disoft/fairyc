# 정수 Vec 뷰와 순회 결과

2026-10-04, SDK stage0.83의 i32/u32/usize 공유·가변 Vec 뷰와 for-in을 구현했다.
고정 배열과 Vec에서 같은 순회 규칙을 사용한다. 뷰 생성은 할당·원소 복사 없이
초기화된 영역만 대여하며, 반복마다 읽으므로 뒤쪽 원소의 변경이 다음 반복에 반영된다.

Windows MSVC 일반·릴리스 44회 실행, 대여 오류 102개, 공개 CLI 5회,
갱신된 compiler의 기존 u8/i64 예제 일반·릴리스 실행 4회가 통과했다.
새 세 타입의 storage-transfer MIR 변조 12개씩을 공유·가변 양쪽에서 거부했고,
일반/query-only 경로에서는 private transfer를 거부했다.
고정 배열 순회는 MIR codec 왕복과 interpreter에서 합계·continue·break·쓰기 순서를 확인했다.
AST·SDK·generic 준비·semantic·C backend·module interface·snapshot·artifact·body hash
관련 패키지 테스트, 관련 compiler 테스트와 go vet도 통과했다.

공개 SDK 인터페이스, snapshot 및 함수/본문 identity의 slice 직렬화를 확장해
뷰 원소 타입을 보존한다. 지역 소유자 탈출, shared store, 가변 부모·형제 충돌,
구조 변경·교체·move·effectful bounds와 조기 return의 move를 거부했다.
극값과 빈 구간, 범위 오류 및 순회 중 checked 산술 overflow/underflow를 검사했다.

[증거](../../docs/ops/evidence/vec-integer-views-20261004.json)에 소스·SDK·compiler·EXE 해시,
CLI argv/cwd/stdout/stderr/exit와 native 원시 결과를 남겼다.
Linux 실제 실행은 이후 일괄 검증한다. PS1 runner, 설치, 성능 실험, push는 추가하지 않았다.
SDK interpreter의 host-storage 실행은 제공하지 않으며 Vec는 native에서 검증했다.
