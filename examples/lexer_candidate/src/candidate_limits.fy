module candidate_limits;

// Each joined owner stays bounded independently of the complete input set.
// Byte limits do not raise the separate binding, depth or work limits.
pub fn usize source_bytes(){return 65536;}
pub fn usize program_bytes(){return 1048576;}
// Workspace/check/lock integration makes the complete current dispatcher
// about 2.5 MB. This bounds aggregate retained bytes only; each joined unit,
// source file and analysis pass retains its independent admission cap.
pub fn usize source_set_bytes(){return 4194304;}
// The complete dispatcher imports 285 source owners. This bounds file/module
// identities only; source-set bytes and each checked unit keep their own caps.
pub fn usize source_files(){return 512;}
pub fn usize manifest_bytes(){return 65536;}

// The source-identified dispatcher exceeds 4096 original declarations.
// This catalog bound is separate from executable callables, types and frames.
pub fn usize project_declarations(){return 8192;}

// Imported layouts combine fields from multiple original source owners.
// The source parser and joined AST retain their separate 1024-field caps.
pub fn usize metadata_fields(){return 2048;}

// The source-identified dispatcher exceeds 4096 rebound fields. This bounds
// their combined table; each independently checked unit retains its 2048 cap.
pub fn usize project_fields(){return 8192;}

// Joined bodies include original modules plus authenticated SDK specializations.
// Per-file parser limits remain separate; all downstream consumers share this cap.
pub fn usize program_expressions(){return 16384;}

// Rebound project functions span many independently bounded source units.
// The current 285-source dispatcher reaches the previous 4096-frame cap.
// Files, types, callables, bytes and work keep their independent bounds.
pub fn usize project_frames(){return 8192;}

// Callable prototypes combine separately bounded unit interfaces.
pub fn usize project_callables(){return 4096;}

// Whole-project rebinding combines independently capped producer histories.
// The complete dispatcher exceeds 4 MiB during owned IR revalidation. Each
// source unit and body retains its separate 1 MiB budget; only the sum grows.
pub fn usize project_work(){return 8388608;}
pub fn bool project_charge(ref mut usize work,usize amount){if(*work>project_work() || amount>project_work()-*work){return false;};*work+=amount;return true;}
