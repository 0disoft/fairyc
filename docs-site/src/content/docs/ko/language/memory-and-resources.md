---
title: 메모리와 자원
description: Fairy C의 이동 전용 resource 소유권, 자동 cleanup, 공유/가변 빌림(ref / ref mut), 반환 출처 규칙을 설명합니다.
---

Fairy C는 가비지 컬렉터 대신 **단일 소유권 기반 `resource` 타입**과 **컴파일 타임 빌림·수명 검사**로 메모리를 관리합니다.

## 복사 가능한 값 vs. 이동 전용 `resource`

정수나 평범한 `struct`는 값으로 복사됩니다. 반면 `resource`로 선언한 타입은 단일 소유자만 가질 수 있는 이동 전용(affine) 값이며, 스코프를 벗어날 때 아직 살아 있으면 `cleanup` 블록이 자동으로 실행됩니다.

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

컴파일러가 검사하는 핵심 소유권 규칙은 다음과 같습니다.

- **전달 시 소유권 이동**: `resource`를 값으로 넘기거나 대입하면 `cleanup` 책임도 함께 이동하며, 이동 후 재사용은 거부됩니다(`E1101`).
- **필드 부분 이동 및 일반 구조체 포함 금지**: `resource` 내부 필드를 따로 꺼내 이동하거나(`E1103`), 일반 `struct` 안에 `resource`를 넣을 수 없습니다(`E1002`).
- **분기 합류 상태**: `if`/`else` 중 한쪽에서만 자원을 소비해 상태가 `MaybeLive`로 남는 경우를 검사합니다(`E1104`).
- **역순 해제**: 블록 종료 시 살아 있는 자원은 선언의 역순으로 정리되며, `Option`/`Result` 내부 자원은 활성 태그에 따라 조건부 정리됩니다.

## 공유 빌림(`ref T`)과 가변 배타 빌림(`ref mut T`)

소유권을 넘기지 않고 일시적으로 접근하려면 참조를 사용합니다.

- `ref T`: 읽기 전용 공유 참조입니다. 루트 값이 수정되거나 이동하지 않는 동안 여러 개가 공존할 수 있습니다.
- `ref mut T`: 독점 가변 참조입니다. 활성화되어 있는 동안에는 다른 읽기·쓰기·이동을 허용하지 않습니다.
- **루트 전체 잠금 규칙**: 배열의 일부 슬라이스(`view[lo..hi]`)나 필드를 빌려도 소유 루트 단위로 충돌을 검사해 가변 에일리어싱을 막습니다(`E2101`).

```c
fn i32 read(ref i32 value) { return *value; }
fn void edit(ref mut i32 value) { *value = *value + 1; }
fn u8 first(ref u8[] view) { return view[0]; }
fn void fill(ref mut u8[] view) { view[0] = 9; }

fn i32 example() {
    var i32 number = 41;
    let ref mut i32 view = ref mut number;
    edit(view);
    var u8[2] data = [3, 4];
    let ref mut u8[] buffer = ref mut data;
    fill(buffer);
    return *view;
}
```

## 반환 출처 규칙과 `Region`

함수가 `ref T`나 `ref u8[]` 같은 빌림을 반환할 때 Stage 0은 **단일 출처(single-origin)** 규칙을 적용합니다.

- 반환되는 참조는 정적 데이터이거나 **단 하나의 매개변수**에서 유래해야 합니다(둘 이상의 매개변수 후보가 섞이면 `E2103`).
- 지역 변수를 가리키는 참조는 반환할 수 없습니다(`E2102`).
- `Region` 영역에서 할당한 값은 영역 밖으로 탈출할 수 없으며(`E3101`), `cleanup`이 필요한 타입은 일반 `Region`에 넣을 수 없습니다(`E3102`).

## 자주 만나는 오류와 해결

- **`E1101` (이동 후 사용)**: 자원이 이미 값으로 소비되었습니다. 호출자가 계속 써야 한다면 `ref` 또는 `ref mut` 매개변수를 사용하세요.
- **`E2101` (대여 충돌)**: 뒤에서 계속 쓰는 `ref` 뷰가 살아 있는 상태에서 원본을 수정하거나 이동하려 할 때 발생합니다.
- **`E2103` (모호한 반환 출처)**: 두 개의 입력 참조 중 하나를 조건부로 반환하려고 할 때 발생합니다. 소유 값을 반환하거나 단일 입력에서 파생되도록 분리하세요.
