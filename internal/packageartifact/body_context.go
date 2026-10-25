package packageartifact

// bodyProjectionContext belongs to one projection call. Its descriptor and
// byte limit stay fixed while bodies share the validated canonical context ID.
// Standalone body projection uses its own context, never a process-wide cache.
type bodyProjectionContext struct {
	descriptor ContextDescriptor
	maxBytes   uint64
	id         ContextID
	ready      bool
}

func (context *bodyProjectionContext) contextID() (ContextID, error) {
	if context.ready {
		return context.id, nil
	}
	id, err := ComputeContextID(context.descriptor, context.maxBytes)
	if err != nil {
		return ContextID{}, err
	}
	context.id, context.ready = id, true
	return id, nil
}
