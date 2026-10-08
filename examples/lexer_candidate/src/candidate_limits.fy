module candidate_limits;

// File admission and owned-program admission use the same source-set budget.
// Byte limits do not raise the separate AST, binding, depth or work limits.
pub fn usize source_bytes(){return 65536;}
pub fn usize program_bytes(){return 1048576;}
pub fn usize source_files(){return 256;}
pub fn usize manifest_bytes(){return 65536;}

// Joined bodies include original modules plus authenticated SDK specializations.
// Per-file parser limits remain separate; all downstream consumers share this cap.
pub fn usize program_expressions(){return 16384;}
