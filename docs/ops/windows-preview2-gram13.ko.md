# GRAM13에서 Fairy C preview.2 검증

공개 Windows AMD64 ZIP의 소스 커밋은 `485592aa611626f427d248de659fa622e917d45f`예요.
ZIP 크기는 10,104,881바이트이고 SHA-256은 아래 값이에요. [GitHub 릴리스](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.2)

```text
03d1c345b21519bdb586357b0a1b8179334b0ded3954f28d794e1e6b4284b9ef
```

## 실행

1. [verify-preview2.ps1](../../tools/windows-preview/verify-preview2.ps1)을 GRAM13의 다운로드 폴더에 복사해요.
2. 일반 64비트 Windows PowerShell 5.1 또는 PowerShell 7을 열어요. 네이티브 단계에는 설치된 x64 MSVC와 Windows SDK가 필요해요. 스크립트와 배포본은 Go, Node.js, Bun, VS Code 또는 저장소 체크아웃 없이 실행할 수 있어요.
3. 아래 명령을 실행해요. 기본 출력 위치는 `$HOME\Downloads`예요. 다른 위치를 쓰려면 이미 만들어진 폴더를 `-OutputDirectory`로 지정해요.

```powershell
powershell.exe -NoProfile -File "$HOME\Downloads\verify-preview2.ps1"
$LASTEXITCODE
```

ZIP과 체크섬을 이미 받았다면 같은 폴더에 두고 `-ZipPath`를 지정해요.
체크섬 파일 이름은 ZIP 이름 뒤에 `.sha256`을 붙인 형태여야 해요. [공개 ZIP](https://github.com/0disoft/fairyc/releases/download/v0.1.0-preview.2/fairyc-windows-amd64-preview-candidate.zip) · [체크섬](https://github.com/0disoft/fairyc/releases/download/v0.1.0-preview.2/fairyc-windows-amd64-preview-candidate.zip.sha256)

```powershell
powershell.exe -NoProfile -File "$HOME\Downloads\verify-preview2.ps1" -ZipPath "$HOME\Downloads\fairyc-windows-amd64-preview-candidate.zip"
$LASTEXITCODE
```

## 확인 항목

| 단계 | 통과 기준 |
| --- | --- |
| 체크섬 | ZIP의 크기와 SHA-256이 고정값과 일치하고, 공개 체크섬 파일도 같은 해시를 가리켜요. 압축 해제 후 `SHA256SUMS.txt`에 있는 7개 파일과 두 실행 파일의 AMD64 PE 헤더를 확인해요. |
| 버전과 LSP | 컴파일러는 정확한 커밋과 `modified: false`를, LSP는 `fairyc-lsp 0.1.0`을 출력해요. 표준 입출력에서 초기화·종료 응답과 기본 기능을 확인해요. |
| Starter | 새 `hello-fairy` 프로젝트에서 `check`, `check --locked`, `test --locked`가 통과하고 테스트 1개가 성공해요. |
| 네이티브 | 개발·릴리스 모드 각각에서 `test --native`, `build`, 생성 EXE 직접 실행, `run`이 통과해요. 빌드와 테스트는 잠금 파일을 검증하고, 두 실행 방식 모두 출력 없이 종료 코드 0을 반환해요. |
| 실행 후 체크섬 | 배포 파일, starter 소스와 잠금 파일 해시가 실행 전과 같아요. |

전체 통과 시 `PASS: 15/15 process checks`와 종료 코드 0을 확인해요.
출력된 `results.json`에서 `host`에 GRAM13의 실제 Windows 컴퓨터 이름이 맞는지,
`passed`가 `true`인지 확인해요. 각 실행은 새 `fairyc-preview2-*` 폴더에
결과 JSON, 체크섬, 명령별 stdout·stderr, starter와 프로필별 EXE를 남겨요.
GRAM13 검증 결과를 공유할 때는 `results.json`과 필요한 `logs\` 파일을 보내면 돼요.

실패하면 종료 코드 1과 함께 `failedStage`, `error`, 실행하지 않은 단계가 기록돼요.
ZIP 해시가 다르면 압축을 풀거나 실행하지 않아요. 네이티브 단계에서 실패하면 해당 로그를 보고
MSVC·SDK 발견 오류와 컴파일·실행 오류를 구분해요. 이 절차는 설치된 도구를 사용하는 별도 호스트 검증이에요.
