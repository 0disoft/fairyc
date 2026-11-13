---
title: 언어 기초
description: Fairy C의 변수 선언, 타입, 함수, 조건문·반복문, Result 처리와 @effects 문법을 차근차근 배웁니다.
---

Fairy C 문법은 C와 아주 비슷해서 C, Java, JavaScript를 조금이라도 본 적이 있다면 금방 눈에 들어옵니다. 다만 "실수로 넘어가기 쉬운 부분"을 문법에서 몇 가지 더 엄격하게 잡아 줍니다.

## 1. 변수 선언: `let`과 `var`

변수를 만들 때는 값을 바꿀 수 있는지 없는지에 따라 `let`과 `var`를 구분해서 씁니다. JavaScript의 `const`/`let`과 비슷하지만, Fairy C에서는 **선언할 때 반드시 초기값을 넣어야 합니다**. 초기화하지 않은 쓰레기 값으로 사고가 나는 걸 막기 위해서입니다.

```c
let i32 limit = 10;       // 나중에 값을 바꿀 수 없음
var i32 total = 0;        // 나중에 total = ... 으로 바꿀 수 있음
var u8[3] bytes = [1, 2, 3];
```

## 2. 기본 타입과 명시적 형변환 (`as`)

- **정수**: 부호 있는 `i8`, `i16`, `i32`, `i64`, `isize` / 부호 없는 `u8`, `u16`, `u32`, `u64`, `usize`
- **참/거짓**: `bool` (`true`, `false`)
- **문자열과 바이트**:
  - `"hello"`: UTF-8 문자열(`str`)
  - `b"hello"`: 바이트 배열/슬라이스
  - `c"hello"`: C 함수에 넘길 때 쓰는 NUL(`\0`) 종료 C 문자열(`CStr`)
- **고정 배열·구조체·열거형**: `T[N]` (`[0; 10]`처럼 같은 값으로 채울 수도 있습니다), `struct`, `enum`

여기서 C와 다른 중요한 점이 두 가지 있습니다.

첫째, **서로 다른 숫자 타입을 자동으로 섞어 주지 않습니다**. `u8` 값과 `i32` 값을 더하려면 아래처럼 `as`로 직접 타입을 맞춰 주어야 합니다.

```c
let u8 byte_val = 40;
let i32 total = (byte_val as i32) + 2;
```

둘째, **정수 오버플로가 나면 멈춥니다**. C에서는 `i32` 최댓값에 `1`을 더하면 조용히 음수로 뒤집히지만, Fairy C는 실행을 즉시 중단(trap)해서 잘못된 계산 결과가 퍼지는 걸 막아 줍니다.

## 3. 함수, `if`, `switch`, 반복문

함수는 `fn 반환타입 이름(매개변수)` 형태로 적습니다. 구조체 뒤에 `::`를 붙여 메서드처럼 만들 수도 있습니다.

```c
struct State { i32 number; }

fn i32 State::read(ref State this) {
    return this.number;
}

fn i32 classify(i32 x) {
    // if-else 자체가 값을 돌려주는 표현식으로도 쓰입니다.
    return if (x > 0) { 1 } else { 0 };
}
```

- **반복문**: `while (조건) { ... }`, C 스타일 `for (var usize i = 0; i < n; i += 1) { ... }`, 그리고 슬라이스를 순회하는 `for-in`을 지원합니다.
- **비교 연산 주의**: `0 < x < 10`처럼 비교를 연달아 쓰면 컴파일 에러(`E0009`)가 납니다. `(0 < x) && (x < 10)`처럼 괄호로 명확하게 묶어 주세요.
- **`defer`**: Go의 `defer`처럼, 현재 블록을 빠져나가기 직전에 꼭 실행할 정리 코드를 미리 예약할 때 씁니다.

## 4. `Option<T>`와 `Result<T, E>`

Fairy C에는 빈 포인터 `null` 대신, "값이 있을 수도 없을 수도 있음"을 표현하는 `Option<T>`(`.some(값)` / `.none`)와, "성공 또는 실패"를 표현하는 `Result<T, E>`(`.ok(값)` / `.err(오류)`)가 있습니다.

특히 함수가 `Result`를 돌려줬는데 호출한 쪽에서 **그 결과를 확인하지 않고 그냥 넘어가면 컴파일 에러(`E4101`)가 납니다**. 에러를 꼭 처리하거나, 정말 버려도 될 때는 `discard`를 적어 의도를 분명히 해야 합니다.

```c
enum Error { Failed; }

fn Result<i32, Error> add_positive(Result<i32, Error> input) {
    // input이 .err(...)면 여기서 곧바로 에러를 돌려주고 함수가 끝납니다.
    let i32 value = input?;
    return .ok(value + 1);
}

fn i32 ignore_result(Result<i32, Error> input) {
    discard input; // "이 결과는 일부러 안 쓰겠다"는 표시
    return 0;
}
```

`?` 연산자 덕분에 `switch`를 길게 중첩하지 않고도 실패를 깔끔하게 바깥으로 전달할 수 있습니다.

## 5. `@effects(...)` 이름표

화면에 글자를 출력하거나(`console`), 파일을 읽고 쓰거나(`filesystem`), 시계를 보거나(`clock`), 프로그램 인자를 읽는(`process`) 함수들은 선언 위에 `@effects(...)`를 적어야 합니다.

```c
import std::io as io;

@effects(console)
fn bool print_u32(u32 value) {
    return switch (io::write_u32(value)) {
        case .ok(_): { true }
        case .err(_): { false }
    };
}
```

만약 `@effects(console)`이 붙은 함수를 부르면서, 부르는 쪽 함수 선언에 `console`을 빼먹으면 컴파일러가 `E7102` 에러로 알려 줍니다. 어떤 코드가 파일이나 콘솔을 건드리는지 함수 선언부만 봐도 바로 알 수 있게 해 주는 장치입니다.
