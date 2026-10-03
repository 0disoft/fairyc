# 정수 고정 배열 뷰 실행 결과

2026-10-04, Windows x64 MSVC에서 일반·릴리스 32회 실행을 통과했다.
i32/u32/usize의 공유·가변 뷰, 극값, 부분 구간, 공유 재대여, Option 반환,
빈 구간, 경계식 평가 순서, 마지막 사용 후 원본 재사용을 실제 Fairy 코드로 확인했다.
끝 읽기/쓰기, 빈 구간 쓰기, 역방향·범위 초과 구간은 bounds trap으로 종료했다.

HIR·semantic·MIR·interpreter·C backend 패키지 테스트, 뷰·캐시 관련 compiler
테스트, 소유 패키지 go vet, 공개 CLI check/build/run 5회가 통과했다.
각 타입에 MIR 변조 7개와 대여 오류 8개를 검사했다.
기존 Vec 캐시 테스트의 전역 storage-borrow 개수 기대값은 SDK 뷰 wrapper까지
세고 있었으므로, 이 테스트의 목적에 맞게 두 인자 원소 대여만 세도록 수정했다.
긴 스캐너 검증이 포함된 compiler 전체 실행은 중단하고 관련 테스트로 좁혔다.

소스·SDK·compiler·EXE 해시와 원시 실행은
[증거](../../docs/ops/evidence/integer-views-20261004.json)에 있다.
배열 크기는 4096 원소 제한, 가변 부모·형제 충돌은 ADR 0415 정책이다.
Linux 실제 실행은 사용자 요청에 따라 이후 일괄 검증한다. 성능 실험과 push는 하지 않았다.
PS1 파일이나 runtime C 함수를 추가하지 않았다.
