package hir

// TypeCCallbackI32UserData is the first bounded callback signature. It is only
// source-admitted as the penultimate foreign parameter, before explicit user data.
var TypeCCallbackI32UserData = builtin("unsafe extern C fn i32(i32, *mut void)")

// ValidBoundedCallbackParameters checks the closed callback/user-data pair.
// A declaration without a callback retains its existing parameter contract.
func ValidBoundedCallbackParameters(parameters []Type) bool {
	for index, typ := range parameters {
		if typ == TypeCCallbackI32UserData && (index != len(parameters)-2 || parameters[index+1] != RawPointer(TypeVoid, true)) {
			return false
		}
	}
	return true
}
