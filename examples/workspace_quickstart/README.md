# Package workspace quickstart

This is the smallest checked-in package-aware Fairy C program. The root `main` package imports the
direct local `answer` package through `fairy.toml`; the native process returns 42 and the isolated
semantic test verifies that dependency without entering the production source set.

From the Fairy C repository root in PowerShell, first build the local compiler as described in
`DEVELOPMENT.md`, then run:

```powershell
Push-Location examples/workspace_quickstart
..\..\build\fairyc.exe check
..\..\build\fairyc.exe test --locked
..\..\build\fairyc.exe build --release --locked
..\..\build\fairyc.exe run --release --locked
Pop-Location
```

The first command creates `fairy.lock` in this example checkout. `test --locked` overlays
`tests/answer.fy`, validates its argument-free `@test fn void`, and runs it through verified MIR and
a fresh semantic interpreter state. It does not invoke MSVC, WSL, or a subprocess and creates no
artifact. Build and run then require those exact production manifest, source, dependency, SDK, and
lock identities, compile the verified package graph through C11 and local x64 MSVC, and return 42.
The native program is Fairy C; the compiler itself remains the Go Stage 0 implementation.

The stable build retains `build/workspace-quickstart.exe`. To select an explicit experimental output
path instead, use:

```powershell
..\..\build\fairyc.exe experimental build-workspace --locked --target x86_64-windows-msvc --profile optimized --output build/workspace-quickstart.exe
```

The experimental output path is relative to this example directory and is never overwritten.
