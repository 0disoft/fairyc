---
title: Introducción
description: Por qué existe Fairy C, qué ideas de sintaxis utiliza y cómo se compara con Brownie C.
---

Escribir en C te da control directo sobre la memoria y las API del sistema operativo, pero es fácil olvidar un `free()`, tocar datos ya liberados o dejar que un entero dé la vuelta silenciosamente al desbordarse.

Fairy C es un lenguaje experimental de programación de sistemas que mantiene la ejecución nativa al estilo C sin recolección de basura, permitiendo que el compilador detecte a tiempo errores de propiedad, préstamos y aritmética.

## Ideas clave del lenguaje

- **Los recursos saben limpiarse solos**: Los tipos que mantienen manejadores o búferes pueden declararse como `resource` con un bloque `cleanup { ... }`. Cuando un recurso vivo sale de ámbito, el compilador programa su limpieza automáticamente.
- **Mover vs. prestar (`ref` y `ref mut`)**: Puedes transferir la propiedad de un recurso o prestar acceso de solo lectura (`ref T`) o mutable exclusivo (`ref mut T`) mientras el compilador comprueba que no haya conflictos superpuestos.
- **Aritmética de enteros verificada**: El desbordamiento de enteros y la división por cero se detienen limpiamente (trap) en lugar de dar la vuelta en silencio.
- **Etiquetas `@effects(...)` explícitas**: Las funciones que interactúan con la consola, el sistema de archivos, el reloj o la red declaran esos efectos en su firma.
- **Interoperabilidad directa con C**: Las estructuras `@repr(C)` y `unsafe extern C fn` permiten que Fairy C llame directamente a bibliotecas C existentes y a las API de Win32.

## Dónde encaja hoy

Fairy C es muy adecuado para pequeñas herramientas CLI nativas, envoltorios de ABI de C y utilidades enfocadas en Win32.

Dado que el compilador es un **prototipo Stage 0**, utiliza los comandos estables del espacio de trabajo (`new`, `check`, `build`, `run`, `test`, `explain`, `targets`) y trata los subcomandos `fairyc experimental ...` como herramientas internas para el desarrollo del compilador. La conformidad formal de destinos (`fairyc targets` actualmente muestra `supported: no`) y una biblioteca estándar completa siguen en progreso.

## Fairy C vs. Brownie C

| Aspecto | Fairy C (`.fy`) | Brownie C (`.brc`) |
| --- | --- | --- |
| Propósito | Programas y bibliotecas nativos independientes | Reglas integradas y comprobaciones de bytes dentro de un host |
| Ejecución | Compilado vía C11 a ejecutables nativos | Ejecutado dentro de una VM de bytecode acotada |
| Acceso a SO y C | `unsafe extern C fn` directo y E/S de stdlib | Sin llamadas arbitrarias a C, archivos ni acceso a red |
| Presupuesto | Ejecución nativa normal del SO | Acotado por un presupuesto de instrucciones `--fuel` |
