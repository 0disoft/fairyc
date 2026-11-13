---
title: 현재 지원 범위와 제한 사항
description: Stage 0 프로토타입의 실제 동작 범위, fairyc targets 기준 타깃 상태, 아직 지원하지 않는 기능을 설명합니다.
---

Fairy C는 현재 활발히 개발 중인 **Stage 0 컴파일러 프로토타입**(`fairyc-language/20`, `fairyc-sdk-stdlib/0 0.0.0-stage0.87`)입니다. 현재 실제로 사용 가능한 범위와 아직 검증·완성되지 않은 범위를 구분해 두었습니다.

## 타깃 지원 상태 (`fairyc targets`)

`fairyc targets`를 실행하면 두 타깃 사양이 출력되며, 공식 타깃 사양 적합성 기준에서는 둘 다 `supported: no`로 표시됩니다.

- `x86_64-windows-msvc`: `emittable: no`, `conformant: no`, `supported: no`
- `x86_64-linux-gnu`: `emittable: no`, `conformant: no`, `supported: no`

실제 의미는 다음과 같습니다.

- **실제 동작하는 부분**: Windows의 MSVC/Windows SDK 또는 Linux의 GCC/Clang이 준비된 환경에서, 구현된 Stage 0 언어 부분집합과 예제(Win32/C 연동 포함)는 `fairyc check`, `build`, `run`, `test`로 정상 빌드 및 실행됩니다.
- **`supported: no`가 뜻하는 부분**: 양방향 C ABI 전체 적합성 테스트, Windows x64 언와인드 픽스처, 완전한 표준 라이브러리 및 런타임 ABI, 툴체인 패키징 배포 등 정식 릴리스 수준의 증명이 아직 남아 있습니다.

## 현재 사용 가능한 기능

- `fairy.toml` + `fairy.lock` 기반 단일 패키지 및 로컬 `[workspace]` 경로 의존성 프로젝트
- 이동 전용 `resource`와 `cleanup`, 전체 루트 기준 `ref` / `ref mut` / 슬라이스 뷰 빌림 검사, 단일 출처 반환 규칙
- 검사된 정수 산술과 `as` 명시적 형변환
- `Option<T>`, `Result<T, E>`(`?` 전파, 미처리 오류 진단), `@must_use`, `@const_eval`, `static_assert`, `defer`, `@effects(...)`
- C ABI 연동(`@repr(C)`, `@link_name`, `unsafe extern C fn`, 동기 C 콜백)
- Stage 0 SDK 모듈 일부(`std::io`, `std::fs`, `std::bytes`, `std::alloc`, `std::text`, `std::vec`, `std::process`)

## 제한 사항 및 미지원 영역

- **전 플랫폼·완제품 보장 없음**: 모든 운영체제나 임의의 C/C++ 라이브러리와의 완벽한 호환성이나 범용 성능 우위는 아직 보장하지 않습니다.
- **제한적인 표준 라이브러리**: 현재 SDK에 구현된 모듈 외에 비동기 런타임, 고수준 HTTP/GUI 프레임워크 등은 포함되지 않습니다.
- **원격 패키지 레지스트리 없음**: 안정 CLI에서는 로컬 워크스페이스 경로 의존성만 다루며 자동 원격 패키지 다운로드나 업데이터는 없습니다.
- **문법·타입 기능 제한**: 제네릭은 명시적 스칼라 인자 제약이 있으며(`E45xx`), 참조를 반환하는 함수 포인터나 일반 클로저·안전 콜백(`E2104`, `E5104`)은 제한됩니다. `f32`/`f64`도 주로 `@repr(C)` 구조체 필드용입니다.
- **실험 명령 제외**: `fairyc experimental ...` 하위 명령은 안정 계약이 아니므로 언제든 변경될 수 있습니다.
