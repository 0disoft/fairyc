---
title: 프로젝트 구성, 모듈과 의존성
description: 코드가 커질 때 파일을 모듈로 나누는 방법, fairy.toml 구조, 로컬 워크스페이스, 테스트 작성법을 알아봅니다.
---

프로그램이 커지면 `src/main.fy` 한 파일에 다 넣지 않고 여러 파일과 패키지로 나누게 됩니다. Fairy C에서는 파일 경로, `module` 선언, `fairy.toml`이 서로 딱 맞아떨어지게 설계되어 있습니다.

## 1. 파일 나누기와 `module` / `import`

`src/` 폴더 아래에 새 파일을 만들었다면 파일 맨 위에 **경로에 맞는 `module` 선언**을 적습니다(`E0002` 방지).

- `src/main.fy` → `module main;`
- `src/helper.fy` → `module helper;` (하위 폴더 `src/net/http.fy`면 `module net::http;`)

다른 파일에서 함수를 가져다 쓸 때는 `import <모듈> as <별칭>;` 형태로 별칭을 꼭 붙여야 합니다. `import foo::*;` 같은 와일드카드 임포트는 어디서 온 이름인지 흐려지기 때문에 허용되지 않습니다(`E0003`).

```c
module main;

import answer as answer;

fn i32 main() {
    return answer::value();
}
```

또 하나 기억할 점은 **기본이 비공개(private)**라는 것입니다. 다른 모듈에서 부를 함수·구조체·리소스 앞에는 꼭 `pub`을 붙여 주세요(`pub fn`, `pub struct`, `pub resource`). `pub`이 빠지면 `E0103` 에러가 납니다.

## 2. `fairy.toml` 살펴보기

프로젝트 루트의 `fairy.toml`은 정해진 필드만 허용합니다. 오타가 난 필드를 조용히 무시하지 않고 바로 알려 줍니다.

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

- `uid`: 점(`.`)으로 나뉜 소문자 고유 ID (예: `example.hello-fairy`)
- `name`: 빌드 결과 파일(`build/<name>.exe`)에 쓰이는 소문자 이름
- `version`: 정확한 `MAJOR.MINOR.PATCH` 버전
- `language`: 현재 컴파일러 버전 문자열(`"fairyc-language/20"`)
- `source`: 소스 코드가 들어 있는 디렉터리(`"src"`)

## 3. 같은 저장소 안의 다른 패키지 연결하기 (`[workspace]`)

예를 들어 `libs/answer` 폴더에 별도 라이브러리 패키지를 두고 메인 앱에서 쓰고 싶다면, 루트 `fairy.toml`에 `[workspace]`와 `[dependencies.<별칭>]`을 추가합니다(`examples/workspace_quickstart` 방식).

```toml
[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "1.0.0"
features = []
```

현재 안정 CLI는 이렇게 **로컬 경로로 연결된 워크스페이스 의존성**만 다룹니다. 원격 패키지 다운로드나 C 컴파일 옵션(`cflags`, `defines`)은 `fairy.toml`에 들어가지 않습니다.

## 4. `fairy.lock`과 `--locked`

`fairyc check`, `build`, `run`을 실행하면 현재 패키지 구성이 `fairy.lock`에 기록됩니다. 그 뒤로 `--locked` 옵션을 붙여 실행하면 "내가 마지막으로 확인한 구성과 바뀐 게 없을 때만 통과시켜 달라"고 검사하므로 협업이나 CI 환경에서 유용합니다.

## 5. `tests/` 폴더에 테스트 추가하기

테스트 코드는 `src/`와 분리된 `tests/` 폴더 아래에 둡니다. 인자가 없고 `void`를 반환하는 비공개 함수 위에 `@test`를 붙이면 됩니다.

```c
module tests::main;

import answer as answer;

@test
fn void answer_is_42() {
    assert(answer::value() == 42);
}
```

- `fairyc test --locked`: 인터프리터로 빠르게 실행
- `fairyc test --native --locked`: 실제 네이티브 바이너리로 빌드해서 실행
- `fairyc test --list --json --locked`: 등록된 테스트 목록 확인
