module resource_properties;
import resource_identity as identity;
import tagged_encoding as tagged;

// Static ownership follows every potentially active tagged payload. Canonical
// encoding bounds recursion; ordinary resources are move-only even when empty.
pub fn bool needs_drop(u32 type) {
    if(identity::valid(type)){return true;}
    if(tagged::option(type)){return needs_drop(tagged::inner(type));}
    return tagged::result(type) && (needs_drop(tagged::success(type)) || needs_drop(tagged::failure(type)));
}
