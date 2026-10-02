package mir

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestOperandJSONPreservesRawBytes(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	for _, value := range []Operand{
		{Kind: OperandByteString, Type: hir.SharedBorrow(hir.TypeByteSlice), StaticData: string(all)},
		{Kind: OperandByteString, Type: hir.SharedBorrow(hir.TypeByteSlice)},
		{Kind: OperandCString, Type: hir.SharedBorrow(hir.TypeCStr), StaticData: "\xff\x80\x00"},
		{Kind: OperandString, Type: hir.SharedBorrow(hir.TypeStr), StaticData: "한글\x00é"},
		{Kind: OperandI64, I64: math.MinInt64},
		{Kind: OperandI32, I32: -1},
		{Kind: OperandUSize, U64: math.MaxUint64},
		{Kind: OperandLocal, Local: 9},
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte(`"StaticData":`)) || !bytes.Contains(encoded, []byte(`"StaticDataBase64":"`+base64.StdEncoding.EncodeToString([]byte(value.StaticData))+`"`)) {
			t.Fatalf("raw bytes are not encoded explicitly: %s", encoded)
		}
		var decoded Operand
		if err := json.Unmarshal(encoded, &decoded); err != nil || !reflect.DeepEqual(value, decoded) {
			t.Fatalf("round trip: %v got %+v want %+v", err, decoded, value)
		}
	}
}

func TestOperandJSONRejectsMalformedOrLegacyStaticData(t *testing.T) {
	for _, data := range []string{
		`{"StaticData":"\ufffd"}`,
		`{}`,
		`{"StaticDataBase64":null}`,
		`{"StaticDataBase64":3}`,
		`{"StaticDataBase64":"?"}`,
		`{"StaticDataBase64":"/w"}`,
		`{"StaticDataBase64":"/x=="}`,
		`{"StaticDataBase64":"/w==\n"}`,
		`{"StaticDataBase64":"/w==","unknown":true}`,
	} {
		original := Operand{Kind: OperandI32, I32: 7}
		decoded := original
		if err := json.Unmarshal([]byte(data), &decoded); err == nil {
			t.Fatalf("accepted %s", data)
		}
		if decoded != original {
			t.Fatalf("failed decode changed its destination for %s", data)
		}
	}
}
