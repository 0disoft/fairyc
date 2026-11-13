---
title: 메모리와 자원
description: resource와 cleanup, 소유권 이동(Move), ref / ref mut 빌림 규칙을 직관적인 예제로 이해합니다.
---

시스템 언어를 처음 배울 때 가장 낯선 부분이 바로 소유권과 빌림입니다. 어렵게 생각하지 말고, **"물건의 주인은 한 명뿐이고, 잠깐 빌려줄 때도 규칙이 있다"**고 생각하면 훨씬 쉬워집니다.

## 1. 일반 값(`struct`) vs. 자원(`resource`)

정수(`i32`)나 숫자만 들어 있는 평범한 `struct`는 다른 변수에 대입하거나 함수에 넘기면 그냥 복사본이 생깁니다.

하지만 **파일 핸들이나 힙 메모리처럼 다 쓴 뒤 꼭 해제(`close`, `free`)해야 하는 값**은 복사되면 곤란합니다. 복사본 두 개가 똑같은 핸들을 각자 해제하려고 하면 이중 해제 버그가 나기 때문이죠.

그래서 Fairy C는 이런 타입을 `resource`로 선언하고 `cleanup { ... }` 블록을 붙입니다.

```c
@repr(C)
struct raw_text_result {
    u32 status;
    u32 allocation_status;
}

@link_name("fy_std_string_drop_storage_v0")
unsafe extern C fn raw_text_result string_drop_storage_raw(ptr mut void value);

pub resource String {
    ptr mut void data;
    usize capacity;
    usize alignment;
    usize length;

    cleanup {
        let raw_text_result dropped = unsafe {
            string_drop_storage_raw(ref mut this as ptr mut void)
        };
    }
}
```

이렇게 해 두면 `String` 변수가 블록(`{ ... }`)을 빠져나갈 때 컴파일러가 `cleanup`을 자동으로 호출해 줍니다. 개발자가 해제 코드를 깜빡할 여지가 없어집니다.

## 2. 소유권 이동(Move) — 한 번 넘기면 끝

`resource` 변수를 다른 변수에 넣거나 함수에 값 그대로 넘기면, **소유권(정리할 책임)도 같이 넘어갑니다**. 내가 가진 열쇠를 친구에게 아예 줘 버린 것과 같습니다.

- 열쇠를 넘긴 뒤 내가 또 그 변수를 쓰려고 하면 컴파일러가 `E1101` 에러로 막아 줍니다.
- `if` 문 안쪽에서만 자원을 넘겨 버리고 `else`에서는 안 넘겼다면, 블록 뒤에서 그 자원이 아직 살아 있는지 애매해지므로 컴파일러가 알려 줍니다(`E1104`).
- 일반 `struct` 안에는 `resource`를 필드로 넣을 수 없고(`E1002`), `resource`에서 필드 하나만 빼내 이동할 수도 없습니다(`E1103`).

## 3. 빌려주기: `ref T`와 `ref mut T`

함수를 부를 때마다 자원을 아예 넘겨 버리면 호출한 쪽에서 다시 못 쓰니 불편하겠죠? 이럴 때 소유권은 내가 쥔 채 **"잠깐 읽기만 해"(공유 빌림 `ref T`)** 또는 **"잠깐 고쳐도 돼"(배타 가변 빌림 `ref mut T`)**라고 빌려줍니다.

```c
fn i32 read(ref i32 value) { return *value; }
fn void edit(ref mut i32 value) { *value = *value + 1; }
fn u8 first(ref u8[] view) { return view[0]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }

fn i32 main() {
    var i32 number = 41;
    let ref mut i32 view = ref mut number;
    edit(view);
    read(view);

    var u8[2] data = [3, 4];
    let ref mut u8[] buffer = ref mut data;
    fill(buffer);
    return *view;
}
```

빌림 규칙은 도서관 책 규칙과 똑같습니다.

1. **읽기 전용(`ref`)은 여러 명이 동시에 빌릴 수 있습니다.** 대신 아무도 그 책을 고치거나 가져가면 안 됩니다.
2. **수정용(`ref mut`)은 한 번에 한 명만 빌릴 수 있습니다.** 수정하는 동안 다른 사람이 동시에 읽거나 고치면 값이 꼬이기 때문입니다(`E2101`).
3. **배열 일부(`view[lo..hi]`)를 빌려도 주인 전체 기준으로 검사합니다.** 앞부분을 빌려 수정하는 동안 뒷부분이라고 해서 또 다른 가변 참조를 동시에 겹쳐 잡을 수는 없습니다.

## 4. 함수에서 참조를 반환할 때 규칙

함수 안에서 만든 지역 변수는 함수가 `return`하는 순간 스택에서 사라집니다. 그러니 지역 변수를 가리키는 `ref`를 반환할 수 없습니다(`E2102`).

또한 `ref`나 `ref u8[]`를 돌려줄 때는 **매개변수로 받은 참조 중 딱 한 군데 출처에서 온 것만** 돌려줄 수 있습니다(`if (flag) { return a; } else { return b; }`처럼 두 입력 중 하나를 섞어 반환하면 `E2103`). 이런 경우에는 참조 대신 새 값을 만들어 돌려주는 편이 훨씬 깔끔합니다.
