---
title: 설치와 환경 준비
description: 소스에서 fairyc를 빌드하고 C 툴체인을 연결하는 방법을 설명합니다.
---

별도 설치 마법사(`.msi` 등)나 Linux용 프리빌드 아카이브는 아직 제공되지 않습니다. 대신 **GitHub Releases의 Windows x64 프리뷰 ZIP**을 받아 바로 체험하거나, **저장소 소스에서 최신 `main` 버전을 직접 빌드**해 사용할 수 있습니다.

| 설치 방식 | 대응 버전 / 기준 | 포함 범위 및 권장 용도 |
| --- | --- | --- |
| **방법 A: 프리뷰 ZIP** | `v0.1.0-preview.1` (`2026-10-25`, 커밋 `682d6591`) | Go/Bun 없이 Windows x64 콘솔·언어 코어를 빠르게 시험할 때 |
| **방법 B: 소스 빌드** | `main` 개발 빌드 (`sdk: 0.0.0-stage0.87`, 커밋 `e1d48cb7`) | `new --template win32`, `package` 등 최신 Win32/데스크톱·Linux 기능을 쓸 때 |

> 참고: 프리뷰 ZIP(`v0.1.0-preview.1`)은 초기 스냅샷이므로, `--template win32`나 `fairyc package` 같은 최근 기능은 <strong>방법 B(소스 빌드)</strong>를 사용해 주세요.

## 운영체제 C 컴파일러 준비 (공통)

문법·빌림 검사(`fairyc check`)와 인터프리터 테스트(`fairyc test`)는 C 컴파일러 없이도 실행되지만, 실제 네이티브 바이너리를 만드는 `build`, `run`, `test --native`에는 운영체제 C 툴체인이 필요합니다.

- **Windows**: Visual Studio 또는 Build Tools에서 **MSVC x64 C/C++ 컴파일러**와 **Windows SDK**를 설치해 주세요. (`new --template win32` 리소스 준비 및 `fairyc package`를 쓰려면 <strong>Python 3.11+</strong>도 필요합니다.)
- **Linux**: `PATH`에서 `gcc` 또는 `clang`을 실행할 수 있어야 합니다.

## 방법 A: Windows 프리뷰 ZIP으로 바로 실행하기 (`v0.1.0-preview.1`)

Go/Bun을 설치하지 않고 Windows x64에서 코어 CLI만 바로 실행하려면 [GitHub Releases (`v0.1.0-preview.1`)](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.1)에서 `fairyc-windows-amd64-preview-candidate.zip`과 `.sha256` 파일을 내려받습니다.

```powershell
# SHA-256 확인 후 압축 해제
Get-FileHash .\fairyc-windows-amd64-preview-candidate.zip -Algorithm SHA256
Expand-Archive .\fairyc-windows-amd64-preview-candidate.zip -DestinationPath .\fairyc-preview
.\fairyc-preview\fairyc.exe --version
```

## 방법 B: 저장소 소스에서 최신 `fairyc` 빌드하기

최신 Stage 0 기능(`--template win32`, `fairyc package`, Linux GNU x64 빌드 등)을 쓰려면 **Git, Go, Bun**을 준비한 뒤 [fairyc GitHub 저장소](https://github.com/0disoft/fairyc)를 복제해 빌드합니다.

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

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
