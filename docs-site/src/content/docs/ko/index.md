---
title: Fairy C
description: 명시적인 소유권, 빌림 검사, C 연동을 갖춘 실험적 네이티브 시스템 프로그래밍 언어입니다.
---

Fairy C는 명시적인 자원 소유권, 컴파일 타임 빌림 검사, 검사된 정수 산술(checked arithmetic), 효과 표기(`@effects(...)`), 그리고 가비지 컬렉터 없는 C ABI 연동을 제공하는 실험적 네이티브 시스템 프로그래밍 언어입니다.

현재 `fairyc` 컴파일러는 Go로 작성된 Stage 0 프로토타입(`fairyc-language/20`)이며, C11 백엔드와 호스트 C 툴체인을 거쳐 네이티브 실행 파일을 생성합니다.

## 시작하기

- **시작**: [소개](./start/introduction/) · [설치와 환경 준비](./start/installation/) · [첫 프로그램](./start/first-program/)

## Brownie C와의 관계

Fairy C는 OS API와 C 라이브러리를 직접 다루는 네이티브 프로그램·라이브러리를 작성하기 위한 시스템 언어입니다.

**Brownie C**는 Fairy C로 구현 중인 별도의 임베디드 규칙·바이트 변환 언어와 제한된 VM 런타임입니다. 직접 네이티브 실행 파일이나 Win32/C 연동 프로그램을 만들고 싶다면 Fairy C를 사용합니다.
