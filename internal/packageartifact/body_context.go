package packageartifact

// bodyContext belongs to one batch of body conversions. Its descriptor and
// byte limit stay fixed while bodies share the validated canonical context ID
// and, during batch reassembly, one package-coordinate resolver.
// Standalone conversions use their own context, never a process-wide cache.
type bodyContext struct {
	descriptor ContextDescriptor
	maxBytes   uint64
	id         ContextID
	ready      bool
	packages   packageResolver
}

func (context *bodyContext) contextID() (ContextID, error) {
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
