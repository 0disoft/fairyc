# Third-party notices

Third-party licenses are not replaced by Fairy C's Apache-2.0 license.

## SCIP index export

The [SCIP Go bindings](https://github.com/scip-code/scip/tree/v0.10.0) and their
[beaut dependency](https://github.com/sourcegraph/beaut/tree/627e4c25335a)
are licensed under Apache-2.0. See [LICENSE](LICENSE) for the license text.

The bindings use [protobuf-go](https://github.com/protocolbuffers/protobuf-go).
Its license follows.

```text
Copyright (c) 2018 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

## Benchmark report renderer

`tools/dev/fairyc-bench` uses go-text/typesetting, golang.org/x/image and renamed Noto font
subsets for multilingual chart exports. Sources and full notices are in
[tools/dev/fairyc-bench/fonts](tools/dev/fairyc-bench/fonts/README.md).

The report also embeds Feather 4.29.2 icons (MIT). Source and license:
[tools/dev/fairyc-bench/assets](tools/dev/fairyc-bench/assets/README.md).

## Go runtime and standard library

- Toolchain inspected: Go 1.27.1 (Windows AMD64, CGO disabled)
- License text: [GO-LICENSE.txt](GO-LICENSE.txt), copied from the installed Go
  toolchain LICENSE; upstream: https://go.dev/LICENSE
- Binary distributions include the full copyright, conditions and disclaimer.
- This top-level notice is not a substitute for checking component-specific
  terms when the linked toolchain or distributed contents change.

## github.com/BurntSushi/toml

- Version: v1.6.0
- License: MIT
- Purpose: strict TOML parsing for the human-authored Fairy C project manifest
- Source: https://github.com/BurntSushi/toml

```text
The MIT License (MIT)

Copyright (c) 2013 TOML authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```
