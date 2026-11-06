module project_atlas_borrow_sites;
import module_unit_checked_bodies as checking;
import module_unit_borrow_sites as sites;
import std::vec as v;

// Consume the same successful type/move/origin/cleanup/Result owner used by IR
// lowering. No caller-supplied syntax scan or unverified loan list is accepted.
pub fn Result<v::Vec<sites::Site>,u32> collect(ref checking::Plan p,usize frame,usize maximum,ref mut usize work){return sites::collect(checking::checked_borrows(p),frame,maximum,work);}
pub fn Result<sites::Collection,u32> collect_all(ref checking::Plan p,usize maximum,ref mut usize work){return sites::collect_all(checking::checked_borrows(p),maximum,work);}
