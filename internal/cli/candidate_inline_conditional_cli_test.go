package cli

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCandidateInlineConditionalPersistentCache(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", "")
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			root := inlineCLIWorkspace(t)
			packet := filepath.Join(root, "observed.bin")
			t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
			command := copiedClosureCommand{t: t, image: image, root: root}
			const providerPath = "libs/answer/src/answer.fy"
			provider := strings.Replace(inlineCLIProvider, "return x+2;", `var i32 result=x;
result=if (x<0) { 1/0 } else if (x==0) { 17 } else { result+2 };
return result;`, 1)
			writeTestSource(t, root, providerPath, provider)

			inlineCallsCLIExactCache(t, &command, 0, "cache: cold stored", "check")
			inlineCallsCLIWarm(t, &command, 42)
			checkInlineCLIPacket(t, packet)

			// Change which arm is selected without editing the caller.
			before := copiedClosureCacheSnapshot(t, root)
			updated := strings.Replace(provider, "x==0", "x==40", 1)
			writeTestSource(t, root, providerPath, updated)
			inlineCallsCLIStaleLock(t, &command)
			inlineCallsCLIRebuilt(t, &command, 17, false, "run")
			if reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, root)) {
				t.Fatal("condition edit did not publish a new project generation")
			}
			inlineCallsCLIWarm(t, &command, 17)
			checkInlineCLIPacket(t, packet)
		})
	}
}
