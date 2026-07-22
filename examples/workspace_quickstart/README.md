# Package workspace quickstart

This is the smallest checked-in package-aware Fairy C program. The root `main` package imports the
direct local `answer` package through `fairy.toml`; the native process returns 42.

From the Fairy C repository root in PowerShell, first build the local compiler as described in
`DEVELOPMENT.md`, then run:

```powershell
Push-Location examples/workspace_quickstart
..\..\build\fairyc.exe experimental check-workspace
..\..\build\fairyc.exe experimental run-workspace --locked --target x86_64-windows-msvc --profile optimized
Pop-Location
```

The first command creates `fairy.lock` in this example checkout. The second command requires those
exact manifest, source, dependency, SDK, and lock identities, compiles the verified package graph
through C11 and local x64 MSVC, executes a temporary PE, and prints a receipt beginning with
`native workspace result: 42`. It does not use WSL and does not execute the FY program through Go.

To retain an executable instead, replace the run command with:

```powershell
..\..\build\fairyc.exe experimental build-workspace --locked --target x86_64-windows-msvc --profile optimized --output build/workspace-quickstart.exe
```

The output path is relative to this example directory and is never overwritten.
