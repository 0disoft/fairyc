---
title: CLI 사용법과 자주 만나는 오류
description: fairyc 안정 명령어, 주요 플래그, 종료 코드, 그리고 진단 코드별 해결 방법을 정리합니다.
---

안정 `fairyc` CLI는 현재 작업 디렉터리의 `fairy.toml`과 `fairy.lock`을 기준으로 동작합니다.

## 안정 명령어

| 명령어 | 설명 |
| --- | --- |
| `fairyc new <name> --uid <uid>` | 새 패키지 골격(`fairy.toml`, `src/main.fy`, `tests/main.fy`) 생성 |
| `fairyc check [--locked]` | 문법·타입·소유권·빌림 검사 및 `fairy.lock` 생성/검증 |
| `fairyc build [--release] [--locked]` | `build/<package-name>.exe`(Windows) 또는 `build/<package-name>`(Linux) 빌드 |
| `fairyc run [--release] [--locked] [-- <args...>]` | 임시 네이티브 실행 파일 빌드 후 표준 입력/인자를 전달해 실행 |
| `fairyc test [옵션] [--locked]` | 시맨틱 인터프리터 또는 `--native` 네이티브 이미지로 테스트 실행 |
| `fairyc explain <코드>` | `E2101`, `E7102` 같은 진단 코드 상세 설명 출력 |
| `fairyc targets` | 등록된 타깃 사양과 현재 적합성 상태 표시 |
| `fairyc --version` / `--help` | 컴파일러 버전 정보와 도움말 출력 |

> **안정 명령과 실험 명령 구분:** `fairyc experimental ...` 하위 명령(`emit-c`, `atlas-*` 등)은 컴파일러 개발용 불안정 인터페이스이므로 일반 개발 흐름에 의존하지 마세요.

## 주요 플래그

- `--locked`: `fairy.lock`을 갱신하지 않고 기존 잠금 상태와 정확히 일치하는지 검증합니다.
- `--release`: `build`, `run`, `test --native`에서 C 백엔드 최적화 빌드를 사용합니다.
- `--json`: `fairyc test` 및 `fairyc test --list` 결과를 JSON으로 출력합니다.
- `--native` (`fairyc test` 전용): 인터프리터 대신 격리된 네이티브 테스트 바이너리로 실행합니다. `--timeout-seconds <1..60>`, `--stdin-file <경로>`, `--capture-output`(`--json`과 함께 사용)을 지원합니다.

## 프로세스 종료 코드

| 코드 | 의미 |
| --- | --- |
| `0` | 정상 성공 (`fairyc run`에서는 실행한 프로그램도 `0`으로 종료된 경우) |
| `2` | 잘못된 CLI 인자 또는 지원하지 않는 옵션 |
| `3` | 문법, 타입, 빌림, 효과, 또는 매니페스트 정적 오류 |
| `4` | 내부 컴파일러 오류 |
| `5` | Fairy 프로그램 런타임 트랩(오버플로, 0으로 나누기, `assert` 실패 등) |
| `6` | C 툴체인·링커·실행 파일 실행 실패 |
| `7` | 워크스페이스, `fairy.toml`, 또는 `fairy.lock` 불일치 |

`fairyc run`은 프로그램이 트랩(코드 `5`)에 걸리지 않고 정상 `return`으로 종료한 경우 `main()`의 반환값을 그대로 프로세스 종료 코드로 전달합니다.

## 자주 만나는 오류와 진단 코드

오류 메시지의 코드로 `fairyc explain <코드>`를 실행할 수 있으며, 주요 코드군은 다음과 같습니다.

| 코드군 | 주요 원인과 해결 |
| --- | --- |
| `E0001..E0011` | UTF-8 오류, 파일 경로와 `module` 불일치(`E0002`), 와일드카드 import 금지(`E0003`), 괄호 없는 연속 비교(`E0009`). |
| `E1001..E1014` | 초기화 누락(`E1001`), 타입 불일치(`E1006`), 스칼라 범위 초과(`E1013`). `as`로 명시적 변환이 필요합니다. |
| `E1101..E1109` | 이동된 `resource` 재사용(`E1101`), 분기 후 `MaybeLive` 상태(`E1104`). 유지할 값은 `ref`로 빌려 전달하세요. |
| `E2101..E2105` | 대여 충돌(`E2101`) 또는 다중 출처 참조 반환(`E2103`). 가변 수정 전 참조 사용을 끝내세요. |
| `E4101..E4202` | 미처리 `Result`(`E4101`) 또는 무시된 `@must_use` 결과(`E4202`). `switch`, `?`, `discard` 중 하나로 처리하세요. |
| `E5101..E5106` | `unsafe` 경계 위반. 원시 포인터 연산과 `extern C` 호출은 `unsafe { ... }` 안에 있어야 합니다. |
| `E7101..E7102` | `@effects(...)` 누락(`E7102`). 호출 대상의 효과(`console`, `filesystem` 등)를 함수 선언에 추가하세요. |
