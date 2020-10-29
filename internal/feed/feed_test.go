package feed

import (
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func md5ToUUID(md5 string) uuid.UUID {
	u, _ := uuid.Parse(md5)
	return u
}

func Test_uuidFromString(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want uuid.UUID
	}{
		{
			"hello world md5 to UUID",
			args{
				"hello world",
			},
			md5ToUUID("5EB63BBBE01EEED093CB22BB8F5ACDC3"),
		},
		{
			"good bye md5 to UUID",
			args{
				"good bye",
			},
			md5ToUUID("2FF613942F0C135C7007802A19494AD0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UUIDFromString(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UUIDFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}