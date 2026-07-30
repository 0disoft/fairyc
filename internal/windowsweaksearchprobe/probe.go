// Package windowsweaksearchprobe owns one closed NOLIBRARY-versus-LIBRARY archive proof.
package windowsweaksearchprobe

const (
	ObjectName        = "weakmode.obj"
	ArchiveName       = "weakmode.lib"
	NoLibraryMember   = "nolib.obj"
	LibraryMember     = "library.obj"
	NoLibraryEntry    = "fy_nl_e"
	LibraryEntry      = "fy_lib_e"
	NoLibraryWeak     = "fy_nl_w"
	NoLibrarySentinel = "fy_nl_s"
	LibraryWeak       = "fy_lib_w"
	NoLibraryFallback = "fy_nl_f"
	LibraryFallback   = "fy_lib_f"
	ExpectedExit      = int32(0)
)

// Unit contains the exact generated-object, archive-member, and runtime identities.
type Unit struct {
	CallerSource      string
	NoLibrarySource   string
	LibrarySource     string
	ObjectName        string
	ArchiveName       string
	NoLibraryMember   string
	LibraryMember     string
	NoLibraryEntry    string
	LibraryEntry      string
	NoLibraryWeak     string
	NoLibrarySentinel string
	LibraryWeak       string
	NoLibraryFallback string
	LibraryFallback   string
	ExpectedExit      int32
}

// Generate returns deterministic C11 sources for two independent archive members.
func Generate() Unit {
	return Unit{
		CallerSource: callerSource, NoLibrarySource: noLibrarySource, LibrarySource: librarySource,
		ObjectName: ObjectName, ArchiveName: ArchiveName, NoLibraryMember: NoLibraryMember, LibraryMember: LibraryMember,
		NoLibraryEntry: NoLibraryEntry, LibraryEntry: LibraryEntry, NoLibraryWeak: NoLibraryWeak, NoLibrarySentinel: NoLibrarySentinel, LibraryWeak: LibraryWeak,
		NoLibraryFallback: NoLibraryFallback, LibraryFallback: LibraryFallback, ExpectedExit: ExpectedExit,
	}
}

const callerSource = `/* fairyc weak-external archive search proof */
extern int fy_nl_e(void);
extern int fy_lib_e(void);

int main(void) {
    if (fy_nl_e() != 17) {
        return 21;
    }
    return fy_lib_e() == 31 ? 0 : 22;
}
`

const noLibrarySource = `/* must remain unextracted for IMAGE_WEAK_EXTERN_SEARCH_NOLIBRARY */
__declspec(noinline) int fy_nl_w(void) {
    return 29;
}

__declspec(noinline) int fy_nl_s(void) {
    return 37;
}
`

const librarySource = `/* must be extracted for IMAGE_WEAK_EXTERN_SEARCH_LIBRARY */
__declspec(noinline) int fy_lib_w(void) {
    return 31;
}
`
