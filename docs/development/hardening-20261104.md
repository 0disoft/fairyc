# Hardening review follow-up, 2026-11-04

Windows local evidence is recorded per slice below. Native Linux execution remains
queued for one combined validation; no Linux pass is implied.

## 1. Stable contract-state type depth

Reject more than 256 compound levels before long prefix suffixes reach type
interning. Boundary and 64 KiB prefix-bomb tests run on Windows. Linux: rerun
`go test -mod=readonly ./internal/contractstatemir ./internal/packageartifact`.
