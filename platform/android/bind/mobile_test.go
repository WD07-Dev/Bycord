package mobile

import "testing"

func TestValidExcluded(t *testing.T) {
	if !ValidExcluded("br,US") {
		t.Fatal("valid country codes were rejected")
	}
	if ValidExcluded("invalid,1A") {
		t.Fatal("invalid country codes were accepted")
	}
}
