# Fairy C Windows 컴파일러·에디터 미리보기

> **초안이라 공개하지 않았습니다.** 릴리스 태그와 제목 버전은 미정이고 공개 다운로드 URL도 없습니다. 파일의 정확한 로컬 경로와 SHA-256은 [자산 인벤토리](./windows-editor-preview-assets.json)에 정리해 두었습니다.

Windows AMD64에서 Stage 0 컴파일러와 VS Code 언어 기능을 직접 써 보는 서명되지 않은 실험 빌드입니다. 구문 검사, 의미 분석 테스트, 언어 서버는 MSVC, Go, 별도 Node.js, Bun 설치 없이 쓸 수 있습니다. 네이티브 빌드·테스트·실행에는 x64 MSVC와 Windows SDK가 필요합니다. VSIX는 VS Code 1.90 이상에서 설치하며, `.fy` 확장자는 그대로 유지합니다.

이번 구성은 컴파일러 ZIP에 `fairyc-lsp.exe`를 동봉하고, 별도 VSIX로 구문 강조·정의 이동·자동 완성·진단 기능을 제공합니다. 프로젝트 소유 코드는 Apache-2.0을 따르며, 의존성의 라이선스와 고지 파일도 함께 제공합니다.

컴파일러 ZIP은 깨끗한 커밋 `315ef56991d20fa0716927e4fcecb464231db6a8`에서 Go 1.27.1과 `-trimpath`, `-ldflags="-s -w"`로 만들었습니다. VSIX는 별도 패키지이며, 새 후보도 기존 파일을 덮어쓰지 않고 별도로 만들었습니다. 버전은 여전히 `0.1.2`입니다. SDK `0.0.0-stage0.87`과 언어 `20`은 각각 SDK와 언어의 식별자이며, 컴파일러 릴리스 번호가 아닙니다.

## 설치

다섯 파일을 받은 폴더에서 ZIP과 VSIX의 해시를 먼저 확인합니다.

```powershell
foreach ($name in @('fairyc-windows-amd64-preview-candidate.zip', 'fairyc-0.1.2.vsix')) {
    $expected = ((Get-Content -LiteralPath "$name.sha256" -Raw).Trim() -split '\s+')[0]
    $actual = (Get-FileHash -LiteralPath $name -Algorithm SHA256).Hash
    if ($actual -ine $expected) { throw "$name 해시가 일치하지 않습니다." }
}
```

1. ZIP을 풉니다.
2. `code --install-extension .\fairyc-0.1.2.vsix`로 확장을 설치합니다.
3. VS Code 사용자 설정의 `fairyc.server.path`에 압축을 푼 `fairyc-lsp.exe`의 절대 경로를 넣습니다.
4. VS Code를 다시 불러오고 신뢰할 수 있는 프로젝트를 연 뒤 `.fy` 파일을 엽니다. 언어 서버는 신뢰한 작업 폴더에서 시작합니다.

## 게시안 파일

기본 게시안은 다음 다섯 파일입니다.

| 파일 | 크기 | SHA-256 |
| --- | ---: | --- |
| `fairyc-windows-amd64-preview-candidate.zip` | 7,733,751 bytes | `6c6f7babb7d49035794051e71026c6916919ce06b591c0c5011a022fd322fcdf` |
| `fairyc-windows-amd64-preview-candidate.zip.sha256` | 확인 파일 | [인벤토리](./windows-editor-preview-assets.json) |
| `fairyc-0.1.2.vsix` | 480,180 bytes | `5da6b9240c829834ba578f8a334ed9c6d3105335a4555f6e3836e81766232612` |
| `fairyc-0.1.2.vsix.sha256` | 확인 파일 | [인벤토리](./windows-editor-preview-assets.json) |
| `fairyc-0.1.2.vsix.build.json` | 732 bytes | `637186cb4ade9ab583674fe8e97d396840f16c1ef4116d8712b34f43b30c0dce` |

ZIP 안의 실행 파일은 다음 해시와 일치해야 합니다.

| 실행 파일 | SHA-256 |
| --- | --- |
| `fairyc.exe` | `960e4439f1f59eb40fece6a8bf905cdef4980af8de3a9848054205ac96bf47b9` |
| `fairyc-lsp.exe` | `6ab141d972848b6027dbbfd045808290320026c39293e5f134a70265d47b1a82` |

내부 전송용 번들은 검증 증거로만 쓰며 게시 파일에 넣지 않습니다.

## 확인한 범위

- 후보 ZIP의 새 프로젝트 생성·검사·테스트·실행 등 16단계를 확인했습니다. 네이티브 테스트와 실행은 development·release 두 모드를 포함합니다.
- 새 VSIX 후보를 Windows PowerShell 5.1로 격리 설치하고, HP15의 실제 VS Code 1.143.1에서 확장 0.1.2의 파일 연결, 활성화, 정의 이동, 자동 완성, 저장 전 진단, 복구를 확인했습니다. 설치 기록에는 VSIX·빌드 출처 JSON·LSP의 해시가 연결되어 있습니다.
- 동봉한 PowerShell 5.1 검증기를 자식 PATH에서 Go, Node, Bun을 뺀 상태로 통과한 결과는 보존된 이전 VSIX를 포함한 전송 묶음의 기록입니다. 새 후보에 대해서는 이 전송 검증을 반복하지 않았습니다. 두 후보의 파일과 증거는 인벤토리에서 구분합니다.

이 결과는 깨끗한 PC에서의 실행을 뜻하지 않습니다. 별도 노트북 GRAM13의 에디터 재검증은 요청에 따라 생략했고, 통과한 검증으로 세지 않습니다. 이 생략은 이번 초안의 게시 차단 항목도 아닙니다.

## 관리자용 게시 전 확인

- 컴파일러 릴리스 이름과 태그는 아직 정하지 않았습니다. 이 문서에 임의의 버전을 붙이지 않습니다.
- 새 후보의 빌드 출처는 함께 게시할 `fairyc-0.1.2.vsix.build.json`에 기록되어 있습니다. 소스 리비전은 커밋 `25d4dfbfecaa0ed4ab40cc9064713864f6aa86a1`이고, `source.modified=false`입니다. 이 JSON은 VSIX 외부의 기록이며, 패키지 내부 메타데이터나 배포자 서명이 아닙니다.
- 커밋 `315ef56991d20fa0716927e4fcecb464231db6a8`의 `Windows native target smoke` CI는 아직 확인하지 않았습니다. 현재 `frontend.yml`의 `workflow_dispatch`에는 대상 리비전 입력이 없습니다. 공개 전에 후보 커밋 자체의 실행 결과를 확보해야 하며, 이후 main HEAD의 push CI를 후보 CI로 대신하지 않습니다.
- 2026-10-27에 확인한 공개 다운로드와 비공개 보안 신고 경로는 과거 기록입니다. 공개 전에 다시 확인하고, 관련 CI 실패도 사용자 경로에 영향을 주는 것부터 처리합니다.
- 다섯 파일의 게시, 릴리스 태그 생성, 푸시는 별도 승인이 필요합니다.
