@PRODUCT@ @VERSION@ — Windows x64

파일의 SHA-256 해시를 계산하는 작은 휴대용 프로그램입니다.
Windows 10 1703 이상(x64)에서 사용할 수 있습니다. 설치는 필요하지 않습니다.

사용 방법
1. ZIP을 원하는 폴더에 모두 풀고 @EXECUTABLE@를 실행하세요.
2. '파일 열기'에서 파일을 선택하면 해시와 파일 크기가 표시됩니다.
   파일 하나를 창이나 EXE 아이콘 위에 놓아도 됩니다. 계산·저장 중에는 드롭을 받지 않습니다.
3. '복사'는 SHA-256 값만 복사하고, '결과 저장'은 UTF-8 JSON 보고서를 저장합니다.
4. 계산 중 '취소'를 누르면 창을 유지한 채 작업을 취소합니다.
5. '예상 SHA-256'에 확인할 해시를 넣으면 일치 여부가 표시됩니다.
   대소문자를 구분하지 않으며 앞뒤 ASCII 공백은 무시합니다. 64자리 16진수만
   허용합니다. 유효한 예상 해시가 있으면 저장 보고서에도 예상값과 일치 여부가
   포함됩니다. 불일치여도 계산된 실제 해시는 복사·저장할 수 있습니다.

단축키
Alt+O: 파일 열기 / Alt+C: 해시 복사 / Alt+S: 결과 저장 / Alt+X: 취소
Alt+E: 예상 SHA-256 입력창
Tab·Shift+Tab: 포커스 이동 / Enter: 포커스된 버튼 실행

명령줄 사용 (PowerShell)
& ".\@EXECUTABLE@" "C:\data\input.bin" # GUI로 바로 열기
& ".\@EXECUTABLE@" --hash "C:\data\input.bin"
& ".\@EXECUTABLE@" --hash "C:\data\input.bin" "C:\data\result.json"
& ".\@EXECUTABLE@" --compare "C:\data\input.bin" "64자리_예상_SHA256" "C:\data\compare.json"

--compare의 마지막 보고서 경로는 생략할 수 있습니다. JSON의 expectedSha256은
소문자로 정규화된 예상값이고 matches는 일치 여부입니다. 종료 코드는 일치 0,
불일치 2, 오류 1입니다. 예상값 형식 오류는 errorCode 12로 표시됩니다.
GUI의 '복사'는 비교 결과와 관계없이 실제 계산된 SHA-256 값만 복사합니다.

계산 중 원본 변경이 감지되면 결과 복사·저장을 막습니다. 파일 쓰기가 끝난 뒤
다시 계산하세요. 원본 파일이나 같은 파일의 하드링크에는 보고서를 덮어쓰지 않습니다.
저장할 수 없다면 쓰기 권한과 저장 위치를 확인하세요.

배포 파일
@EXECUTABLE@: 실행 파일
README.txt: 이 안내문
LICENSE.txt: Apache License 2.0 전문
SHA256SUMS.txt: 위 세 파일의 SHA-256 체크섬

체크섬은 파일 손상 여부를 확인하는 용도이며 배포자의 신원을 인증하지는 않습니다.
이 패키지에는 코드서명이 적용되지 않았습니다.
문의·소스: https://github.com/0disoft/fairyc
