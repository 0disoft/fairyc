---
title: C 연동과 Win32 데스크톱 앱 입문
description: "@repr(C), unsafe extern C fn, C 콜백 문법과 실제 Win32 예제(win32_minimal, win32_hash) 구성을 살펴봅니다."
---

기존 C 라이브러리나 Windows Win32 API를 그대로 불러 쓰고 싶을 때 Fairy C가 제공하는 연동 문법과, 저장소에 들어 있는 두 가지 Win32 예제를 살펴보겠습니다.

## 1. C 연동의 핵심 문법 3가지

### `@repr(C)` — 구조체 메모리 배치 맞추기
Fairy C 컴파일러는 일반 구조체의 필드 배치를 임의로 바꿀 수 있습니다. C 함수와 구조체를 주고받으려면 `@repr(C)`를 붙여서 C와 똑같은 순서·정렬로 고정해야 합니다(`examples/win32_hash/src/writer.fy`).

```c
@repr(C)
pub struct Observation {
    u32 phase;
    u32 error;
    u32 exists;
}
```

### `unsafe extern C fn`과 `unsafe { ... }`
C 쪽에 있는 함수를 선언할 때는 `unsafe extern C fn`을 붙이고, 부를 때는 `unsafe { ... }` 블록으로 감쌉니다. "여기서부터는 C 세계이므로 컴파일러의 안전 검사가 닿지 않는다"는 경계를 표시하는 문법입니다.

```c
@effects(filesystem)
unsafe extern C fn void hash_writer_drop(ptr mut void context);

@effects(filesystem)
pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe { hash_writer_drop(this.handle); } }
}
```

이렇게 `resource`의 `cleanup` 안에 `unsafe` 해제를 감싸 두면, 바깥쪽 Fairy C 코드는 `unsafe` 없이 안전하게 자원을 쓸 수 있습니다.

C 쪽 심볼 이름이 Fairy C 함수 이름과 다르다면 `@link_name("c_symbol")`로 실제 링크 심볼을 지정할 수 있습니다.

```c
@link_name("strlen")
unsafe extern C fn usize c_strlen(ptr u8 text);
```

### C 콜백 넘기기 (`examples/c_interop_callback`)
C 함수에 Fairy C 함수를 콜백으로 넘기는 것도 가능합니다. 다만 안전 참조(`ref T`)를 `extern C` 시그니처에 그대로 넣을 수는 없고(`E5102`), 원시 포인터(`ptr mut void`)를 통해 주고받습니다.

```c
unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    return if (value >= 10) { 1 } else { 0 };
}
```

## 2. Win32 앱 연결하기 (`fairy.windows.json`)

Windows 창 앱을 만들 때는 패키지 옆에 `fairy.windows.json` 파일을 두어 필요한 C 파일, 리소스(`.rc`), Windows 시스템 라이브러리(`user32.lib`, `gdi32.lib` 등)를 지정합니다.

```json
{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}
```

## 3. 실제 예제 1: `examples/win32_minimal`

작은 Win32 GUI 예제입니다. 역할을 딱 절반씩 나누어 구현되어 있습니다.

- **C 브리지 (`native/window.c`, `native/ime.c`)**: Windows 창 생성, 메시지 루프, 한글 IME 처리, DPI API 담당
- **Fairy C (`src/main.fy`)**: DPI 배율에 맞춘 컨트롤 위치 계산(`on_size`, `on_dpi`), 키보드·Tab 이동(`on_key`, `on_tab`), 버튼 클릭 시 입력 텍스트 처리(`on_click`) 담당

```powershell
cd examples/win32_minimal
fairyc build --release
.\build\win32-minimal.exe
```

참고로 이 예제는 테스트용 파이프 진단 코드를 빼고 배포용 래퍼로 빌드해 본 기록(`windows-gui-slim-20261111`)에서 Windows x64 MSVC 릴리스 기준 Go CLI 빌드 `121,344`바이트, Authored CLI 빌드 `123,904`바이트(`121 KiB`)로 측정되었습니다(정적 CRT 포함, `USER32`/`GDI32`/`IMM32`/`KERNEL32` 사용).

## 4. 실제 예제 2: `examples/win32_hash` (Fairy Hash)

파일을 골라 SHA-256 해시를 계산하고 JSON 리포트를 저장하는 데스크톱 + CLI 앱입니다. `std::fs`, `std::bytes`, `std::process`, `std::io` 표준 모듈과 `native/*.c` Win32 브리지를 함께 사용하며, `main()`에는 `@effects(console, filesystem, clock)`이 붙어 있습니다.

```powershell
cd examples/win32_hash
fairyc build --release
.\build\win32-hash.exe --hash path\to\file
```

아이콘(`.ico`), 버전 정보(`VERSIONINFO`), 공용 컨트롤 매니페스트까지 포함한 `2026-11-13` Windows x64 릴리스(`--locked --release`) 포터블 빌드 기준 `FairyHash.exe` 크기는 `196,096`바이트(`191.5 KiB`)로 측정되었습니다(리소스 추가 전 빌드는 `188,416`바이트).

> 위 `121 KiB`, `191.5 KiB` 크기는 두 예제를 해당 Windows x64 MSVC 릴리스 조건과 C 브리지 구성으로 빌드했을 때의 구체적인 사례 수치일 뿐, 모든 Fairy C 프로그램이 항상 이 크기로 나온다는 뜻은 아닙니다.
