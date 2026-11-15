---
title: 소유권, 빌림, resource와 region
description: Fairy C의 Move와 Copy, ref/ref mut 빌림, resource cleanup, region, @abilities/burn, @effects, unsafe/raw pointer를 이해합니다.
---

Fairy C를 처음 볼 때 가장 낯설지만, 가장 큰 강점이기도 한 부분입니다. C처럼 메모리를 직접 다루면서도 "해제된 포인터 다시 쓰기"나 "정리 코드 깜빡하기"를 컴파일러가 막아 주는 장치들이 여기에 모여 있습니다.

## 1. 복사(Copy)와 소유권 이동(Move)

정수(`i32`), `bool`, 작은 일반 `struct` 같은 값들은 다른 변수나 함수로 넘길 때 값이 그대로 복사(Copy)됩니다. 원래 변수를 계속 써도 문제없습니다.

반대로 파일 핸들이나 동적 배열처럼 `resource`로 정의된 값은 단 하나의 변수만 주인(소유자)이 될 수 있습니다. 함수에 값으로 넘기면 소유권이 함께 이동(Move)하며, 넘긴 뒤 원래 변수를 다시 쓰려고 하면 컴파일러가 에러로 막습니다.

## 2. 빌리기: `ref`와 `ref mut`

주인이 가진 값을 넘겨주지 않고 잠시만 보거나 고치고 싶다면 **참조**(빌림)를 씁니다.

- **`ref T` (읽기 전용 빌림)**: 여러 군데에 동시에 빌려 줄 수 있습니다.
- **`ref mut T` (가변 빌림)**: 한 번에 딱 한 곳에만 빌려 줄 수 있습니다.

```c
fn i32 read_num(ref i32 n) {
    return *n;
}

fn void add_ten(ref mut i32 n) {
    *n += 10;
}

fn i32 main() {
    var i32 score = 67;
    add_ten(ref mut score);
    return read_num(ref score);   // 77
}
```

참조를 통해 실제 값을 읽거나 고칠 때는 포인터처럼 `*n`을 씁니다. 함수가 빌려 받은 참조를 반환할 때는 "원래 주인보다 오래 살아남을 수 없다"는 수명(origin) 규칙이 적용되므로, 함수 내부 지역 변수의 참조를 바깥으로 return할 수는 없습니다.

## 3. 자동 정리 블록: `resource`와 `cleanup`

C에서는 `malloc`/`free`, `CreateFile`/`CloseHandle`을 짝지어야 해서, 중간에 `if (...) return;`으로 빠져나가면 정리를 놓치기 쉽습니다. Fairy C에서는 `resource` 안에 `cleanup { ... }` 블록을 한 번만 적어 두면 됩니다.

```c
resource Buffer {
    ptr mut u8 data;

    cleanup {
        // 변수가 스코프를 벗어나거나 early return할 때 자동 실행!
    }
}
```

- 소유자가 다른 곳으로 이동하지 않고 스코프 끝에 도달하면 컴파일러가 `cleanup`을 자동으로 끼워 넣어 줍니다.
- `resource`의 내부 필드는 해당 모듈 밖에서는 비공개(private)로 보호되므로, 외부 코드가 핸들을 함부로 빼내 망가뜨릴 수 없습니다.
- 정리 코드 자체가 파일 닫기나 잠금 해제 같은 효과를 일으킨다면 `@effects(...) cleanup { ... }`처럼 `cleanup`에도 효과를 붙일 수 있으며, 이 `resource`를 사용하는 호출자도 해당 효과를 함께 선언해야 합니다(`E7102`).

## 4. `region`: 한꺼번에 빌리고 한꺼번에 비우기

잠깐 쓰고 버릴 임시 버퍼들을 매번 개별 해제하기 번거로울 때는 `region` 구문을 씁니다.

```c
import std::alloc as alloc;

fn i32 use_scratch(alloc::Arena arena) {
    var alloc::Arena mutable_arena = arena;
    region scratch in ref mut mutable_arena {
        // 이 블록 안에서 scratch에 할당한 임시 데이터는
        // 블록을 빠져나갈 때 한꺼번에 정리됩니다.
    }
    return 0;
}
```

## 5. `@abilities`와 선형 자원 `burn`

기본 `resource`는 스코프를 벗어나면 자동으로 `cleanup`이 실행됩니다. 하지만 "절대로 그냥 사라지면 안 되는 티켓이나 영수증"처럼, 코드 흐름상 깜빡하고 버리는 것조차 막고 싶다면 `@abilities(...)`를 붙입니다.

```c
@abilities(store, key)   // drop이 빠졌으므로 자동 폐기 금지!
resource Receipt {
    u32 amount;
    cleanup {}
}
```

- `drop`이 빠진 `resource`는 반드시 다른 함수로 넘기거나 반환해야 하며, 그냥 블록 끝에 남겨 두면 `E1108` 컴파일 에러가 납니다.
- 해당 자원을 선언한 모듈 안에서 명시적으로 없애야 할 때만 `burn receipt;` 구문을 써서 소멸시킵니다. (`@abilities`에는 `drop`, `store`, `key`를 지정할 수 있습니다.)

## 6. `@effects(...)`: 함수가 바깥세상에 닿는지 표시하기

파일, 콘솔, 네트워크 같은 외부 권한을 건드리는 함수에는 `@effects(...)`를 적습니다.

```c
@effects(console)
fn void greet() {
    // 콘솔 출력 호출 가능
}
```

자주 쓰이는 효과 이름은 `console`, `filesystem`, `environment`, `process`, `network`, `clock`, `random` 등입니다. `@effects(console)`이 붙은 함수를 부르는 상위 함수 역시 `@effects(console)`을 선언해야 합니다(`E7102` 검사). 아무것도 붙이지 않으면 외부 효과 없는 순수한(빈 집합) 함수로 취급됩니다.

## 7. `unsafe`와 원시 포인터(`ptr` / `ptr mut`)

Win32 API나 C 라이브러리를 부를 때는 컴파일러가 메모리 수명을 자동으로 증명할 수 없으므로 `unsafe { ... }` 블록과 원시 포인터(`ptr T`, `ptr mut T`)를 씁니다.

```c
unsafe extern C fn void c_free(ptr mut void p);
```

- 원시 포인터 역참조, `ptr`로의 캐스트, `extern C` 함수 호출은 반드시 `unsafe` 함수 또는 `unsafe { ... }` 안에서만 허용됩니다.
- 주의: `unsafe`는 "메모리 안전 검사를 개발자가 직접 책임진다"는 표시일 뿐, `@effects(...)` 요구사항을 면제해 주지는 않습니다.
- `null` 리터럴은 `ptr T`, `ptr mut T`, `unsafe extern C fn` 타입을 초기화하거나 `p == null`, `p != null`로 비교할 때만 사용할 수 있으며, 안전한 `ref T` / `ref mut T`에는 `null`을 넣을 수 없습니다.

---

다음 장에서는 코드를 파일 여러 개로 나누는 `module`/`import`, `fairy.toml` 프로젝트 설정, 그리고 `std` 표준 라이브러리를 살펴봅니다.
- 이전: [배열, 슬라이스, 문자열 뷰](../arrays-and-slices/)
- 다음: [모듈, 패키지, 표준 라이브러리](../modules-and-projects/)
