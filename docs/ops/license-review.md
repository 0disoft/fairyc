# License and distribution-notice review

- Reviewed: 2026-09-16
- Repository baseline: `f3e29321c2adc2c02168d4ad84e4c981ebc1c963`
- Status: engineering inventory and selection proposal; no license adopted

Subsequent owner decision (2026-09-16): Apache-2.0 was approved for project-owned
compiler/runtime/SDK material. [LICENSING.md](../../LICENSING.md) now owns the
applied scope. The inventory and options below are the pre-adoption record,
not a statement that the current project is still unlicensed. Existing TOML
terms are preserved; Go's license and all current project notice files are now
included by the local ZIP script. This does not establish full provenance or
authorize an official release.

## Verified inventory

| Surface | Evidence | Distribution action still required |
| --- | --- | --- |
| TOML parser | `go.mod`, `go.sum`, offline module graph and retained EXE metadata all identify `github.com/BurntSushi/toml v1.6.0`; imported by `internal/projectmanifest/manifest.go`; its module has no further module requirements | Preserve the existing MIT copyright/permission/disclaimer text in source and binary distribution materials |
| Existing project notice | [THIRD_PARTY_NOTICES.md](../../THIRD_PARTY_NOTICES.md) already contains the TOML MIT text, consistent with local cached COPYING and upstream v1.6.0 | Include it in the eventual distribution; its existence does not license Fairy C itself |
| Go runtime/standard library | Retained EXE reports Go 1.26.4, Windows/AMD64 and CGO_ENABLED=0; installed toolchain LICENSE contains Go's BSD-style three-condition terms | Include Go copyright, conditions and disclaimer with binary distribution; review applicable component-specific notices for the exact linked toolchain, not just third-party Go modules |
| Fairy C compiler/source | No tracked LICENSE or COPYING file found; this review does not establish contributor ownership from filenames or Git authors | Owner selects terms and confirms rights to apply them to contributed/copied/generated material |
| Embedded Fairy SDK | `internal/sdkstdlib/stdlib.go` embeds `sources/std/*.fy`; `vector.go` embeds the optional vector source | Explicitly include SDK in the selected licensing scope and cover any redistributed embedded/source material |
| Embedded/emitted C runtime | `internal/cruntime/runtime.go` embeds scalar/drop source; `internal/cstdlib` embeds target/runtime and probe sources | Explicitly cover runtime sources and runtime portions incorporated into generated programs; inventory copied material before making a provenance claim |
| Build prerequisites | ZIP contains no Go/Node/Bun/MSVC installers or redistributable DLLs | Do not bundle tools or Microsoft redistributables without separately checking their applicable terms; runtime/linkage of future user-program packages is a separate review |

The module graph is not a complete license bill of materials: Go standard-library
subcomponents, embedded data, source fixtures and generated output need separate
boundaries. A bounded copyright/SPDX/license-marker search in compiler/tool
sources found no additional third-party declarations, but absence of markers is
not proof of original authorship or complete provenance.

## Upstream sources and notice meaning

- [TOML v1.6.0 COPYING](https://raw.githubusercontent.com/BurntSushi/toml/v1.6.0/COPYING)
  is MIT. Preserve its copyright and permission text; keep the disclaimer with
  the complete license. The existing repository notice already provides this.
- [Go LICENSE](https://go.dev/LICENSE) requires source retention and reproduction
  of the copyright, conditions and disclaimer in binary distribution materials;
  it also restricts use of the named organizations/contributors for endorsement.
  The locally installed Go 1.26.4 LICENSE was inspected as well.
- [MIT text](https://opensource.org/license/mit) and
  [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) are the primary
  sources for the choices below. This report paraphrases them; it adds no license
  text to the project and grants no rights on behalf of any contributor.

## Compiler distribution is not the same as user-program distribution

`fairyc.exe` includes the Go runtime, TOML dependency and embedded compiler-owned
sources. Its notice bundle must cover that delivered material. Generated native
programs can contain Fairy C runtime/SDK implementations; their notice requirements
must be documented separately from the compiler's dependency list. Do not blindly
attach every compiler dependency to every user program, or claim all generated
output is exempt. Distinguish user-authored code from incorporated runtime code.

## Choices for owner approval

| Option | Benefit | Tradeoff |
| --- | --- | --- |
| Apache-2.0 for compiler, owned runtime and SDK | Explicit contributor patent grant and related termination provisions; one licensing scheme across owned components | More redistribution details: license copy, change markings and applicable attribution/NOTICE retention |
| MIT for compiler, owned runtime and SDK | Short, widely understood permissive terms and simpler notice handling | No express patent-grant clause in the license text; patent questions are less explicit than Apache-2.0 |

Recommendation: Apache-2.0 across the project-owned compiler/runtime/SDK if the
owner prioritizes explicit patent terms for a systems-language ecosystem. MIT
is a reasonable alternative if minimizing license-text complexity is the priority.
Neither option replaces third-party licenses, proves contributor ownership,
promises freedom from patents, or automatically decides licensing of user code.
This is a proposal, not a legal clearance or an applied project policy. If rights
ownership or patent concerns are material, obtain appropriate legal review.

## Gaps in the retained local ZIP

The previously tested local-only ZIP contains only the EXE, README and checksums;
it omits even the existing TOML notice. Do not publish it. The inventory intent
read its extracted EXE (SHA-256
`911fc1f19ab3fe85c82e7915b161c96969111e78afff7a1777ea88be744edd67`).
Metadata identifies `d9936781cf0a39a70944662b9d5ee901722f42e7` with
`vcs.modified=true`; this was a packaging rehearsal, not a clean release build.

Before distribution, after the owner chooses terms:

1. Confirm copyright holder/contributor authority and define compiler, runtime,
   SDK, documentation and example licensing scopes.
2. Add the selected project license and required third-party notices, including
   Go and any applicable linked subcomponent notices. Preserve TOML's existing text.
3. Include those files in ZIP inventory/hash verification and document downstream
   notice duties for incorporated runtime/SDK code.
4. Rebuild from a clean selected commit and verify exact artifact identity,
   archive contents and starter workflow; retain a reproducible source reference.
5. Finish release ownership, reporting route and clean-host checks in
   [public readiness](parser-public-readiness.md) before an explicit publication.

The configured `fairyc_license_inventory` passed offline metadata/module inspection.
Source/license files and official texts were read; no dependencies, licenses,
archive contents or production code were changed by this review. No complete
history/provenance audit, linked-subcomponent inventory, legal sign-off or release
validation is claimed. Those unresolved boundaries prevent a compliance-complete
claim; the current two-option decision does not require speculative code changes.
