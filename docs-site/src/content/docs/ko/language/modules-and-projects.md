---
title: 프로젝트 구성, 모듈과 의존성
description: fairy.toml 매니페스트, fairy.lock, 모듈 가시성, 로컬 워크스페이스 의존성, tests/ 테스트 구성을 설명합니다.
---

Fairy C 패키지는 루트의 `fairy.toml` 매니페스트, 선택적 로컬 워크스페이스 의존성, 그리고 `fairy.lock` 잠금 파일을 중심으로 구성됩니다.

## `fairy.toml` 매니페스트 (`fairyc.manifest/v1`)

`fairy.toml`은 닫힌 스키마(closed schema)를 사용하므로 정의되지 않은 필드는 무시되지 않고 오류로 거부됩니다.

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

- **`uid`**: 점(`.`)으로 구분된 두 개 이상의 소문자 케밥 세그먼트 식별자입니다.
- **`name`**: 소문자 케밥 패키지 이름으로, 빌드 산출물(`build/<name>.exe`) 이름에 쓰입니다.
- **`version`**: 정확한 SemVer `MAJOR.MINOR.PATCH` 버전만 허용합니다(버전 범위 불가).
- **`language`**: 정확한 언어 리비전 문자열(현재 Stage 0에서는 `"fairyc-language/20"`)을 적습니다.
- **`source`**: 소스 루트 상대 경로(예: `"src"`)를 명시적으로 지정합니다.

## 모듈, 임포트, 가시성

- **경로와 모듈 이름**: `.fy` 파일 상단의 `module ...;`은 `source` 아래 상대 경로와 일치해야 합니다(`E0002`).
- **명시적 별칭 임포트**: 와일드카드 임포트(`*`)는 허용되지 않으며(`E0003`), 반드시 `as <별칭>`을 붙입니다.

```c
module main;

import answer as answer;

fn i32 main() {
    return answer::value();
}
```

- **`pub` 가시성**: 함수·구조체·리소스는 기본적으로 모듈 비공개이며, 다른 모듈에서 쓰려면 `pub`을 붙여야 합니다(`E0103`).

## 로컬 워크스페이스와 경로 의존성

여러 로컬 패키지를 묶을 때는 루트 `fairy.toml`에 `[workspace]`와 `[dependencies.<이름>]`을 명시합니다(`examples/workspace_quickstart`).

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.workspace-quickstart.app"
name = "workspace-quickstart"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"

[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
```

워크스페이스 규칙:

- `workspace.members`는 정렬되고 중복 없는 명시적 상대 경로 목록이어야 합니다(상위 디렉터리 자동 탐색 없음).
- 각 `[dependencies.<별칭>]`은 `path`, 정확한 `version`, `features` 배열(없으면 `features = []`)을 모두 적어야 합니다.
- C 전처리기 `defines`, `include_paths`, `cflags`나 원격 레지스트리 자동 다운로드는 매니페스트 v1에 포함되지 않습니다.

## `fairy.lock`과 `--locked`

`--locked` 없이 `fairyc check`·`build`·`run`을 실행하면 `fairy.lock`이 생성·갱신됩니다. 반대로 `--locked`를 주면 매니페스트나 로컬 의존성 상태가 잠금 파일과 다를 때 종료 코드 `7`로 즉시 실패합니다.

## `tests/` 테스트 작성과 실행

독립된 `tests/` 폴더의 `.fy` 파일에 `@test`를 붙인 비공개·안전·무인자 `void` 함수를 작성합니다(`E9001`).

```c
module tests::main;

import answer as answer;

@test
fn void answer_is_42() {
    assert(answer::value() == 42);
}
```

실행 명령:

- `fairyc test --locked` (시맨틱 인터프리터 실행)
- `fairyc test --native --locked` (격리된 네이티브 바이너리로 컴파일 후 실행)
- `fairyc test --list --json --locked` (테스트 목록 JSON 조회)
