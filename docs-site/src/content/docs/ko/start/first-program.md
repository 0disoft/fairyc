---
title: 첫 프로그램
description: 새 Fairy C 프로젝트를 만들고, 생성된 코드를 읽어 보며 검사·테스트·실행까지 해 봅니다.
---

이제 직접 첫 번째 Fairy C 프로그램을 만들고 실행해 봅시다.

## 1. 새 프로젝트 만들기

컴파일러 저장소 바깥의 빈 폴더로 이동한 뒤 `fairyc new`로 시작 프로젝트를 만듭니다. 이때 패키지를 구분하는 고유 이름(`--uid`)을 꼭 함께 적어 주어야 합니다.

```powershell
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
```

명령이 성공하면 폴더 안에 다섯 파일이 생깁니다.

- `fairy.toml`: 프로젝트 설정 파일
- `src/main.fy`: 실제 프로그램 코드가 들어가는 곳
- `tests/main.fy`: 테스트 코드가 들어가는 곳
- `.gitignore`, `README.md`

먼저 `fairy.toml`을 열어 보면 이렇게 생겼습니다.

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

그다음 `src/main.fy` 코드를 봅시다. 아주 간단합니다.

```c
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

한 줄씩 뜯어볼까요?

1. `module main;`: 이 파일의 모듈 이름입니다. 진입 파일인 `src/main.fy`는 `main` 모듈이어야 합니다.
2. `pub fn i32 answer()`: C와 비슷하지만 함수 이름(`answer`) 앞에 반환 타입 `i32`가 오고, 앞에 `fn`이 붙습니다. `pub`은 테스트 파일 같은 다른 모듈에서도 이 함수를 부를 수 있게 열어 준다는 뜻입니다.
3. `fn i32 main()`: 프로그램이 시작하는 지점입니다. 여기서 `return`한 `i32` 숫자가 운영체제에 돌려주는 프로세스 종료 코드(exit code)가 됩니다. `42 - 42`는 `0`이므로 정상 종료를 뜻하게 됩니다.

## 2. 문법·타입 검사하기 (`check`)

실행 파일을 만들기 전에 먼저 코드에 문제가 없는지 검사해 봅시다.

```powershell
fairyc check
```

정상이라면 이런 문장이 나옵니다.

```text
check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated
```

처음 `check`를 돌리면 폴더에 `fairy.lock` 파일이 자동으로 만들어집니다. 그다음부터는 `--locked`를 붙여 실행할 수 있는데, "잠금 파일에 기록된 구성 그대로 검사해 달라"는 뜻입니다.

```powershell
fairyc check --locked
```

## 3. 테스트 돌려보기 (`test`)

`tests/main.fy`에는 이미 `answer() == 42`를 확인하는 테스트가 하나 들어 있습니다. 새 테스트를 추가할 때는 테스트 함수 위에 `@test`를 붙이면 됩니다.

```c
module tests::main;

import main as app;

@test
fn void answer_is_42() {
    assert(app::answer() == 42);
}
```

테스트를 실행해 볼까요?

```powershell
fairyc test --locked
```

출력:

```text
test tests::main::answer_is_42 ... ok
test result: ok. 1 passed; 0 failed; lock verified
```

실제 네이티브 실행 파일로 컴파일해서 테스트하고 싶다면 `fairyc test --native --locked`를 쓰면 됩니다.

## 4. 직접 실행하고 빌드 파일 남기기 (`run`, `build`)

임시 실행 파일을 만들어 곧바로 실행하고 치워 주려면 `run`을 씁니다.

```powershell
fairyc run --locked
```

`main()`이 `0`을 반환했으니 아무 오류 없이 종료 코드 `0`으로 끝납니다. 만약 `build/hello-fairy.exe`(Linux는 `build/hello-fairy`)처럼 실행 파일을 폴더에 남겨 두고 싶다면 `build` 명령을 씁니다.

```powershell
fairyc build --release --locked
```

## 처음 해 볼 때 자주 걸리는 부분

- **`--uid`를 빼먹은 경우**: `fairyc new`는 `--uid example.hello-fairy`처럼 점(`.`)으로 나뉜 두 단어 이상의 소문자 ID가 꼭 필요합니다.
- **이름에 대문자를 쓴 경우**: 패키지 `name`과 `uid`는 모두 소문자와 하이픈(`-`)만 허용합니다.
- **설정을 고친 직후 `--locked`를 쓴 경우**: `fairy.toml`을 고쳤다면 먼저 `--locked` 없이 `fairyc check`를 한 번 실행해 `fairy.lock`을 갱신해야 합니다.
- **`run`/`build`에서 종료 코드 `6`이 난 경우**: C 컴파일러(Windows는 MSVC/SDK, Linux는 `gcc`/`clang`)를 찾지 못했다는 뜻이니 터미널 환경을 확인해 주세요.
