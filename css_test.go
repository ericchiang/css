package css

import (
	"reflect"
	"testing"
)

func TestClassSelector(t *testing.T) {
	tests := []struct {
		s       string
		want    *classSelector
		wantErr bool
	}{
		{".foo", &classSelector{"foo"}, false},
		{".bar()", nil, true},
		{"foo", nil, true},
	}
	for _, test := range tests {
		p := newParser(test.s)
		got, err := p.classSelector()
		if (err != nil) != test.wantErr {
			t.Errorf("parsing %q got err=%v, want err=%t", test.s, err, test.wantErr)
			continue
		}
		if test.wantErr {
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("parsing %q got %v, want %v", test.s, got, test.want)
		}
	}
}
