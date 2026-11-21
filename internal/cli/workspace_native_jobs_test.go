package cli

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestNativeWorkspaceTestBuildsOverlapInExecutionOrder(t *testing.T) {
	t.Parallel()
	tests := []workspaceTest{
		{display: "first", entry: "tests::native::passes"},
		{display: "second", entry: "tests::native::fails"},
		{display: "third", entry: "tests::native::passes"},
	}
	for _, jobs := range []int{1, 2} {
		t.Run(fmt.Sprint(jobs), func(t *testing.T) {
			var log bytes.Buffer
			prepared := 0
			executed := []byte{}
			firstEntered := make(chan struct{})
			secondDone := make(chan struct{})
			prepare := func(_ mir.Program, _ string, _ cdriver.Profile, _ io.Writer) (nativeWorkspaceTestBuild, int) {
				index := prepared
				prepared++
				if index == 2 && len(executed) != len(tests)-jobs {
					t.Errorf("third build was prepared after %d executions with %d jobs", len(executed), jobs)
				}
				return func(stderr io.Writer) (nativeWorkspaceTestArtifact, int) {
					if jobs == 2 {
						switch index {
						case 0:
							close(firstEntered)
							select {
							case <-secondDone:
							case <-time.After(5 * time.Second):
								t.Error("first build did not overlap the second")
								return nativeWorkspaceTestArtifact{}, ExitBuildFailure
							}
						case 1:
							select {
							case <-firstEntered:
							case <-time.After(5 * time.Second):
								t.Error("second build did not overlap the first")
								return nativeWorkspaceTestArtifact{}, ExitBuildFailure
							}
							defer close(secondDone)
						}
					}
					fmt.Fprintf(stderr, "build%d ", index)
					return nativeWorkspaceTestArtifact{artifact: cdriver.NativeArtifact{Artifact: cdriver.Artifact{Bytes: []byte{byte(index)}}}}, ExitSuccess
				}, ExitSuccess
			}
			execute := func(artifact nativeWorkspaceTestArtifact, _ io.Reader, _ []string, _ time.Duration) (nativeWorkspaceTestOutcome, error) {
				index := artifact.artifact.Bytes[0]
				executed = append(executed, index)
				fmt.Fprintf(&log, "run%d ", index)
				if index == 1 {
					return nativeWorkspaceTestOutcome{ExitCode: 106, TrapCategory: "assertion-failed", Stdout: "payload"}, nil
				}
				return nativeWorkspaceTestOutcome{}, nil
			}
			outcomes, code := runNativeWorkspaceTestsWith(nativeTestFixtureProgram(), tests, cdriver.LowProfile, nativeWorkspaceTestInput{}, nil, time.Minute, true, false, jobs, &log, prepare, execute)
			if code != ExitSuccess || !slices.Equal(executed, []byte{0, 1, 2}) || log.String() != "build0 run0 build1 run1 build2 run2 " {
				t.Fatalf("code=%d execution=%v log=%q", code, executed, &log)
			}
			if len(outcomes) != len(tests) || outcomes[1].display != "second" || outcomes[1].trapCategory != "assertion-failed" || outcomes[1].stdout != "payload" {
				t.Fatalf("outcomes: %+v", outcomes)
			}
		})
	}
}

func TestNativeWorkspaceTestBuildFailureDrainsLaterBuild(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		tests := []workspaceTest{
			{display: "first", entry: "tests::native::passes"},
			{display: "second", entry: "tests::native::fails"},
			{display: "unstarted", entry: "tests::native::passes"},
		}
		entered, release := make(chan struct{}), make(chan struct{})
		prepared := 0
		prepare := func(_ mir.Program, _ string, _ cdriver.Profile, _ io.Writer) (nativeWorkspaceTestBuild, int) {
			index := prepared
			prepared++
			return func(stderr io.Writer) (nativeWorkspaceTestArtifact, int) {
				if index == 0 {
					<-entered
					fmt.Fprint(stderr, "first build failed")
					return nativeWorkspaceTestArtifact{}, ExitBuildFailure
				}
				close(entered)
				<-release
				fmt.Fprint(stderr, "later diagnostic")
				return nativeWorkspaceTestArtifact{}, ExitSuccess
			}, ExitSuccess
		}
		var stderr bytes.Buffer
		done := make(chan struct{})
		go func() {
			defer close(done)
			outcomes, code := runNativeWorkspaceTestsWith(nativeTestFixtureProgram(), tests, cdriver.LowProfile, nativeWorkspaceTestInput{}, nil, time.Minute, false, false, 2, &stderr, prepare,
				func(nativeWorkspaceTestArtifact, io.Reader, []string, time.Duration) (nativeWorkspaceTestOutcome, error) {
					t.Error("execution started after a build failure")
					return nativeWorkspaceTestOutcome{}, nil
				})
			if code != ExitBuildFailure || outcomes != nil || stderr.String() != "first build failed" || prepared != 2 {
				t.Errorf("code=%d outcomes=%v stderr=%q prepared=%d", code, outcomes, &stderr, prepared)
			}
		}()
		synctest.Wait()
		select {
		case <-done:
			t.Error("returned before the later build finished")
		default:
		}
		close(release)
		<-done
	})
}
