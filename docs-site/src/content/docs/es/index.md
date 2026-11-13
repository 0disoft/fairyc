---
title: Fairy C
description: Un lenguaje experimental de programación de sistemas nativos con limpieza explícita de recursos, verificación de préstamos e interoperabilidad con C.
---

Fairy C es un lenguaje experimental de programación de sistemas diseñado para generar binarios nativos pequeños como C, detectando en tiempo de compilación fugas de recursos, préstamos colgantes y desbordamientos de enteros.

El compilador actual `fairyc` es un prototipo Stage 0 escrito en Go. Compila paquetes `.fy` a través de un backend C11 usando la cadena de herramientas de C de tu sistema (MSVC + Windows SDK en Windows, o GCC/Clang en Linux).

## Por dónde empezar

- **Primeros pasos**: [Introducción](./start/introduction/) · [Instalación y configuración](./start/installation/) · [Primer programa](./start/first-program/)
- **Aprender el lenguaje**: [Variables, tipos y operadores](./language/basics/) · [Funciones y flujo de control](./language/control-flow-and-functions/) · [Estructuras, enums y switch](./language/structs-enums-and-errors/) · [Arreglos, slices y vistas](./language/arrays-and-slices/) · [Propiedad, préstamos y recursos](./language/memory-and-resources/) · [Módulos, paquetes y std](./language/modules-and-projects/)
- **Ejemplos**: [Interoperabilidad con C y aplicaciones Win32](./examples/c-interop-and-win32/)
- **CLI y estado**: [Uso de la CLI y errores comunes](./cli/commands-and-errors/) · [Soporte actual y limitaciones](./support/status-and-limits/)

## Fairy C vs. Brownie C

Mientras **Fairy C** crea ejecutables nativos independientes y bibliotecas de C/SO, **Brownie C** es un lenguaje aparte de reglas integradas y transformación de bytes que se ejecuta dentro de una VM acotada. El propio compilador y la VM de Brownie C están escritos en Fairy C.
