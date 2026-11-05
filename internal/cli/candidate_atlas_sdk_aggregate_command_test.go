package cli

import "testing"

// Keep both client aggregate kinds and a same-name Vec beside the authenticated
// SDK owner. The SDK stores i32; Flag<bool> has an actual cleanup body.
const atlasSDKUserAggregateCommandSource = `module main;import std::vec as v;
pub struct Vec<T>{T value;}
pub resource Flag<T>{T value;cleanup{assert(this.value);}}
fn i32 leaf(){let flag=Flag<bool>{.value=true};let cell=Vec<i32>{.value=42};var values=v::empty<i32>();assert(v::push<i32>(ref mut values,cell.value).is_ok);return *v::at<i32>(ref values,0);}
fn i32 main(){return leaf();}`

func TestCandidateAtlasSDKUserAggregateCommand(t *testing.T) {
	main := `module main;import project_atlas_cli as atlas;@effects(filesystem,console,environment)fn i32 main(){return atlas::arguments();}`
	candidateAtlasProbeSize(t, "project_atlas_cli", main, 2<<20, verifyCandidateAtlasSDKUserAggregateCommand)
}

func verifyCandidateAtlasSDKUserAggregateCommand(t *testing.T, image string) {
	t.Helper()
	verifyCandidateAtlasSDKCommandSource(t, image, atlasSDKUserAggregateCommandSource, "std::vec", "at", "Vec", "Flag")
}
