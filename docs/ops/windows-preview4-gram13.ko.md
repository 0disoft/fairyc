# GRAM13에서 Fairy C preview.4 검증

공개 Windows AMD64 ZIP의 소스 커밋은 `d08fe22b21fb6eb8481915c34abb463a9aa69cb3`이고, 크기는 11,001,080바이트예요. [GitHub 릴리스](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.4)

```text
d71ac2491e8059b54fe382613827d74046cb79e9f0f361364b6c29dcc454b930
```

## 실행

1. [Windows 공개 ZIP 검증 스크립트](../../tools/windows-preview/verify-preview2.ps1)를 GRAM13의 다운로드 폴더에 `verify-fairyc-preview.ps1`로 저장해요. GitHub 스크립트 화면에서 **Raw** 또는 다운로드 버튼을 사용해요.
2. 64비트 Windows PowerShell 5.1 또는 PowerShell 7을 열어요. 네이티브 단계에는 설치된 x64 MSVC와 Windows SDK가 필요해요. Go, Node.js, Bun, VS Code, 저장소 체크아웃 없이 검증할 수 있어요.
3. 아래 명령을 실행하면 preview.4의 공개 ZIP과 체크섬을 다운로드해요. 기본 결과 폴더는 `$HOME\Downloads`예요. 다른 위치는 이미 만들어진 폴더를 `-OutputDirectory`로 지정해요.

```powershell
powershell.exe -NoProfile -File "$HOME\Downloads\verify-fairyc-preview.ps1" -Release v0.1.0-preview.4
$LASTEXITCODE
```

이미 다운로드했다면 ZIP과 `.zip.sha256` 파일을 같은 폴더에 두고 `-ZipPath`를 지정해요. 스크립트가 두 파일을 새 결과 폴더로 복사해서 동일하게 검증해요. [공개 ZIP](https://github.com/0disoft/fairyc/releases/download/v0.1.0-preview.4/fairyc-windows-amd64-preview-candidate.zip) · [체크섬](https://github.com/0disoft/fairyc/releases/download/v0.1.0-preview.4/fairyc-windows-amd64-preview-candidate.zip.sha256)

```powershell
powershell.exe -NoProfile -File "$HOME\Downloads\verify-fairyc-preview.ps1" `
  -Release v0.1.0-preview.4 -ZipPath "$HOME\Downloads\fairyc-windows-amd64-preview-candidate.zip"
$LASTEXITCODE
```

## 통과 기준

| 단계 | 확인 내용 |
| --- | --- |
| 체크섬 | 고정 ZIP 크기와 SHA-256, 공개 체크섬 파일을 대조해요. 압축을 푼 뒤 `SHA256SUMS.txt`의 파일 7개와 두 실행 파일의 AMD64 PE 헤더를 확인해요. |
| 버전·LSP | 컴파일러의 정확한 커밋과 `modified: false`, LSP 버전 및 표준 입출력 초기화·종료 응답을 확인해요. |
| Starter | 새 프로젝트에서 `new`, `check`, `check --locked`, `test --locked`가 통과하고 테스트 1개가 성공해요. |
| 네이티브 | 개발·릴리스 각각에서 `test --native`, `build`, 생성 EXE 직접 실행, `run`이 통과해요. 직접 실행과 `run`은 출력 없이 종료 코드 0을 반환해요. |
| 실행 후 무결성 | 배포 파일 7개와 starter 소스·잠금 파일의 해시가 실행 전과 일치해요. |

성공하면 `PASS: 15/15 process checks`와 종료 코드 `0`이 표시돼요. 새 `fairyc-preview4-*` 폴더의 `results.json`에서 `tag`가 `v0.1.0-preview.4`, `passed`가 `true`, `host`가 GRAM13의 실제 Windows 컴퓨터 이름인지 확인해요. 결과를 공유할 때는 `results.json`과 필요한 `logs\` 파일을 보내면 돼요. 폴더에는 개발·릴리스로 생성한 EXE도 남아요.

실패하면 종료 코드 `1`과 `failedStage`, `error`, 건너뛴 단계가 기록돼요. ZIP 해시가 다르면 압축을 풀기 전에 중단해요. 네이티브 단계가 실패하면 `logs\`에서 MSVC·SDK 발견 오류인지 빌드·실행 오류인지 확인해요.
