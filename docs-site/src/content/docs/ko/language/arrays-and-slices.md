---
title: 4. 배열, 슬라이스, 문자열 뷰
description: Fairy C의 고정 배열 T[N], 슬라이스 뷰 ref T[] / ref mut T[], 범위 슬라이싱, 구조체 배열 필드, str/u8[]/CStr 차이를 배웁니다.
---

시스템 프로그래밍에서는 데이터를 연속된 메모리 블록으로 다루는 일이 많습니다. Fairy C는 길이가 컴파일 시점에 정해지는 **고정 배열**과, 메모리를 복사하지 않고 일부 구간만 안전하게 빌려 보는 **슬라이스 뷰**를 제공합니다.

## 1. 고정 크기 배열 (`T[N]`)

배열 타입은 `요소타입[개수]` 형태로 씁니다. (현재 Stage 0에서 고정 배열 길이는 `1`~`4096` 범위를 지원합니다.)

```c
let i32[3] scores = [10, 20, 30];
var u8[16] buffer = [0; 16];       // 0으로 16칸 채우기

let usize len = scores.len;        // 3 (usize)
let i32 first = scores[0];
buffer[1] = 65;                    // var 배열이므로 수정 가능
buffer[1] += 1;                    // 66
```

- **인덱스는 `usize`입니다**: `scores[i]`의 인덱스 타입은 `usize`입니다. 리터럴 `0`, `1`은 문맥상 자동으로 `usize`로 처리됩니다.
- **경계 밖 접근은 실행 중단(trap)**: `scores[5]`처럼 배열 길이 이상을 읽거나 쓰려고 하면, 다른 메모리를 망가뜨리는 대신 즉시 `bounds` 트랩으로 안전하게 멈춥니다.

구조체 안에 고정 길이 스칼라 배열(또는 단순 구조체 배열)을 필드로 넣는 것도 지원됩니다(`@repr(C)` 제외).

```c
struct Packet {
    u8[8] bytes;
    usize length;
}
```

## 2. 슬라이스 뷰(`ref T[]`, `ref mut T[]`)

함수에 배열을 넘길 때마다 통째로 복사하면 무겁고 비효율적입니다. 이럴 때는 배열의 시작 위치와 길이 정보만 빌려 오는 **슬라이스 뷰**를 씁니다.

```c
fn u8 first_byte(ref u8[] view) {
    if (view.is_empty) { return 0; }
    return view[0];
}

fn void clear_head(ref mut u8[] view) {
    if (view.len > 0) {
        view[0] = 0;
    }
}

fn u8 demo() {
    var u8[4] data = [70, 97, 105, 114];
    clear_head(ref mut data);
    return first_byte(ref data);
}
```

- **`ref T[]`**: 읽기 전용 뷰입니다. `.len`, `.is_empty`, `view[index]` 읽기가 됩니다.
- **`ref mut T[]`**: 값을 수정할 수 있는 가변 뷰입니다. 빌림 변수 자체가 `let`이어도 참조 대상이 가변(`var`)에서 왔으므로 내용을 바꿀 수 있습니다.

### 구간 잘라 보기(`arr[start..end]`)

`ref u8[]` 뷰나 이름 붙은 고정 배열에서 일부분만 잘라 볼 수 있습니다. 끝 인덱스(`end`)는 포함되지 않는 반열린 구간(`start <= index < end`)입니다.

```c
let ref u8[] full = b"Fairy C";
let ref u8[] head = full[0..5];   // b"Fairy"
let ref u8[] tail = full[6..];    // b"C"
let ref u8[] copy_all = full[..]; // 전체 범위
```

고정 배열 `points`에서 구간을 직접 빌릴 때는 `ref points[1..3]` 또는 `ref mut points[1..3]`처럼 적습니다. 범위가 역전되거나 길이를 넘으면 역시 `bounds` 트랩이 발생합니다.

## 3. 문자열과 바이트 리터럴 한눈에 구분하기

Fairy C에는 용도가 다른 세 가지 문자열·바이트 리터럴이 있습니다. C와의 연동을 염두에 두고 처음부터 역할을 갈라놓았습니다.

| 리터럴 | 타입 | 특징과 용도 |
| --- | --- | --- |
| `"Fairy"` | `ref str` | UTF-8 검증된 문자열 뷰. `.len`과 `.is_empty` 지원 (바이트 길이) |
| `b"Fairy\xff"` | `ref u8[]` | 임의의 바이트열 뷰. 인덱싱(`b[0]`), 슬라이싱(`b[1..3]`) 가능 |
| `c"Fairy"` | `ref CStr` | 끝에 `\0`(NUL) 바이트가 보장되어 C API에 바로 넘기기 좋음 |

주의할 점은 `ref str`은 UTF-8 바이트 시퀀스로서의 정합성을 지키기 위해 `s[0]` 같은 임의 바이트 인덱싱을 허용하지 않는다는 것입니다. 바이트 단위로 쪼개 보고 싶다면 `std::text::as_bytes(s)`로 `ref u8[]` 뷰를 얻은 뒤 다루면 됩니다.

---

다음 장에서는 Fairy C의 가장 핵심 개념인 소유권 이동, `ref`/`ref mut` 빌림 규칙, 자동 소멸 `resource`, `region`, 그리고 `@abilities`/`@effects`/`unsafe`까지 한 번에 정리합니다.
- 이전: [3. 구조체, enum, switch](../structs-enums-and-errors/)
- 다음: [5. 소유권, 빌림, resource와 region](../memory-and-resources/)
