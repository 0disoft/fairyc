module module_project_borrow_observations;
import module_borrow_observations as observations;
import module_project_identified_compile as units;
import module_identified_ir as identified;
import module_project_frames as frames;
import std::vec as v;
import module_unit_borrow_sites as sites;

// Follow exactly the linker's frame map. Aliases must agree, including empty
// observations; selecting a representative must never conceal disagreement.
pub fn Result<observations::Table,u32> build(ref units::Plan input,ref frames::Plan map,ref mut usize work){
    var rows=v::empty<sites::Site>();var ranges=v::empty<sites::Range>();var seen=v::empty<u8>();discard observations::initialize(frames::count(map),ref mut ranges,ref mut seen,work)?;var usize unit=0;
    while(unit<units::count(input)){let p=units::at(input,unit);if(!identified::has_borrows(p)){return .err(9);}let source=identified::borrows(p);
        if(observations::count(source)!=identified::frame_count(p)){return .err(9);}var usize i=1;
        while(i<=identified::frame_count(p)){discard observations::add_table(ref mut rows,ref mut ranges,ref mut seen,frames::mapped(map,unit,i),source,i,work)?;i+=1;}unit+=1;
    }return observations::seal(rows,ranges,ref seen,work);
}
