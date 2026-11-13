---
title: 함수와 제어문
description: Fairy C의 함수 선언, 이름 있는 인자, 제네릭 함수, if 표현식, C 스타일 for와 for-in 순회, defer 문법을 배웁니다.
---

변수를 배웠으니 이제 흐름을 제어하는 함수, 조건문, 반복문을 살펴볼 차례입니다. 전체적으로는 C와 닮았지만, `if`가 값을 만들어 내고 `for-in` 순회와 `defer` 정리를 기본 제공하는 등 현대적인 편의 기능이 들어가 있습니다.

## 1. 함수 선언과 이름 있는 인자

함수는 `fn 반환타입 이름(매개변수)` 형태로 씁니다. 돌려줄 값이 없다면 반환 타입 자리에 `void`를 적습니다.

```c
fn i32 clamp(i32 value, i32 minimum, i32 maximum) {
    if (value < minimum) { return minimum; }
    if (value > maximum) { return maximum; }
    return value;
}

fn void log_step() {
    // void 함수는 return; 없이 끝나도 괜찮습니다.
}
```

C나 Rust와 달리 **함수 본문 마지막 줄에서 세미콜론을 빼고 자동 반환하는 문법은 없습니다**. 값을 돌려주려면 꼭 `return 값;`을 써야 합니다.

### 이름 있는 인자(Argument labels)

같은 모듈 안의 일반 함수를 부를 때는 인자가 헷갈리지 않게 매개변수 이름을 `.이름 = 값` 형태로 함께 적을 수 있습니다.

```c
let i32 result = clamp(77, .minimum = 0, .maximum = 100);
```

단, 순서를 뒤바꾸거나 일부 인자를 생략하는 기본값 기능은 없으며, 해당 위치의 매개변수 이름과 정확히 일치할 때 가독성 보조용으로 쓸 수 있습니다.

## 2. 제네릭 함수 (`<T>`)

함수 이름 뒤에 `<T>`를 붙이고 호출할 때 `<i32>`처럼 구체적인 타입을 넘기는 스칼라 제네릭 함수도 지원합니다.

```c
fn T pick_first<T>(T first, T second) {
    return first;
}

fn i32 main() {
    return pick_first<i32>(77, 0);
}
```

지금 단계에서는 `bool`, `u8`, `i32`, `u32`, `usize` 같은 기본 스칼라 타입을 제네릭 인자로 넣어서 컴파일 타임에 특수화(specialization)하는 용도로 사용할 수 있습니다. (사용자 정의 `struct<T>`나 타입 추론 생략은 아직 지원하지 않습니다.)

## 3. 표현식으로 쓰는 `if / else`

조건문은 괄호 `(...)`가 필수입니다. 그리고 블록 마지막에 세미콜론 없는 식을 남기면 `if` 자체가 하나의 값(표현식)이 됩니다.

```c
let i32 score = 85;
let i32 grade = if (score >= 90) {
    1
} else if (score >= 80) {
    2
} else {
    3
};
```

두 갈래에서 만들어 내는 타입은 서로 같아야 하며, 변수 초기화나 `return if (...) { ... } else { ... };`처럼 바로 쓸 수 있어 편리합니다.

## 4. `while`과 두 가지 `for` 루프

### `while`과 C 스타일 `for`

```c
var i32 n = 0;
while (n < 5) {
    n += 1;
}

var i32 sum = 0;
for (var i32 i = 0; i < 10; i += 1) {
    if (i == 3) { continue; }
    if (i == 8) { break; }
    sum += i;
}
```

C 스타일 `for (초기화; 조건; 증감)`에서 각 부분은 비워 둘 수 있고, 조건을 비우면 `true`로 취급됩니다. `continue`를 만나면 `증감` 절을 실행한 뒤 다음 조건으로 넘어갑니다.

### 배열·슬라이스를 도는 `for-in`

배열이나 슬라이스 뷰를 인덱스 실수 없이 처음부터 끝까지 훑고 싶다면 `for-in`을 씁니다.

```c
var i32[4] items = [10, 20, 30, 40];

// 값을 하나씩 복사해서 읽기
var i32 total = 0;
for (let i32 item in items) {
    total += item;
}

// 원본 배열을 직접 빌려 수정하기
for (let ref mut i32 item in items) {
    *item += 1;
}
```

원하는 구간만 슬라이스로 잘라 `for (let u8 b in data[1..4]) { ... }`처럼 순회하는 것도 가능합니다.

## 5. `assert`, `defer`, `source_location()`

- **`assert(조건);`**: 조건이 거짓이면 실행을 즉시 중단합니다. 테스트나 빠른 검증에 자주 씁니다.
- **`defer 함수호출();`**: 현재 블록이 끝나기 직전(정상 종료든 `return`이든)에 해당 `void` 함수 호출이 실행됩니다. 등록 시점에 스칼라 인자 값을 미리 평가해 두며, 블록의 자원 `cleanup`보다 먼저 실행됩니다.
- **`source_location()`**: `"src/main.fy:12:5@240"` 같은 현재 코드 위치를 `ref str` 문자열로 돌려주는 내장 함수입니다. 로깅이나 디버깅 도우미를 만들 때 유용합니다.

---

다음 장에서는 데이터를 묶는 `struct`, 상태를 구분하는 `enum`, 그리고 패턴 매칭 `switch`와 `Option`/`Result` 에러 처리를 배웁니다.
- 이전: [변수, 타입, 연산자](../basics/)
- 다음: [구조체, enum, switch](../structs-enums-and-errors/)
