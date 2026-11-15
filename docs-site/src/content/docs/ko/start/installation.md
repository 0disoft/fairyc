---
title: 설치와 환경 준비
description: 소스에서 fairyc를 빌드하고 C 툴체인을 연결하는 방법을 설명합니다.
---

아직 미리 빌드된 설치 파일(`.msi`, `.tar.gz` 등)은 따로 배포되지 않습니다. 대신 저장소에서 `fairyc` 실행 파일을 직접 빌드한 다음, 운영체제의 C 컴파일러와 함께 사용합니다.

## 미리 준비할 것

1. **Git, Go, Bun**: Stage 0 컴파일러가 Go로 작성되어 있어 빌드에 필요합니다.
2. **운영체제 C 컴파일러** (`build`, `run`, `test --native` 실행 시 필요):
   - **Windows**: Visual Studio 또는 Build Tools에서 **MSVC x64 C/C++ 컴파일러**와 **Windows SDK**를 설치해 주세요.
   - **Linux**: `PATH`에서 `gcc` 또는 `clang`을 실행할 수 있어야 합니다.

> **참고:** 문법이나 빌림만 검사하는 `fairyc check`와 인터프리터 모드 `fairyc test`는 C 컴파일러 없이도 돌아갑니다. 하지만 실제 실행 파일을 만들 때는 MSVC나 GCC/Clang이 꼭 필요합니다.

## 저장소 내려받고 이동하기

아직 소스 코드가 없다면 [fairyc GitHub 저장소](https://github.com/0disoft/fairyc)를 복제한 뒤 저장소 루트 폴더로 이동합니다.

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

## `fairyc` 빌드하기

`fairyc` 저장소 루트 폴더에서 아래 명령을 실행합니다.

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
```

Linux라면 파일 이름에서 `.exe`만 빼고 `-o build/fairyc`로 빌드하면 됩니다. 편하게 쓰려면 `build` 폴더를 `PATH`에 등록해 두세요.

## 잘 설치됐는지 확인하기

```powershell
./build/fairyc.exe --version
./build/fairyc.exe targets
```

`--version`을 실행하면 다음과 비슷한 버전 정보가 출력됩니다.

```text
Fairy C Stage 0
compiler: development
sdk: fairyc-sdk-stdlib/0 0.0.0-stage0.87
language: fairyc-language/20
runtime: fairyc-runtime/0
revision: unknown
modified: unknown
```

`fairyc targets`를 실행해 보면 `x86_64-windows-msvc`와 `x86_64-linux-gnu` 항목 아래에 `supported: no`가 표시되는데, 설치가 잘못된 것은 아닙니다. 정식 사양 전체가 끝나기 전까지 프로토타입 상태로 표시해 둔 것뿐이며, C 툴체인이 준비되어 있다면 바로 다음 장의 첫 프로그램을 빌드하고 실행할 수 있습니다.
