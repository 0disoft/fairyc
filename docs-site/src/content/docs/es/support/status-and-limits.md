---
title: Soporte actual y limitaciones
description: Qué funciona hoy en Stage 0 de Fairy C, qué sigue incompleto y qué significa fairyc targets.
---

Fairy C es un **prototipo activo de compilador Stage 0**. Saber qué funciona hoy frente a lo que sigue en progreso ayuda a fijar expectativas realistas.

## Qué funciona hoy

- Paquetes y dependencias locales por ruta `[workspace]` (`fairy.toml` + `fairy.lock`)
- Propiedad de `resource` con `cleanup`, préstamos `ref`/`ref mut`, vistas slice (`u8[]`) y retornos de préstamo de un único origen
- Aritmética entera verificada, `Option<T>`, `Result<T, E>`, propagación con `?` y `@effects(...)`
- Interoperabilidad directa con la ABI de C (`@repr(C)`, `unsafe extern C fn`, callbacks síncronos en C, ejemplos de puente Win32)
- Módulos básicos del SDK Stage 0 (`std::io`, `std::fs`, `std::bytes`, `std::alloc`, `std::text`, `std::vec`, `std::process`)

Con MSVC + Windows SDK en Windows o GCC/Clang en Linux, puedes compilar y ejecutar herramientas de consola y ejemplos de Win32 dentro de este subconjunto. Ten en cuenta que `fairyc targets` actualmente imprime `supported: no` tanto para `x86_64-windows-msvc` como para `x86_64-linux-gnu` porque la conformidad formal con la especificación de destinos y la cobertura completa de la stdlib aún no han terminado.

## Limitaciones actuales

- **No es una cadena de herramientas universal terminada**: No se afirma cobertura completa de plataformas, compatibilidad arbitraria con C++ ni garantías generales de rendimiento.
- **Biblioteca estándar parcial**: No incluye runtimes asíncronos, clientes/servidores HTTP ni toolkits de GUI multiplataforma.
- **Solo dependencias locales**: No hay registro remoto de paquetes ni descargador automático.
- **Restricciones del lenguaje**: Los genéricos se limitan a usos escalares restringidos, los callbacks y cierres que devuelven préstamos están restringidos (`E2104`, `E5104`), y `f32`/`f64` están pensados principalmente para diseños C con `@repr(C)`.
