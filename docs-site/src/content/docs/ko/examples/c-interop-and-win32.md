---
title: C 연동과 Win32 데스크톱 앱 입문
description: "@repr(C), unsafe extern C fn, 콜백, 그리고 실제 Win32 예제(win32_minimal, win32_hash)와 측정 기록을 소개합니다."
---

Fairy C는 C 호환 구조체 레이아웃, 명시적 `unsafe extern C fn` 선언, 그리고 Windows용 C 브리지 설정을 통해 C 라이브러리 및 Win32 API와 직접 연동합니다.

## C ABI 연동의 핵심 요소

1. **C 호환 구조체 레이아웃 (`@repr(C)`)**: 필드 순서와 정렬을 타깃 C ABI에 맞춥니다(`examples/win32_hash/src/writer.fy`).

```c
@repr(C)
pub struct Observation {
    u32 phase;
    u32 error;
    u32 exists;
}
```

2. **외부 함수 선언과 `unsafe` 블록**: 원시 포인터와 외부 C 함수는 `unsafe extern C fn`(필요 시 `@link_name(...)` 병기)으로 선언하고 `unsafe { ... }` 안에서 호출합니다(`E5101`).

```c
@effects(filesystem)
unsafe extern C fn void hash_writer_drop(ptr mut void context);

@effects(filesystem)
pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe { hash_writer_drop(this.handle); } }
}
```

3. **동기 C 콜백 (`examples/c_interop_callback`)**: 동기 C 콜백 전달은 가능하지만, `extern C` 시그니처에 안전 참조(`ref T`)를 직접 넣거나(`E5102`), C 예외/`longjmp`가 Fairy 프레임을 거슬러 올라오게 해서는 안 됩니다(`E5103`).

```c
unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    return if (value >= 10) { 1 } else { 0 };
}
```

## Win32 빌드 설정 (`fairy.windows.json`)

Win32 GUI 예제에서는 `fairy.windows.json` (`"schema": "fairyc.windows-build/v1"`)을 통해 Windows 서브시스템, 시스템 라이브러리, C 브리지 소스, `.rc` 리소스를 함께 링크합니다.

```json
{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}
```

## 실제 예제 1: `examples/win32_minimal`

`examples/win32_minimal`은 역할을 다음처럼 나눕니다.

- **네이티브 C 브리지 (`native/window.c`, `native/ime.c`)**: `HWND` 생성, Win32 메시지 루프, `IMM32` 한글 조합(IME) 처리, 폰트·DPI API 담당.
- **Fairy 애플리케이션 (`src/main.fy`)**: DPI 배율 기반 컨트롤 배치(`on_size`, `on_dpi`), 키보드·Tab 탐색(`on_key`, `on_tab`), 고정 바이트 버퍼를 이용한 입력/출력 처리(`on_click`) 담당.

Windows 환경에서 빌드 및 실행:

```powershell
cd examples/win32_minimal
fairyc build --release
.\build\win32-minimal.exe
```

### 실측 EXE 크기 기록 (`windows-gui-slim-20261111`)

테스트용 JSONL 파이프·스모크 진단을 제외한 배포용 래퍼(`build_distribution.py`)를 적용했을 때 Windows x64 MSVC 릴리스 기준 크기 변화는 다음과 같습니다.

| 컴파일러 | 프로파일 | 일반 빌드 | 배포 빌드 |
| --- | --- | ---: | ---: |
| Go CLI | release | 141,824 B | 121,344 B |
| Authored CLI | release | 144,384 B (141 KiB) | 123,904 B (121 KiB) |

정적 CRT가 포함되며 `USER32`, `GDI32`, `IMM32`, `KERNEL32` 시스템 DLL을 임포트합니다.

## 실제 예제 2: `examples/win32_hash` (Fairy Hash)

`examples/win32_hash` (Fairy Hash)는 데스크톱 GUI와 `--hash <파일> [저장경로]` CLI를 모두 지원하는 SHA-256 파일 무결성 유틸리티입니다.

- `std::fs`, `std::bytes`, `std::process`, `std::io` 표준 모듈과 얇은 Win32 C 브리지(`native/window.c`, `native/reader.c`, `native/writer.c`)를 함께 사용합니다.
- `src/main.fy`의 `main()`은 `@effects(console, filesystem, clock)`을 선언하며, `src/writer.fy`의 `resource Writer`로 비동기 저장 핸들을 안전하게 해제합니다.

CLI 해시 모드 실행 예시:

```powershell
cd examples/win32_hash
fairyc build --release
.\build\win32-hash.exe --hash path\to\file
```

### 실측 EXE 크기 기록 (`windows-hash-package-20261113`)

`2026-11-13` 포터블 패키지 검증(`fairyc-go.exe build --locked --release`, 9종 해상도 `.ico`, `VERSIONINFO`, Common Controls v6 매니페스트 내장, 테스트/스크립트 제외) 기준 측정값은 다음과 같습니다.

- `FairyHash.exe` (아이콘·버전 리소스 내장): **196,096바이트 (191.5 KiB)** (리소스 추가 전 빌드는 **188,416바이트**)
- 무서명 포터블 ZIP (`fairy-hash-0.1.0-windows-x64.zip`): **105,640바이트 (103.16 KiB)**

> **측정 범위 안내:** 위 `121 KiB`, `191.5 KiB` 수치는 각각 `win32_minimal`과 `win32_hash`의 특정 Windows x64 MSVC 릴리스 빌드 조건과 C 브리지·정적 CRT 구성을 전제로 측정된 결과입니다. 모든 Fairy C 프로그램에 대해 동일한 바이너리 크기나 일반적 성능 우위를 보장하는 수치로 해석해서는 안 됩니다.
