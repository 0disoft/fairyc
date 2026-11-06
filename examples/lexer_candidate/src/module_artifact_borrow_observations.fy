module module_artifact_borrow_observations;
import module_borrow_observations as observations;
import module_unit_borrow_sites as sites;
import module_program_artifact_wire as budget;
import std::vec as v;

// Observation operations retain their own bounded history. Fold measured work
// into the artifact's existing 1Gi-work history without reinterpreting its
// accumulated value as a 16Mi-work observation counter, including on failure.
pub fn Result<bool,u32> initialize(usize frames,ref mut v::Vec<sites::Range> ranges,ref mut v::Vec<u8> seen,ref mut usize work){
    var usize measured=0;let result=observations::initialize(frames,ranges,seen,ref mut measured);if(!budget::charge(work,measured)){discard result;return .err(3);}return result;
}
pub fn Result<bool,u32> add_sites(ref mut v::Vec<sites::Site> rows,ref mut v::Vec<sites::Range> ranges,ref mut v::Vec<u8> seen,usize frame,ref sites::Collection source,usize source_frame,ref mut usize work){
    var usize measured=0;let result=observations::add_sites(rows,ranges,seen,frame,source,source_frame,ref mut measured);if(!budget::charge(work,measured)){discard result;return .err(3);}return result;
}
pub fn Result<observations::Table,u32> seal(v::Vec<sites::Site> rows,v::Vec<sites::Range> ranges,ref v::Vec<u8> seen,ref mut usize work){
    var usize measured=0;let result=observations::seal(rows,ranges,seen,ref mut measured);if(!budget::charge(work,measured)){discard result;return .err(3);}return result;
}
pub fn Result<observations::Table,u32> copy(ref observations::Table p,ref mut usize work){
    var usize measured=0;let result=observations::copy(p,ref mut measured);if(!budget::charge(work,measured)){discard result;return .err(3);}return result;
}
