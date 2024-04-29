package log

import (
	"log/slog"
	"reflect"
	"testing"
)

func Test_normalizeGroupsAndAttributes(t *testing.T) {
	type args struct {
		groupOfAttrs []groupOrAttrs
		numAttrs     int
	}
	tests := []struct {
		name string
		args args
		want []groupOrAttrs
	}{
		{
			name: "attrs with group",
			args: args{
				groupOfAttrs: []groupOrAttrs{{
					group: "123",
					attrs: []slog.Attr{{
						Key:   "key",
						Value: slog.IntValue(1),
					}},
				}},
				numAttrs: 1,
			},
			want: []groupOrAttrs{
				{
					group: "123",
					attrs: []slog.Attr{{
						Key:   "key",
						Value: slog.IntValue(1),
					}},
				},
			},
		},
		{
			name: "no attrs with group",
			args: args{
				groupOfAttrs: []groupOrAttrs{{
					group: "123",
					attrs: []slog.Attr{},
				}},
				numAttrs: 0,
			},
			want: []groupOrAttrs{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeGroupsAndAttributes(tt.args.groupOfAttrs, tt.args.numAttrs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalizeGroupsAndAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}
