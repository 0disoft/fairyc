@PRODUCT@ @VERSION@ — Windows x64

설치 없이 @EXECUTABLE@를 실행하세요. Windows 10 1703 이상(x64)이 필요합니다.
입력창에 내용을 넣고 '실행'을 누르면 결과를 표시합니다.
Alt+I: 입력창 / Alt+R: 실행 / Tab·Shift+Tab: 이동 / Enter: 실행 / Esc: 닫기
파일 읽기(Alt+O)는 파일을 비동기로 읽어 바이트 수와 LF 줄바꿈 수를 표시합니다.
진행률·경과 시간을 표시하며 취소(Alt+X, 작업 중 Esc)는 창을 유지합니다.

CLI (PowerShell): & ".\@EXECUTABLE@" --process "내용"
파일 작업: & ".\@EXECUTABLE@" --scan "C:\data\input.bin"
취소 검증: & ".\@EXECUTABLE@" --scan "C:\data\input.bin" --cancel-after 65536
파일 작업은 진행률과 최종 결과를 줄마다 JSON으로 출력합니다.
종료 코드: 완료 0 / 취소 2 / 오류 1. 취소 기준은 최대 64 KiB 단위로 확인합니다.
입력 내용은 외부 서비스로 전송하지 않습니다.

배포 파일: @EXECUTABLE@, README.txt, LICENSE.txt, SHA256SUMS.txt
코드서명은 적용되지 않았습니다. 체크섬은 배포자 신원을 인증하지 않습니다.
