---
title: 언어 기초
description: Fairy C의 변수 선언, 타입 체계, 함수, 제어문, 검사 산술, @effects 표기를 설명합니다.
---

Fairy C는 C와 유사한 문법에 명시적 타입, 중괄호 블록, 오버플로를 잡아내는 검사 산술(checked arithmetic), 그리고 부수 효과 표기(`@effects(...)`)를 결합한 언어입니다.

## 바인딩: `let`과 `var`

- `let`은 불변 바인딩, `var`는 가변 바인딩을 선언합니다.
- 지역 변수는 선언 시 반드시 초기화해야 합니다(`E1001`).
- 정수 리터럴은 지정된 스칼라 타입 범위 안에 있을 때 문맥에 맞게 타입이 결정됩니다.

```c
let i32 limit = 10;
var i32 total = 0;
var u8[3] bytes = [1, 2, 3];
```

## 기본 타입과 복합 타입

- **정수 및 불리언**: `i8`, `i16`, `i32`, `i64`, `u8`, `u16`, `u32`, `u64`, `isize`, `usize`, `bool`.
- **부동소수점**: `f32`, `f64`는 Stage 0에서 `@repr(C)` 레이아웃 연동용으로만 쓰며 일반 안전 산술 타입으로는 제한됩니다.
- **바이트와 문자열**:
  - `b"..."`는 바이트 리터럴, `"..."`는 UTF-8 `str`, `c"..."`는 C 연동용 NUL 종료 `CStr` 리터럴입니다.
  - 배열과 슬라이스 뷰에서는 `.len`과 인덱스 접근(`bytes[0]`)을 지원합니다.
- **고정 배열·구조체·열거형**: `T[N]` (`[0; 512]` 반복 초기화 또는 `[1, 2, 3]` 목록), 명목형 `struct`, `enum`.
- **태그 유니온**: `Option<T>` (`.some(v)` / `.none`) 및 `Result<T, E>` (`.ok(v)` / `.err(e)`).

## 함수, 제어문, 검사된 산술

함수 반환 타입은 이름 앞에 오며, `void`가 아닌 모든 경로는 `return`으로 끝나야 합니다.

```c
struct State { i32 number; }

fn i32 State::read(ref State this) {
    return this.number;
}

fn i32 classify(i32 x) {
    return if (x > 0) { 1 } else { 0 };
}
```

- **분기와 반복**: `if` / `else`, `switch`, `while`, C 스타일 `for`, 뷰 순회 `for-in`.
- **`defer`**: 스코프를 빠져나가기 직전에 정리 코드를 실행합니다.
- **검사 산술**: 정수 `+`, `-`, `*`, `/`, 시프트 연산은 오버플로나 0으로 나누기가 발생하면 조용히 넘치지 않고 런타임 트랩(종료 코드 `5`)으로 중단됩니다.
- **명시적 형변환**: 암묵적 숫자 승격이 없으므로 `value as i32`처럼 `as`를 명시해야 합니다.

## `Result<T, E>` 처리와 `@effects(...)`

Fairy C에서는 `Result` 반환값을 무시할 수 없습니다. `switch`로 분기하거나 `?`로 전파하거나 `discard`로 명시적 폐기해야 합니다(`E4101`).

```c
enum Error { Failed; }

fn Result<i32, Error> propagate(Result<i32, Error> input, Error fallback) {
    let i32 value = input?;
    return .err(fallback);
}

fn i32 ignore(Result<i32, Error> input) {
    discard input;
    return 0;
}
```

콘솔 출력, 파일시스템, 환경 변수, 프로세스 인자, 네트워크, 시계, 난수에 접근하는 함수는 `@effects(...)`에 해당 권한(`console`, `filesystem`, `environment`, `process`, `network`, `clock`, `random`)을 선언해야 합니다.

```c
import std::io as io;

@effects(console)
fn bool print_number(u32 value) {
    return switch (io::write_u32(value)) {
        case .ok(_): { true }
        case .err(_): { false }
    };
}
```

호출한 함수나 암묵적으로 정리되는 `resource`가 요구하는 효과를 호출자 함수에 빠뜨리면 `E7102` 컴파일 오류가 발생합니다.

## 자주 하는 실수

- **`i32`와 `usize` 혼합**: 암묵적 승격이 없으므로 `as`로 명시적 변환이 필요합니다.
- **연속 비교식 생략**: `a < b < c`는 허용되지 않으며 `(a < b) && (b < c)`처럼 써야 합니다(`E0009`).
- **`Result` 또는 `@must_use` 반환값 방치**: `switch`, `?`, 또는 `discard expr;`로 처리해야 합니다.
