---
title: Uso de la CLI y errores comunes
description: Comandos cotidianos de fairyc, códigos de salida y soluciones rápidas para errores frecuentes del compilador.
---

Aquí tienes una guía rápida práctica de los comandos estables de la CLI `fairyc` y los errores del compilador con los que es más probable que te encuentres al principio.

## Comandos cotidianos

| Comando | Cuándo usarlo |
| --- | --- |
| `fairyc new <name> --uid <uid> [--template <console\|win32>]` | Crear una carpeta de paquete nueva de consola o escritorio Win32 |
| `fairyc check [--locked]` | Comprobación rápida de sintaxis, tipos y préstamos sin compilar C |
| `fairyc build [--release] [--locked]` | Generar `build/<name>.exe` (Windows) o `build/<name>` (Linux) |
| `fairyc package [--locked] [--json]` | Compilar en modo release un proyecto de escritorio Win32 y validar un ZIP local sin firmar en `dist/` (Windows x64) |
| `fairyc run [--release] [--locked] [-- <args...>]` | Compilar y ejecutar de inmediato un binario temporal |
| `fairyc test [--native] [--locked]` | Ejecutar `tests/` en el intérprete o como binarios nativos (`--native`) |
| `fairyc explain <code>` | Leer detalles sobre un código de diagnóstico como `E2101` |

Cíñete a estos comandos estables: los subcomandos `fairyc experimental ...` son herramientas internas para el desarrollo del compilador.

## Códigos de salida de un vistazo

- **`0`**: Éxito (`run` también reenvía el valor de retorno distinto de cero de tu `main()` si no se detiene con un trap)
- **`2`**: Opciones o sintaxis de invocación de CLI incorrectas
- **`3`**: Error de compilación en tu código `.fy` o en el manifiesto
- **`5`**: Detención en tiempo de ejecución (trap por desbordamiento entero, división por cero o `assert` fallido)
- **`6`**: No se encontró o falló el compilador o enlazador de C del sistema
- **`7`**: Desajuste entre `fairy.toml` y `fairy.lock` (vuelve a ejecutar `fairyc check`)

## Errores comunes y cómo solucionarlos

Ejecuta `fairyc explain <CODE>` para cualquier código que veas. La mayoría de los errores iniciales caen en uno de estos casos:

1. **`E1001` (variable sin inicializar)**: Escribe `var i32 count = 0;` en lugar de `var i32 count;`.
2. **`E1006` / `E0009` (tipos enteros mezclados o comparaciones encadenadas)**: Convierte explícitamente con `as` y escribe `(0 < x) && (x < 10)` en lugar de `0 < x < 10`.
3. **`E1101` (uso después de mover)**: Pasaste un `resource` por valor e intentaste usarlo de nuevo. Préstalo con `ref` o `ref mut` si quien llama todavía lo necesita.
4. **`E2101` (conflicto de préstamo)**: Mutaste o moviste una variable mientras un préstamo `ref` seguía usándose más adelante en el bloque. Termina de usar el préstamo antes de mutar.
5. **`E4101` (`Result` ignorado)**: Maneja los retornos `Result` con `switch`, `?` o `discard` explícito.
6. **`E7102` (falta `@effects(...)`)**: Añade el efecto de la función llamada (como `console` o `filesystem`) al atributo `@effects(...)` de tu función.
