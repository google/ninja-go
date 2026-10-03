# frontend

This package writes build status updates using the Android Ninja frontend
protocol, which Soong reads to render its UI.

This package comes from other projects and was then modified. Each file keeps
its original license:

| File | Origin | License |
| :--- | :--- | :--- |
| `frontend.go`, `proto/doc.go` | Modified from Siso's `toolsupport/soongutil` package (https://chromium.googlesource.com/build/+/refs/heads/main/siso/toolsupport/soongutil/) | BSD-3-Clause, see [LICENSE](../LICENSE) |
| `proto/frontend.proto`, `proto/frontend.pb.go` | Android's Ninja fork, `src/frontend.proto` (https://android.googlesource.com/platform/external/ninja/) | Apache-2.0, see the file headers. It lives here because it shares a Go package with `proto/doc.go`. |
