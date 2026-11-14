---
title: Interoperabilidad con C y aplicaciones Win32
description: Llama a funciones de C con @repr(C) y unsafe extern C fn, y explora ejemplos reales de Win32 (win32_minimal y win32_hash).
---

Cuando quieras reutilizar bibliotecas C existentes o llamar directamente a las API de Windows Win32, Fairy C ofrece primitivas explícitas de diseño de memoria en C y funciones externas.

## 1. Tres piezas clave de interoperabilidad con C

### `@repr(C)` para diseños de estructura
Normalmente el compilador es libre de reordenar los campos de un `struct`. Añade `@repr(C)` al compartir una estructura a través de una frontera con C para que el diseño y la alineación coincidan con C (`examples/win32_hash/src/writer.fy`):

```c
@repr(C)
pub struct Observation {
    u32 phase;
    u32 error;
    u32 exists;
}
```

### `unsafe extern C fn` y `unsafe { ... }`
Declara funciones externas de C con `unsafe extern C fn` e invócalas dentro de `unsafe { ... }`. Colocar la llamada `unsafe` dentro del bloque `cleanup` de un `resource` mantiene limpio y seguro el resto de tu código Fairy C:

```c
@effects(filesystem)
unsafe extern C fn void hash_writer_drop(ptr mut void context);

@effects(filesystem)
pub resource Writer {
    ptr mut void handle;
    cleanup { unsafe { hash_writer_drop(this.handle); } }
}
```

Cuando el símbolo exportado en C tenga un nombre distinto al declarado en Fairy C, añade `@link_name("c_symbol")`:

```c
@link_name("strlen")
unsafe extern C fn usize c_strlen(ptr u8 text);
```

### Pasar callbacks a C (`examples/c_interop_callback`)
Se pueden pasar callbacks síncronos de Fairy a través de firmas con punteros sin procesar (`unsafe extern C fn` no puede aceptar directamente `ref T` seguro — `E5102`):

```c
unsafe extern C fn i32 legacy_filter(
    unsafe extern C fn i32(i32, ptr mut void) predicate, ptr mut void data);

unsafe fn i32 keep_large(i32 value, ptr mut void data) {
    return if (value >= 10) { 1 } else { 0 };
}
```

## 2. Configuración del puente Win32 (`fairy.windows.json`)

Los ejemplos de escritorio usan una configuración `fairy.windows.json` (`fairyc.windows-build/v1`) para enlazar adaptadores en C, recursos `.rc` y bibliotecas de Windows:

```json
{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}
```

## 3. Ejemplo 1: `examples/win32_minimal`

Una GUI mínima de Win32 que separa la fontanería del sistema operativo de la lógica de la aplicación:

- **Puente en C (`native/window.c`, `native/ime.c`)**: Creación de ventana, bucle de eventos Win32, composición Hangul/IME y APIs de DPI y fuentes.
- **Fairy C (`src/main.fy`)**: Diseño escalado por DPI (`on_size`, `on_dpi`), manejo de teclado y Tab (`on_key`, `on_tab`) y procesamiento de texto (`on_click`).

```powershell
cd examples/win32_minimal
fairyc build --release
.\build\win32-minimal.exe
```

En la medición de empaquetado para distribución `windows-gui-slim-20261111` (eliminando los ayudantes opcionales de pipe/smoke test mientras se mantienen la CRT estática y las importaciones de DLL de Windows), el binario release para Windows x64 MSVC midió `121,344` bytes con la CLI en Go y `123,904` bytes (`121 KiB`) con la CLI propia.

## 4. Ejemplo 2: `examples/win32_hash` (Fairy Hash)

Una utilidad combinada de escritorio y CLI `--hash <file> [report]` para integridad SHA-256 que combina `std::fs`, `std::bytes`, `std::process`, `std::io` y `@effects(console, filesystem, clock)` con un puente delgado en C (`native/window.c`, `native/reader.c`, `native/writer.c`).

```powershell
cd examples/win32_hash
fairyc build --release
.\build\win32-hash.exe --hash path\to\file
```

En la compilación release bloqueada para Windows x64 del `2026-11-13` (`fairyc-go.exe build --locked --release`, incrustando un `.ico` de 9 tamaños, `VERSIONINFO` y el manifiesto de Common Controls v6), `FairyHash.exe` midió `196,096` bytes (`191.5 KiB`; `188,416` bytes antes de incrustar esos recursos de icono y versión).

> Estas mediciones de tamaño corresponden a esas compilaciones release concretas de Windows x64 MSVC con sus adaptadores en C y CRT estática: son cifras reales de ejemplo, no una garantía general para todos los binarios de Fairy C.
