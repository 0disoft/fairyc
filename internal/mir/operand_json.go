package mir

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// StaticData is a comparable raw byte string in memory, not JSON text. Share
// one lossless representation across whole-program, body and package codecs.
type operandJSON struct {
	Kind             OperandKind
	Local            LocalID
	I32              int32
	I64              int64 `json:",omitempty"`
	U64              uint64
	Type             hir.Type
	StaticDataBase64 *string
}

func (value Operand) MarshalJSON() ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(value.StaticData))
	return json.Marshal(operandJSON{
		Kind: value.Kind, Local: value.Local, I32: value.I32, I64: value.I64,
		U64: value.U64, Type: value.Type, StaticDataBase64: &encoded,
	})
}

func (value *Operand) UnmarshalJSON(data []byte) error {
	var encoded operandJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&encoded); err != nil {
		return fmt.Errorf("decode MIR operand: %w", err)
	}
	if encoded.StaticDataBase64 == nil {
		return fmt.Errorf("MIR operand requires StaticDataBase64")
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(*encoded.StaticDataBase64)
	if err != nil {
		return fmt.Errorf("decode MIR operand static bytes: %w", err)
	}
	if base64.StdEncoding.EncodeToString(raw) != *encoded.StaticDataBase64 {
		return fmt.Errorf("MIR operand static bytes are not canonical Base64")
	}
	*value = Operand{
		Kind: encoded.Kind, Local: encoded.Local, I32: encoded.I32, I64: encoded.I64,
		U64: encoded.U64, Type: encoded.Type, StaticData: string(raw),
	}
	return nil
}
