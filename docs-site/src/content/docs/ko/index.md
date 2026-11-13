---
title: Fairy C
description: 명시적인 자원 관리, 빌림 검사, C 연동을 갖춘 실험적 네이티브 시스템 프로그래밍 언어입니다.
---

Fairy C는 C 언어처럼 가볍고 빠른 네이티브 프로그램을 만들면서도, 메모리 해제 누락이나 잘못된 참조 사용 같은 실수를 컴파일러가 미리 잡아 주는 시스템 프로그래밍 언어입니다.

현재 `fairyc` 컴파일러는 Go로 만든 Stage 0 프로토타입이며, `.fy` 소스를 C11 코드로 바꾼 뒤 Windows의 MSVC나 Linux의 GCC/Clang을 통해 실제 실행 파일(`.exe` 등)로 빌드합니다.

## 어디부터 볼까요?

- **시작하기**: [소개](./start/introduction/) · [설치와 환경 준비](./start/installation/) · [첫 프로그램](./start/first-program/)
- **문법 배우기**: [언어 기초](./language/basics/) · [메모리와 자원](./language/memory-and-resources/) · [프로젝트 구성, 모듈과 의존성](./language/modules-and-projects/)
- **예제**: [C 연동과 Win32 데스크톱 앱 입문](./examples/c-interop-and-win32/)
- **CLI와 지원 현황**: [CLI 사용법과 자주 만나는 오류](./cli/commands-and-errors/) · [현재 지원 범위와 제한 사항](./support/status-and-limits/)

## Brownie C와는 무엇이 다른가요?

Fairy C가 직접 실행 파일이나 OS 연동 프로그램을 만드는 **네이티브 언어**라면, **Brownie C**는 프로그램 안에 끼워 넣어 작은 규칙이나 바이트 검사만 안전하게 돌리는 **임베디드 언어**입니다. 실제로 Brownie C 컴파일러와 가상 머신 자체가 바로 이 Fairy C로 작성되고 있습니다.
