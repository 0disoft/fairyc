---
title: 구조체, enum, switch
description: Fairy C의 struct와 필드 단축 초기화, 메서드(Type::method), enum, switch 패턴 매칭, Option/Result/? 전파를 익힙니다.
---

조금만 프로그램이 커져도 여러 값을 하나로 묶고, 성공/실패나 여러 상태 중 하나를 깔끔하게 표현하고 싶어집니다. Fairy C는 `struct`, `enum`, `switch`, 그리고 `Option`·`Result` 조합으로 이를 명확하게 다룹니다.

## 1. 구조체(`struct`)와 필드 단축 문법

관련된 필드를 묶어 새 타입을 만들 때는 `struct`를 씁니다. 다른 모듈에서도 필드에 접근할 수 있게 하려면 필드 앞에 `pub`를 붙입니다.

```c
pub struct Point {
    pub i32 x;
    pub i32 y;
}

fn i32 demo() {
    let i32 x = 10;
    let i32 y = 32;

    // .x = x 와 .y = y 를 줄여서 .y 로 쓸 수 있습니다.
    let Point p = Point { .x = x, .y };
    return p.x + p.y;
}
```

JavaScript나 Rust의 shorthand처럼 지역 변수 이름과 필드 이름이 같다면 `.y`처럼 값 대입을 생략할 수 있습니다. (단, 모든 필드를 선언 순서대로 빠짐없이 채워야 합니다.)

### 타입 메서드(`Type::method`)

구조체(또는 `resource`)에 딸린 함수나 메서드는 함수 이름 앞에 `타입명::`을 붙여서 선언합니다. 첫 번째 매개변수 이름이 `this`면 `.method()` 문법으로 호출할 수 있습니다.

```c
struct Counter {
    i32 value;
}

fn i32 Counter::read(ref Counter this) {
    return this.value;
}

fn void Counter::add(ref mut Counter this, i32 delta) {
    this.value += delta;
}

fn i32 main() {
    var Counter c = Counter { .value = 75 };
    c.add(2);
    return c.read();
}
```

## 2. 열거형(`enum`)과 C 호환 enum

여러 상태 중 하나를 표현하고 각 상태에 추가 데이터까지 담고 싶다면 `enum`을 씁니다.

```c
enum ParseError {
    Empty;
    InvalidByte { u8 offset; u8 byte; };
}
```

값을 만들 때는 기대하는 타입이 정해져 있다면 앞의 `ParseError::`를 생략하고 `.Empty`, `.InvalidByte { .offset = 2, .byte = 255 }`처럼 점(`.`)으로 시작해 간단히 생성합니다.

C 구조체 안에 들어가야 하는 단순 정수 열거형이라면 `@repr(c::uint)`처럼 C 정수 기반 표현을 지정할 수도 있습니다.

```c
@repr(c::uint)
enum Status {
    Ready;
    Busy;
}
```

## 3. `switch` 패턴 매칭

`switch`는 모든 경우(variant)를 빠뜨리지 않고 처리했는지 컴파일러가 검사합니다. C처럼 `break`를 잊어서 다음 케이스로 넘어가는 실수(fallthrough)는 아예 발생하지 않습니다.

```c
fn i32 code_of(ParseError err) {
    return switch (err) {
        case .Empty: { 1 }
        case .InvalidByte { .offset = pos, .byte = _ }: {
            (pos as i32) + 10
        }
    };
}
```

- 각 `case`는 반드시 `{ ... }` 블록을 가집니다.
- 필요 없는 필드는 `_`로 무시할 수 있고, 마지막 줄에 세미콜론 없이 식을 남기면 `switch` 전체가 값을 반환합니다.
- 모든 `case`를 빠짐없이 적어야 하므로, 나중에 `enum`에 새 상태를 추가했을 때 처리 누락을 컴파일러가 바로 알려 줍니다.

## 4. `Option<T>`, `Result<T, E>`, 그리고 `?`

값이 있을 수도 없을 수도 있을 때는 `Option<T>`, 작업이 성공하거나 오류를 돌려줄 때는 `Result<T, E>`를 씁니다.

- **`Option<T>`**: `.some(값)` 또는 `.none`
- **`Result<T, E>`**: `.ok(값)` 또는 `.err(오류)` (`result.is_ok`로 `bool` 성공 여부를 바로 확인할 수도 있습니다)

함수가 실패하면 바로 바깥으로 오류를 전달하고 싶을 때는 `switch`를 길게 쓰는 대신 식 뒤에 `?`를 붙이면 됩니다.

```c
fn Result<i32, ParseError> add_one(Result<i32, ParseError> input) {
    let i32 n = input?;    // .err(...) 면 여기서 즉시 반환!
    return .ok(n + 1);
}
```

### `@must_use`와 `discard`

`Result`를 반환하는 함수를 부르고 결과를 아무 데도 쓰지 않으면 컴파일러가 경고가 아니라 에러로 막습니다. 정말로 결과를 버려도 괜찮은 상황이라면 `discard` 키워드로 "내가 알고 버리는 것"임을 명시해야 합니다.

```c
discard add_one(.err(.Empty));
```

일반 함수라도 반환값을 꼭 확인하게 만들고 싶다면 `@must_use` 속성을 붙일 수 있습니다.

```c
@must_use
fn usize bytes_written() {
    return 16;
}
```

---

다음 장에서는 고정 크기 배열과 안전하게 구간을 가리키는 슬라이스 뷰, 그리고 문자열 뷰의 차이를 알아봅니다.
- 이전: [함수와 제어문](../control-flow-and-functions/)
- 다음: [배열, 슬라이스, 문자열 뷰](../arrays-and-slices/)
