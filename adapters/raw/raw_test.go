package raw

import (
	"testing"
)

func TestToJSON(t *testing.T) {

	toJSON := funcs["toJSON"].(func(interface{}) string)

	type args struct {
		value interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"json_string", args{`{"my":"json"}`}, `{"my":"json"}`},
		{"json_string1", args{`{"my":"json", "dry":"ice"}`}, `{"my":"json", "dry":"ice"}`},
		{"nonjson_string1", args{`{"my":json"}`}, `"{\"my\":json\"}"`},
		{"nonjson_string2", args{`Alice and Bob`}, `"Alice and Bob"`},
		{"other", args{struct{ Name string }{"Megan"}}, `{"Name":"Megan"}`},
		{"json_byte", args{[]byte(`{"my":"json"}`)}, `{"my":"json"}`},
		{"json_byte1", args{[]byte(`{"my":"json", "dry":"ice"}`)}, `{"my":"json", "dry":"ice"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toJSON(tt.args.value); got != tt.want {
				t.Errorf("toJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
