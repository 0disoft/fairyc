---
title: Instalación y configuración
description: Compila el compilador Stage 0 fairyc desde el código fuente y asegúrate de que tu cadena de herramientas C esté lista.
---

Todavía no hay un instalador independiente ni binarios precompilados para descargar. Debes compilar `fairyc` desde el código fuente del repositorio y combinarlo con el compilador de C de tu plataforma.

## Requisitos previos

1. **Git, Go y Bun**: Necesarios para compilar el compilador Stage 0 en Go (`tools/run-go.ts` envuelve a Go mediante Bun).
2. **Un compilador de C nativo** (necesario para `fairyc build`, `fairyc run` y `fairyc test --native`):
   - **Windows**: Visual Studio / Build Tools con el **compilador C/C++ MSVC x64** y el **Windows SDK**.
   - **Linux**: `gcc` o `clang` disponible en `PATH`.

> **Consejo:** `fairyc check` y `fairyc test` en modo intérprete no invocan un compilador de C, así que puedes verificar tipos e ideas antes de configurar MSVC o GCC.

## Clonar el repositorio

Si aún no tienes el código fuente en local, clona el [repositorio de fairyc en GitHub](https://github.com/0disoft/fairyc) y entra en la raíz del proyecto:

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

## Compilar `fairyc`

Desde la raíz del repositorio `fairyc`:

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o build/fairyc.exe ./cmd/fairyc
```

En Linux, usa `-o build/fairyc`. Añade `build/` a tu `PATH` (o ejecuta `./build/fairyc.exe` directamente).

## Verificar la compilación

Ejecuta `--version` y `targets`:

```powershell
./build/fairyc.exe --version
./build/fairyc.exe targets
```

Ejemplo de salida de `--version`:

```text
Fairy C Stage 0
compiler: development
sdk: fairyc-sdk-stdlib/0 0.0.0-stage0.87
language: fairyc-language/20
runtime: fairyc-runtime/0
revision: unknown
modified: unknown
```

Al ejecutar `fairyc targets`, tanto `x86_64-windows-msvc` como `x86_64-linux-gnu` muestran `supported: no`. Eso simplemente refleja que la cobertura formal de la especificación aún está incompleta en Stage 0; tus compilaciones y ejecuciones locales estarán listas en cuanto tengas instalada tu cadena de herramientas de C.
