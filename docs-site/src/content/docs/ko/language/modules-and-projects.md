---
title: 모듈, 패키지, 표준 라이브러리
description: Fairy C의 module과 import, pub 가시성, fairy.toml 워크스페이스 구성, 그리고 std 표준 모듈 구성을 알아봅니다.
---

단일 파일 연습을 넘어가면 기능을 여러 파일과 패키지로 나누고, 표준 라이브러리를 가져다 쓰게 됩니다. Fairy C는 헤더 파일(`#include`)이나 매크로 전처리기 없이 깔끔한 모듈 시스템을 씁니다.

## 1. 파일 경로가 곧 `module` 이름

모든 `.fy` 파일 맨 위에는 `module` 선언이 하나 옵니다. 파일 경로가 `src/app/math.fy`라면 모듈 선언도 그 경로에 맞춰 씁니다.

```c
module app::math;

pub fn i32 add(i32 a, i32 b) {
    return a + b;
}
```

- **`pub`**: 다른 모듈에서 가져다 쓸 수 있게 열어 주는 키워드입니다. 안 붙이면 같은 파일/모듈 내부 전용입니다.
- **전처리기 없음**: `#include`, `#define`, 매크로 치환 같은 전처리 단계가 전혀 없습니다.

## 2. `import ... as ...`

다른 모듈이나 패키지의 기능을 가져올 때는 항상 별칭(`as 이름`)을 지정합니다. 별칭으로 네임스페이스가 명확히 갈리기 때문에 이름 충돌이 일어나지 않습니다.

```c
module main;

import app::math as math;
import std::io as io;

@effects(console)
fn i32 main() {
    let i32 v = math::add(20, 22);
    discard io::write_i32(v);
    return 0;
}
```

와일드카드 임포트(`import ...::*;`)나 순환 임포트는 허용되지 않습니다.

## 3. `fairy.toml`과 `fairy.lock`

패키지 설정은 `fairy.toml`에 적습니다. 직접 의존하는 로컬 경로나 표준 라이브러리를 `[dependencies]` 아래에 연결하면 됩니다.

```toml
[package]
name = "my_app"
uid = "example.my-app"
version = "0.1.0"

[dependencies]
answer = { path = "../libs/answer" }
```

- `[workspace]`를 루트 `fairy.toml`에 두면 여러 앱/라이브러리 패키지를 한 묶음으로 관리할 수 있습니다.
- `fairyc check`나 `fairyc test`를 실행하면 해석된 패키지 그래프가 `fairy.lock`에 기록됩니다. 빌드 재현성이 필요할 때는 `--locked` 옵션을 붙여 잠금 파일과 다를 경우 즉시 멈추게 할 수 있습니다.
- Windows GUI 앱처럼 링킹 설정이 필요할 때는 `fairy.windows.json` 매니페스트를 함께 둡니다.

## 4. 현재 제공되는 `std` 표준 모듈

현재 저장소의 `examples/`에서 실제 사용하는 주요 표준 모듈은 다음과 같습니다. (사용하려면 프로젝트에 `std` 의존성이나 해당 SDK 루트가 연결되어 있어야 합니다.)

| 모듈 | 주요 역할 | 대표 기능 감각 |
| --- | --- | --- |
| `std::io` | 콘솔 입출력 (`@effects(console)`) | `write`, `write_i32`, `write_i64`, `write_bytes`, `write_stderr` |
| `std::vec` | 동적 가변 배열 `Vec<T>` | `empty<T>()`, `push<T>`, `pop<T>`, `at<T>`, `truncate<T>`, `clear<T>` |
| `std::text` | UTF-8 문자열 & CString 도구 | `validate_utf8`, `copy`, `append`, `as_bytes`, `cstring_from_text` |
| `std::bytes` | 바이트 버퍼와 슬라이스 도우미 | 바이트 비교/조작 유틸리티 |
| `std::fs` | 파일 열기·읽기·쓰기 (`@effects(filesystem)`) | `open_read`, `create_truncate`, `read`, `write_all`, `seek_start`, `close` |
| `std::num` | 안전한 문자열 파싱과 좁히기 변환 | `parse_i64_decimal`, `usize_from_i64`, `u8_from_i64` |
| `std::process` | 실행 인자·프로세스 접근 | `process::arg(...)` 등 |
| `std::alloc` | 명시적 메모리 할당 인터페이스 | 명시적 할당자 기반 버퍼/자원 관리 |

이제 기본 문법부터 모듈 구성까지 모두 훑었으니, 실제 C 연동과 Win32 예제로 넘어가 보겠습니다!

- 이전: [소유권, 빌림, resource와 region](../memory-and-resources/)
- 다음: [C 연동과 Win32 앱](../../examples/c-interop-and-win32/)
