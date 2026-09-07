package atlasmap

func (store *MapStore) transportVersion() (schema, encoding, logical string) {
	if store.v2 {
		return TransportSchemaV2, TransportEncodingV2, LogicalSchemaV2
	}
	return TransportSchema, TransportEncoding, LogicalSchema
}
