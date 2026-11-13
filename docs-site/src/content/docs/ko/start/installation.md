---
title: 설치와 환경 준비
description: 저장소 소스에서 Stage 0 fairyc 컴파일러를 빌드하고 네이티브 C 툴체인을 준비하는 방법입니다.
---

현재 Fairy C는 별도의 전역 설치 관리자나 사전 빌드 다운로드 링크를 제공하지 않습니다. 저장소 소스에서 `fairyc` CLI를 빌드한 뒤 운영체제의 C 컴파일러·SDK와 함께 사용합니다.

## 필요한 도구

1. **Git, Go, Bun**: Stage 0 Go 컴파일러 프로토타입 빌드에 필요합니다(`tools/run-go.ts`가 Bun을 통해 Go 명령을 실행합니다).
2. **네이티브 C 툴체인** (`fairyc build`, `fairyc run`, `fairyc test --native` 실행 시 필요):
   - **Windows (`x86_64-windows-msvc`)**: Visual Studio 또는 Build Tools의 **MSVC x64 C/C++ 컴파일러**와 **Windows SDK**가 필요합니다.
   - **Linux (`x86_64-linux-gnu`)**: `PATH`에서 `gcc` 또는 `clang`을 실행할 수 있어야 합니다.

> **참고:** `fairyc check`와 인터프리터 기반 `fairyc test`는 네이티브 링킹 없이 동작하지만, 실제 실행 파일을 생성·실행하는 `build`, `run`, `test --native`에는 MSVC/Windows SDK 또는 GCC/Clang이 필요합니다.

## 컴파일러 CLI 빌드

`fairyc` 저장소 루트에서 다음 명령을 실행합니다.

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
```

Linux에서는 출력 파일명을 `-o build/fairyc`로 지정합니다. 이후 `./build/fairyc.exe`를 직접 실행하거나 `build/` 폴더를 `PATH`에 추가합니다.

## 설치 확인

빌드한 바이너리로 버전과 타깃 상태를 확인합니다.

```powershell
./build/fairyc.exe --version
./build/fairyc.exe targets
```

현재 Stage 0에서 확인되는 출력 예시는 다음과 같습니다.

```text
Fairy C Stage 0
compiler: development
sdk: fairyc-sdk-stdlib/0 0.0.0-stage0.87
language: fairyc-language/20
runtime: fairyc-runtime/0
revision: unknown
modified: unknown
```

`fairyc targets`는 `x86_64-linux-gnu`와 `x86_64-windows-msvc`를 표시하면서 현재 `supported: no` 및 미충족 근거(`complete-runtime`, `complete-standard-library` 등)를 함께 출력합니다. 이는 설치 오류가 아니라 공식 타깃 적합성 인증이 아직 미완성임을 뜻하는 정상 출력이며, 호스트 C 툴체인이 있으면 로컬 빌드와 실행은 바로 진행할 수 있습니다.

## 준비 단계에서 자주 겪는 오류

- **종료 코드 `6`으로 빌드 실패**: Windows에서 MSVC / Windows SDK를 찾지 못했거나 Linux `PATH`에 `gcc` / `clang`이 없을 때 발생합니다.
- **작업 디렉터리 오류**: `check`, `build`, `run`, `test`는 현재 디렉터리에 있는 `fairy.toml`을 기준으로 동작합니다.
- **다음 단계**: [첫 프로그램](../first-program/)으로 이동해 패키지를 생성하고 실행해 봅니다.
