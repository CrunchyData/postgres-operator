package kubeapi

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

func TestSanitizeLabelValue(t *testing.T) {
	for _, tt := range []struct{ input, expected string }{
		{"", ""},
		{"a-very-fine-label", "a-very-fine-label"},
		{"TestSomething/With_Underscore/#01", "TestSomething-With_Underscore--01"},
		{strings.Repeat("abc456ghi0", 8), "abc456ghi0abc456ghi0abc456ghi0abc456ghi0abc456ghi0abc456ghi0abc"},
	} {
		if errors := validation.IsValidLabelValue(tt.expected); len(errors) != 0 {
			t.Fatalf("bug in test: %q is invalid: %v", tt.expected, errors)
		}
		if actual := SanitizeLabelValue(tt.input); tt.expected != actual {
			t.Errorf("expected %q to be %q, got %q", tt.input, tt.expected, actual)
		}
	}
}
