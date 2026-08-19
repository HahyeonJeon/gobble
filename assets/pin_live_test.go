//go:build live

package assets

import (
	"testing"
)

func TestPinFetchOfficialPins(t *testing.T) {
	wgs := PinWGSTest1FASTQ
	sars := pinSARSCoV2R1
	var dests []string
	for _, pin := range []Pin{wgs, sars} {
		dest, err := FetchPin(pin)
		if err != nil {
			t.Fatalf("download %s: %v", pin.URL, err)
		}
		if dest != pin.CachePath() {
			t.Fatalf("FetchPin(%s) = %q, want %q", pin.Name, dest, pin.CachePath())
		}
		if err := pin.Check(dest); err != nil {
			t.Fatalf("Check(%s) error = %v, want nil", dest, err)
		}
		dests = append(dests, dest)
	}
	if dests[0] == dests[1] {
		t.Fatalf("FetchPin dest = %q for both pins", dests[0])
	}
}
