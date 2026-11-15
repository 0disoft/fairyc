---
title: Instalación y configuración
description: Compila el compilador Stage 0 fairyc desde el código fuente y asegúrate de que tu cadena de herramientas C esté lista.
---

Todavía no existe un instalador independiente (`.msi`) ni paquetes precompilados para Linux. Sin embargo, puedes descargar el **archivo ZIP de vista previa para Windows x64** desde GitHub Releases o **compilar la versión actual de `main` desde el código fuente**.

| Método | Versión / referencia | Cuándo conviene |
| --- | --- | --- |
| **Opción A: ZIP de vista previa** | `v0.1.0-preview.1` (`2026-10-25`, commit `682d6591`) | Probar la CLI básica en Windows x64 sin instalar Go ni Bun |
| **Opción B: Compilar desde fuente** | Compilación de desarrollo `main` (`sdk: 0.0.0-stage0.87`, commit `e1d48cb7`) | Usar funciones recientes como `new --template win32`, `package` o compilar en Linux |

> Nota: Como `v0.1.0-preview.1` es una instantánea anterior, los comandos añadidos recientemente como `fairyc new --template win32` y `fairyc package` requieren la **Opción B (Compilar desde fuente)**.

## Compilador de C del sistema (requisito común)

`fairyc check` y `fairyc test` en modo intérprete no invocan un compilador de C, pero `fairyc build`, `fairyc run` y `fairyc test --native` sí requieren una cadena de herramientas C nativa:

- **Windows**: Visual Studio / Build Tools con el **compilador C/C++ MSVC x64** y el **Windows SDK**. (Para recursos Win32 y `fairyc package` también se requiere **Python 3.11+** en `PATH`.)
- **Linux**: `gcc` o `clang` disponible en `PATH`.

## Opción A: Usar el ZIP de vista previa en Windows (`v0.1.0-preview.1`)

Para probar la CLI principal en Windows x64 sin instalar Go ni Bun, descarga `fairyc-windows-amd64-preview-candidate.zip` y su archivo `.sha256` desde [GitHub Releases (`v0.1.0-preview.1`)](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.1):

```powershell
# Verifica el hash SHA-256, descomprime y comprueba la versión
Get-FileHash .\fairyc-windows-amd64-preview-candidate.zip -Algorithm SHA256
Expand-Archive .\fairyc-windows-amd64-preview-candidate.zip -DestinationPath .\fairyc-preview
.\fairyc-preview\fairyc.exe --version
```

## Opción B: Compilar `fairyc` desde fuente (`main`)

Para usar las funciones Stage 0 más recientes (`--template win32`, `fairyc package`, compilaciones en Linux GNU x64), asegúrate de tener **Git, Go y Bun**, clona el [repositorio de fairyc en GitHub](https://github.com/0disoft/fairyc) y entra en la raíz del proyecto:

```bash
git clone https://github.com/0disoft/fairyc.git
cd fairyc
```

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
