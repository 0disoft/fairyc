---
title: 첫 프로그램
description: 새 Fairy C 패키지를 만들고, 정적 검사·테스트·네이티브 실행 및 빌드까지 순서대로 진행합니다.
---

이 가이드에서는 `fairyc new`로 시작 패키지를 만든 뒤, 정적 검사와 테스트를 실행하고 네이티브 바이너리로 실행·빌드하는 전체 흐름을 확인합니다.

## 1. 새 패키지 생성

컴파일러 저장소 바깥의 깨끗한 작업 폴더에서 `--uid` 플래그를 포함해 `fairyc new`를 실행합니다.

```powershell
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
```

다음 다섯 파일이 생성됩니다.

- `fairy.toml`: 패키지 매니페스트
- `src/main.fy`: 프로그램 진입점 소스
- `tests/main.fy`: 기본 테스트 모듈
- `.gitignore`, `README.md`

`fairy.toml` 내용은 다음과 같습니다.

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

`src/main.fy` 내용은 다음과 같습니다.

```c
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

실행 가능한 Fairy C 패키지는 `module main;`의 `fn i32 main()`에서 시작하며, 반환한 `i32` 값이 프로세스 종료 코드가 됩니다.

## 2. 정적 검사 실행

문법, 타입, 소유권, 빌림 규칙을 검사하고 `fairy.lock`을 생성합니다.

```powershell
fairyc check
```

예상 출력:

```text
check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated
```

`fairy.lock`이 만들어진 이후에는 `--locked` 옵션을 함께 전달해 의도치 않은 잠금 변경을 막을 수 있습니다.

```powershell
fairyc check --locked
```

## 3. 기본 테스트 실행

시맨틱 인터프리터로 `tests/` 테스트를 실행합니다.

```powershell
fairyc test --locked
```

예상 출력:

```text
test tests::main::answer_is_42 ... ok
test result: ok. 1 passed; 0 failed; lock verified
```

실제 네이티브 실행 이미지로 테스트하려면 `fairyc test --native --locked`를 사용합니다.

## 4. 네이티브 실행 및 빌드

임시 네이티브 바이너리를 만들어 바로 실행합니다.

```powershell
fairyc run --locked
```

`answer() - 42`의 결과가 `0`이므로 프로세스가 종료 코드 `0`으로 끝나며, 폴더에 임시 실행 파일을 남기지 않습니다.

결과 실행 파일을 `build/hello-fairy.exe`(Windows) 또는 `build/hello-fairy`(Linux)로 남기려면 `build` 명령을 사용합니다.

```powershell
fairyc build --release --locked
```

## 처음 시작할 때 자주 겪는 실수

- **`fairyc new`에서 `--uid` 누락**: `--uid`는 필수이며, 소문자 케밥 세그먼트 두 개 이상을 점(`.`)으로 이어 적어야 합니다(예: `example.hello-fairy`).
- **패키지 이름이나 `uid`의 대문자 사용**: 매니페스트 규칙상 소문자 케밥 표기만 허용됩니다.
- **의존성이나 매니페스트 수정 후 바로 `--locked` 사용**: 변경 후 먼저 `fairyc check`로 `fairy.lock`을 갱신해야 `--locked`가 통과합니다.
- **`run`/`build` 시 종료 코드 `6` 발생**: Windows의 MSVC/Windows SDK 또는 Linux의 `gcc`/`clang`이 셸에서 접근 가능한지 확인하세요.
